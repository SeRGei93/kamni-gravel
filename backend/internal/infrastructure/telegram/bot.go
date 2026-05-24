package telegram

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// Bot представляет Telegram бота
type Bot struct {
	api            *tgbotapi.BotAPI
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

	// Command handlers
	registerParticipantHandler *command.RegisterParticipantHandler
	addGiftHandler             *command.AddGiftHandler
	submitResultHandler        *command.SubmitResultHandler
	assignPrizeHandler         *command.AssignPrizeHandler

	// Query handlers
	getParticipantsHandler *query.GetParticipantsHandler
	getGiftsHandler        *query.GetGiftsHandler
	getEventsHandler       *query.GetEventsHandler
	getCriteriaHandler     *query.GetCriteriaHandler
	getStatsHandler        *query.GetStatsHandler
}

// Config представляет конфигурацию бота
type Config struct {
	Token          string
	Debug          bool
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
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	api.Debug = cfg.Debug

	// Создаём session manager
	sessionManager := session.NewManager(cfg.SessionTimeout)

	// Создаём command handlers
	registerParticipantHandler := command.NewRegisterParticipantHandler(
		userRepo,
		eventRepo,
		participantRepo,
	)

	addGiftHandler := command.NewAddGiftHandler(
		userRepo,
		eventRepo,
		giftRepo,
	)

	submitResultHandler := command.NewSubmitResultHandler(
		participantRepo,
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
	getStatsHandler := query.NewGetStatsHandler(
		eventRepo,
		participantRepo,
		giftRepo,
		resultRepo,
		criteriaRepo,
	)

	bot := &Bot{
		api:                        api,
		sessionManager:             sessionManager,
		userRepo:                   userRepo,
		eventRepo:                  eventRepo,
		participantRepo:            participantRepo,
		resultRepo:                 resultRepo,
		giftRepo:                   giftRepo,
		criteriaRepo:               criteriaRepo,
		giftCriteriaRepo:           giftCriteriaRepo,
		prizeAssignmentRepo:        prizeAssignmentRepo,
		registerParticipantHandler: registerParticipantHandler,
		addGiftHandler:             addGiftHandler,
		submitResultHandler:        submitResultHandler,
		assignPrizeHandler:         assignPrizeHandler,
		getParticipantsHandler:     getParticipantsHandler,
		getGiftsHandler:            getGiftsHandler,
		getEventsHandler:           getEventsHandler,
		getCriteriaHandler:         getCriteriaHandler,
		getStatsHandler:            getStatsHandler,
	}

	log.Printf("Authorized on account %s", api.Self.UserName)

	return bot, nil
}

// Start запускает бота
func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Println("Bot started, waiting for updates...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Bot stopped")
			return ctx.Err()
		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

// handleUpdate обрабатывает входящее обновление
func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Обработка команд
	if update.Message != nil && update.Message.IsCommand() {
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
}

// SendMessage отправляет текстовое сообщение
func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	return err
}

// SendMessageWithKeyboard отправляет сообщение с inline клавиатурой
func (b *Bot) SendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

// EditMessage редактирует существующее сообщение
func (b *Bot) EditMessage(chatID int64, messageID int, text string) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	_, err := b.api.Send(msg)
	return err
}

// AnswerCallback отвечает на callback query
func (b *Bot) AnswerCallback(callbackID string, text string) error {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := b.api.Request(callback)
	return err
}

// DeleteMessage удаляет сообщение
func (b *Bot) DeleteMessage(chatID int64, messageID int) error {
	msg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := b.api.Request(msg)
	return err
}
