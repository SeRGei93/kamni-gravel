package telegram

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// Bot представляет Telegram бота
type Bot struct {
	api            *telegrambot.Bot
	debug          bool
	miniappURL     string
	sessionManager *session.Manager

	// Repositories
	userRepo            repository.UserRepository
	eventRepo           repository.EventRepository
	participantRepo     repository.ParticipantRepository
	resultRepo          repository.ResultRepository
	giftRepo            repository.GiftRepository
	criteriaRepo        repository.CriteriaRepository
	giftCriteriaRepo    repository.GiftCriteriaRepository
	prizeAssignmentRepo repository.PrizeAssignmentRepository
	userBlacklistRepo   repository.UserBlacklistRepository

	// Command handlers
	registerParticipantHandler *command.RegisterParticipantHandler
	addGiftHandler             *command.AddGiftHandler
	submitResultHandler        *command.SubmitResultHandler
	assignPrizeHandler         *command.AssignPrizeHandler

	// Query handlers
	getParticipantsHandler   *query.GetParticipantsHandler
	getGiftsHandler          *query.GetGiftsHandler
	getEventsHandler         *query.GetEventsHandler
	getCriteriaHandler       *query.GetCriteriaHandler
	getStatsHandler          *query.GetStatsHandler
	isUserBlacklistedHandler *query.IsUserBlacklistedHandler
}

// Config представляет конфигурацию бота
type Config struct {
	Token          string
	Debug          bool
	MiniappURL     string
	SessionTimeout time.Duration
}

// NewBot создаёт новый экземпляр бота
func NewBot(
	cfg Config,
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	resultRepo repository.ResultRepository,
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
	giftCriteriaRepo repository.GiftCriteriaRepository,
	prizeAssignmentRepo repository.PrizeAssignmentRepository,
	userBlacklistRepo repository.UserBlacklistRepository,
) (*Bot, error) {
	miniappURL := validateMiniappURL(cfg.MiniappURL)

	// Создаём session manager
	sessionManager := session.NewManager(cfg.SessionTimeout)

	// Создаём command handlers
	registerParticipantHandler := command.NewRegisterParticipantHandler(
		userRepo,
		eventRepo,
		participantRepo,
		userBlacklistRepo,
	)

	addGiftHandler := command.NewAddGiftHandler(
		userRepo,
		eventRepo,
		giftRepo,
		userBlacklistRepo,
	)

	submitResultHandler := command.NewSubmitResultHandler(
		participantRepo,
		eventRepo,
		resultRepo,
	)

	assignPrizeHandler := command.NewAssignPrizeHandler(
		participantRepo,
		giftRepo,
		prizeAssignmentRepo,
	)

	// Создаём query handlers
	getParticipantsHandler := query.NewGetParticipantsHandler(participantRepo)
	getGiftsHandler := query.NewGetGiftsHandler(giftRepo, criteriaRepo)
	getEventsHandler := query.NewGetEventsHandler(eventRepo)
	getCriteriaHandler := query.NewGetCriteriaHandler(criteriaRepo)
	isUserBlacklistedHandler := query.NewIsUserBlacklistedHandler(userBlacklistRepo)
	getStatsHandler := query.NewGetStatsHandler(
		eventRepo,
		participantRepo,
		giftRepo,
		resultRepo,
		criteriaRepo,
	)

	telegramBot := &Bot{
		debug:                      cfg.Debug,
		miniappURL:                 miniappURL,
		sessionManager:             sessionManager,
		userRepo:                   userRepo,
		eventRepo:                  eventRepo,
		participantRepo:            participantRepo,
		resultRepo:                 resultRepo,
		giftRepo:                   giftRepo,
		criteriaRepo:               criteriaRepo,
		giftCriteriaRepo:           giftCriteriaRepo,
		prizeAssignmentRepo:        prizeAssignmentRepo,
		userBlacklistRepo:          userBlacklistRepo,
		registerParticipantHandler: registerParticipantHandler,
		addGiftHandler:             addGiftHandler,
		submitResultHandler:        submitResultHandler,
		assignPrizeHandler:         assignPrizeHandler,
		getParticipantsHandler:     getParticipantsHandler,
		getGiftsHandler:            getGiftsHandler,
		getEventsHandler:           getEventsHandler,
		getCriteriaHandler:         getCriteriaHandler,
		getStatsHandler:            getStatsHandler,
		isUserBlacklistedHandler:   isUserBlacklistedHandler,
	}

	opts := []telegrambot.Option{
		telegrambot.WithDefaultHandler(telegramBot.handleUpdate),
	}
	if cfg.Debug {
		opts = append(opts, telegrambot.WithDebug())
	}

	api, err := telegrambot.New(cfg.Token, opts...)
	if err != nil {
		log.Printf("Failed to initialize Telegram bot API: %v", err)
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	telegramBot.api = api

	log.Printf("Telegram bot initialized successfully: bot_id=%d debug=%t", api.ID(), cfg.Debug)

	return telegramBot, nil
}

func validateMiniappURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		log.Printf("INFO Telegram miniapp URL omitted; WebApp button disabled")
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("WARN Telegram miniapp URL ignored: reason=malformed_url")
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		log.Printf("WARN Telegram miniapp URL ignored: reason=unsupported_scheme scheme=%q", parsed.Scheme)
		return ""
	}
	if parsed.Host == "" {
		log.Printf("WARN Telegram miniapp URL ignored: reason=missing_host scheme=%q", parsed.Scheme)
		return ""
	}

	log.Printf("INFO Telegram miniapp URL configured: scheme=%s host=%s", parsed.Scheme, parsed.Host)
	return rawURL
}

// Start запускает бота
func (b *Bot) Start(ctx context.Context) error {
	log.Println("Telegram bot started, waiting for updates...")
	b.api.Start(ctx)
	log.Println("Telegram bot stopped")

	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

// handleUpdate обрабатывает входящее обновление
func (b *Bot) handleUpdate(ctx context.Context, _ *telegrambot.Bot, update *models.Update) {
	if update == nil {
		b.logDebug("Unsupported Telegram update: nil update")
		return
	}
	if b.shouldSilentlyIgnoreBlacklistedUpdate(ctx, update) {
		return
	}

	// Обработка команд
	if update.Message != nil && messageCommand(update.Message) != "" {
		b.handleCommand(ctx, update.Message)
		return
	}

	// Обработка callback-запросов (inline кнопки)
	if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery)
		return
	}

	// Обработка обычных сообщений
	if update.Message != nil {
		b.handleMessage(ctx, update.Message)
		return
	}

	b.logDebug("Unsupported Telegram update kind: update_id=%d", update.ID)
}

func (b *Bot) shouldSilentlyIgnoreBlacklistedUpdate(ctx context.Context, update *models.Update) bool {
	telegramUserID, updateKind, ok := telegramUpdateSender(update)
	if !ok {
		return false
	}

	isBlacklisted, err := b.isUserBlacklistedHandler.Handle(ctx, query.IsUserBlacklistedQuery{
		TelegramUserID: telegramUserID,
	})
	if err != nil {
		log.Printf("ERROR Telegram blacklist guard failed: operation=blacklist_guard telegram_user_id=%d update_kind=%s error=%v", telegramUserID, updateKind, err)
		return false
	}
	if !isBlacklisted {
		return false
	}

	log.Printf("INFO Telegram update silently ignored: operation=blacklist_guard telegram_user_id=%d update_kind=%s", telegramUserID, updateKind)
	return true
}

// SendMessage отправляет текстовое сообщение
func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string) (*models.Message, error) {
	msg, err := b.api.SendMessage(ctx, &telegrambot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		log.Printf("Telegram API call failed: operation=send_message chat_id=%d error=%v", chatID, err)
		return nil, err
	}

	return msg, nil
}

// SendMessageWithKeyboard отправляет сообщение с inline клавиатурой
func (b *Bot) SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard models.InlineKeyboardMarkup) (*models.Message, error) {
	msg, err := b.api.SendMessage(ctx, &telegrambot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: keyboard,
	})
	if err != nil {
		log.Printf("Telegram API call failed: operation=send_message_with_keyboard chat_id=%d error=%v", chatID, err)
		return nil, err
	}

	return msg, nil
}

// EditMessage редактирует существующее сообщение
func (b *Bot) EditMessage(ctx context.Context, chatID int64, messageID int, text string) error {
	_, err := b.api.EditMessageText(ctx, &telegrambot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
	})
	if err != nil {
		log.Printf("Telegram API call failed: operation=edit_message chat_id=%d message_id=%d error=%v", chatID, messageID, err)
		return err
	}

	return nil
}

// AnswerCallback отвечает на callback query
func (b *Bot) AnswerCallback(ctx context.Context, callbackID string, text string) error {
	_, err := b.api.AnswerCallbackQuery(ctx, &telegrambot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
	if err != nil {
		log.Printf("Telegram API call failed: operation=answer_callback callback_id=%s error=%v", callbackID, err)
		return err
	}

	return nil
}

// DeleteMessage удаляет сообщение
func (b *Bot) DeleteMessage(ctx context.Context, chatID int64, messageID int) error {
	_, err := b.api.DeleteMessage(ctx, &telegrambot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
	if err != nil {
		log.Printf("Telegram API call failed: operation=delete_message chat_id=%d message_id=%d error=%v", chatID, messageID, err)
		return err
	}

	return nil
}

func (b *Bot) logDebug(format string, args ...any) {
	if !b.debug {
		return
	}

	log.Printf(format, args...)
}
