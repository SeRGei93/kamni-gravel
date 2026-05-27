package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// Bot представляет Telegram бота
type Bot struct {
	api              telegramAPI
	debug            bool
	botUsername      string
	miniappURL       string
	adminChatID      int64
	publicChatID     int64
	adminChatLogOnce sync.Once
	sessionManager   *session.Manager

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
	withdrawParticipantHandler *command.WithdrawParticipantHandler
	assignPrizeHandler         *command.AssignPrizeHandler

	// Query handlers
	getParticipantsHandler   *query.GetParticipantsHandler
	getGiftsHandler          *query.GetGiftsHandler
	getEventsHandler         *query.GetEventsHandler
	getCriteriaHandler       *query.GetCriteriaHandler
	getStatsHandler          *query.GetStatsHandler
	isUserBlacklistedHandler *query.IsUserBlacklistedHandler
}

type telegramAPI interface {
	Start(ctx context.Context)
	SendMessage(ctx context.Context, params *telegrambot.SendMessageParams) (*models.Message, error)
	CopyMessage(ctx context.Context, params *telegrambot.CopyMessageParams) (*models.MessageID, error)
	ForwardMessage(ctx context.Context, params *telegrambot.ForwardMessageParams) (*models.Message, error)
	EditMessageText(ctx context.Context, params *telegrambot.EditMessageTextParams) (*models.Message, error)
	AnswerCallbackQuery(ctx context.Context, params *telegrambot.AnswerCallbackQueryParams) (bool, error)
	DeleteMessage(ctx context.Context, params *telegrambot.DeleteMessageParams) (bool, error)
}

// Config представляет конфигурацию бота
type Config struct {
	Token          string
	AdminChatID    int64
	PublicChatID   int64
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

	withdrawParticipantHandler := command.NewWithdrawParticipantHandler(participantRepo)

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
		adminChatID:                cfg.AdminChatID,
		publicChatID:               cfg.PublicChatID,
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
		withdrawParticipantHandler: withdrawParticipantHandler,
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

	botUsername, err := lookupBotUsername(api)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("WARN Could not resolve Telegram bot username: reason=timeout")
		} else {
			log.Printf("WARN Could not resolve Telegram bot username: reason=%v", err)
		}
	} else {
		telegramBot.botUsername = botUsername
		log.Printf("INFO Telegram bot username resolved from API")
	}

	telegramBot.api = api
	telegramBot.logChatMode("public", cfg.PublicChatID)
	telegramBot.logChatMode("admin", cfg.AdminChatID)
	log.Printf("Telegram bot initialized successfully: debug=%t", cfg.Debug)

	return telegramBot, nil
}

func (b *Bot) logChatMode(name string, chatID int64) {
	if chatID == 0 {
		log.Printf("INFO Telegram %s chat mode disabled", name)
		return
	}

	log.Printf("INFO Telegram %s chat mode enabled", name)
}

func lookupBotUsername(api *telegrambot.Bot) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	me, err := api.GetMe(ctx)
	if err != nil {
		return "", err
	}
	if me == nil {
		return "", fmt.Errorf("telegram bot me is nil")
	}

	return strings.TrimSpace(me.Username), nil
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
	if b.shouldIgnoreNonPrivateUpdate(update) {
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

func (b *Bot) shouldIgnoreNonPrivateUpdate(update *models.Update) bool {
	if update == nil {
		return false
	}

	if update.Message != nil {
		if isPrivateTelegramChat(update.Message.Chat) {
			return false
		}
		b.logDebug("Telegram update ignored outside private chat: update_id=%d chat=%s kind=%s", update.ID, b.chatLogMarker(update.Message.Chat.ID), messageUpdateKind(update.Message))
		return true
	}

	if update.CallbackQuery != nil && update.CallbackQuery.Message.Message != nil {
		chat := update.CallbackQuery.Message.Message.Chat
		if isPrivateTelegramChat(chat) {
			return false
		}
		b.logDebug("Telegram callback update ignored outside private chat: update_id=%d chat=%s", update.ID, b.chatLogMarker(chat.ID))
		return true
	}

	if update.CallbackQuery != nil && update.CallbackQuery.Message.InaccessibleMessage != nil {
		chat := update.CallbackQuery.Message.InaccessibleMessage.Chat
		if isPrivateTelegramChat(chat) {
			return false
		}
		b.logDebug("Telegram inaccessible callback update ignored outside private chat: update_id=%d chat=%s", update.ID, b.chatLogMarker(chat.ID))
		return true
	}

	return false
}

func (b *Bot) shouldSilentlyIgnoreBlacklistedUpdate(ctx context.Context, update *models.Update) bool {
	if update != nil && update.Message != nil && len(update.Message.NewChatMembers) > 0 {
		return false
	}

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
		log.Printf("Telegram API call failed: operation=send_message chat=%s error=%v", b.chatLogMarker(chatID), err)
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
		log.Printf("Telegram API call failed: operation=send_message_with_keyboard chat=%s error=%v", b.chatLogMarker(chatID), err)
		return nil, err
	}

	return msg, nil
}

func (b *Bot) notifyAdminAboutGift(ctx context.Context, gift *entity.Gift, sourceRefs []giftSourceRef) error {
	if b == nil || gift == nil {
		return nil
	}
	if b.api == nil {
		return fmt.Errorf("telegram api is nil")
	}

	if b.adminChatID == 0 {
		b.adminChatLogOnce.Do(func() {
			log.Printf("WARN Admin gift notification skipped: reason=admin_chat_disabled")
		})
		return nil
	}

	var sourceErr error
	if len(sourceRefs) == 0 {
		log.Printf("WARN Admin gift notification uses summary only: gift_id=%d event_id=%d user_id=%d reason=no_source_refs", gift.ID, gift.EventID, gift.UserID)
	} else if err := b.copyOrForwardGiftSources(ctx, sourceRefs); err != nil {
		sourceErr = err
		log.Printf(
			"WARN Admin gift source forwarding failed: gift_id=%d event_id=%d user_id=%d error=%v",
			gift.ID,
			gift.EventID,
			gift.UserID,
			err,
		)
	}

	if summaryErr := b.sendAdminGiftSummary(ctx, gift); summaryErr != nil {
		log.Printf(
			"ERROR Admin gift summary notification failed: gift_id=%d event_id=%d user_id=%d chat=admin error=%v",
			gift.ID,
			gift.EventID,
			gift.UserID,
			summaryErr,
		)
		if sourceErr != nil {
			return fmt.Errorf("notify admin about gift: copy=%w summary=%w", sourceErr, summaryErr)
		}
		return summaryErr
	}

	if sourceErr != nil {
		log.Printf("INFO Admin gift summary notification sent after source forwarding failure: gift_id=%d event_id=%d user_id=%d chat=admin", gift.ID, gift.EventID, gift.UserID)
		return nil
	}

	log.Printf("INFO Admin gift notification sent: gift_id=%d event_id=%d user_id=%d source_message_count=%d chat=admin", gift.ID, gift.EventID, gift.UserID, len(sourceRefs))
	return nil
}

func (b *Bot) copyOrForwardGiftSources(ctx context.Context, sourceRefs []giftSourceRef) error {
	if b == nil || len(sourceRefs) == 0 {
		return nil
	}
	if b.api == nil {
		return fmt.Errorf("telegram api is nil")
	}

	var lastErr error
	allDelivered := true

	for _, sourceRef := range sourceRefs {
		if sourceRef.ChatID == 0 || sourceRef.MessageID <= 0 {
			log.Printf("WARN Gift source ref skipped: reason=invalid_message_identifiers kind=%s", sourceRef.UpdateKind)
			allDelivered = false
			continue
		}

		copyParams := &telegrambot.CopyMessageParams{
			ChatID:     b.adminChatID,
			FromChatID: sourceRef.ChatID,
			MessageID:  sourceRef.MessageID,
		}

		if _, err := b.api.CopyMessage(ctx, copyParams); err == nil {
			b.logDebug("Gift source forwarded by copy: kind=%s message_id=%d", sourceRef.UpdateKind, sourceRef.MessageID)
			continue
		} else {
			forwardParams := &telegrambot.ForwardMessageParams{
				ChatID:     b.adminChatID,
				FromChatID: sourceRef.ChatID,
				MessageID:  sourceRef.MessageID,
			}

			if _, forwardErr := b.api.ForwardMessage(ctx, forwardParams); forwardErr != nil {
				log.Printf(
					"WARN Gift source copy and forward failed: kind=%s copy_error=%v forward_error=%v",
					sourceRef.UpdateKind,
					err,
					forwardErr,
				)
				lastErr = forwardErr
				allDelivered = false
			}
		}
	}

	if allDelivered {
		return nil
	}

	if lastErr != nil {
		return lastErr
	}

	return fmt.Errorf("some gift source messages not delivered to admin")
}

func (b *Bot) sendAdminGiftSummary(ctx context.Context, gift *entity.Gift) error {
	text := b.adminGiftSummaryText(gift)
	_, err := b.SendMessage(ctx, b.adminChatID, text)
	if err != nil {
		return fmt.Errorf("send admin gift summary: %w", err)
	}

	return nil
}

func (b *Bot) adminGiftSummaryText(gift *entity.Gift) string {
	if gift == nil {
		return "Новый подарок на модерацию: данные недоступны."
	}

	return fmt.Sprintf(
		"Новый подарок на проверку\n\nID подарка: %d\nID события: %d\nПользователь: %d\nФильтр пола: %s\nФильтр велосипеда: %s\nСтатус: %s\nФото: %d",
		gift.ID,
		gift.EventID,
		gift.UserID,
		gift.GenderFilter,
		gift.BikeTypeFilter,
		gift.ReviewStatus,
		len(gift.Attachments),
	)
}

// EditMessage редактирует существующее сообщение
func (b *Bot) EditMessage(ctx context.Context, chatID int64, messageID int, text string) error {
	_, err := b.api.EditMessageText(ctx, &telegrambot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
	})
	if err != nil {
		log.Printf("Telegram API call failed: operation=edit_message chat=%s message_id=%d error=%v", b.chatLogMarker(chatID), messageID, err)
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
		log.Printf("Telegram API call failed: operation=delete_message chat=%s message_id=%d error=%v", b.chatLogMarker(chatID), messageID, err)
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

func (b *Bot) chatLogMarker(chatID int64) string {
	if b != nil {
		if b.adminChatID != 0 && chatID == b.adminChatID {
			return "admin"
		}
		if b.publicChatID != 0 && chatID == b.publicChatID {
			return "public"
		}
	}
	if chatID == 0 {
		return "unknown"
	}

	return "private_or_other"
}
