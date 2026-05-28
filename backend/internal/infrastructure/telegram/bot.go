package telegram

import (
	"context"
	"errors"
	"fmt"
	"html"
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

const (
	telegramCaptionLimit    = 1024
	telegramTextLimit       = 4096
	telegramMediaGroupLimit = 10
	adminGiftMiniappLabel   = "призовой фонд"
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
	SendPhoto(ctx context.Context, params *telegrambot.SendPhotoParams) (*models.Message, error)
	SendMediaGroup(ctx context.Context, params *telegrambot.SendMediaGroupParams) ([]*models.Message, error)
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

func (b *Bot) notifyAdminAboutGift(ctx context.Context, gift *entity.Gift) error {
	if b == nil {
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

	giftID, eventID, userID := adminGiftLogFields(gift)
	if gift == nil {
		log.Printf("WARN Admin gift notification uses fallback text: gift_id=%d event_id=%d user_id=%d chat=admin reason=gift_nil", giftID, eventID, userID)
	}

	photoFileIDs := giftPhotoFileIDs(gift)
	if gift != nil && len(gift.Attachments) > 0 && len(photoFileIDs) == 0 {
		log.Printf("WARN Admin gift notification has no usable photos: gift_id=%d event_id=%d user_id=%d chat=admin attachment_count=%d", giftID, eventID, userID, len(gift.Attachments))
	}

	switch len(photoFileIDs) {
	case 0:
		if err := b.sendAdminGiftTextNotification(ctx, gift); err != nil {
			log.Printf("ERROR Admin gift notification failed: gift_id=%d event_id=%d user_id=%d chat=admin delivery=text error=%v", giftID, eventID, userID, err)
			return err
		}
		log.Printf("INFO Admin gift notification sent: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=0 media_group_count=0 delivery=text", giftID, eventID, userID)
		return nil

	case 1:
		if err := b.sendAdminGiftPhotoNotification(ctx, gift, photoFileIDs[0]); err != nil {
			log.Printf("WARN Admin gift photo notification failed: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=1 error=%v", giftID, eventID, userID, err)
			if fallbackErr := b.sendAdminGiftTextNotification(ctx, gift); fallbackErr != nil {
				log.Printf("ERROR Admin gift notification failed: gift_id=%d event_id=%d user_id=%d chat=admin delivery=text_fallback error=%v", giftID, eventID, userID, fallbackErr)
				return fmt.Errorf("send admin gift photo: %w; fallback: %w", err, fallbackErr)
			}
			log.Printf("INFO Admin gift notification sent: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=1 media_group_count=0 delivery=text_fallback", giftID, eventID, userID)
			return nil
		}
		log.Printf("INFO Admin gift notification sent: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=1 media_group_count=0 delivery=photo", giftID, eventID, userID)
		return nil

	default:
		mediaGroupCount, err := b.sendAdminGiftMediaGroupNotification(ctx, gift, photoFileIDs)
		if err != nil {
			log.Printf("WARN Admin gift media group notification failed: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=%d media_group_count=%d error=%v", giftID, eventID, userID, len(photoFileIDs), mediaGroupCount, err)
			if fallbackErr := b.sendAdminGiftTextNotification(ctx, gift); fallbackErr != nil {
				log.Printf("ERROR Admin gift notification failed: gift_id=%d event_id=%d user_id=%d chat=admin delivery=text_fallback error=%v", giftID, eventID, userID, fallbackErr)
				return fmt.Errorf("send admin gift media group: %w; fallback: %w", err, fallbackErr)
			}
			log.Printf("INFO Admin gift notification sent: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=%d media_group_count=%d delivery=text_fallback", giftID, eventID, userID, len(photoFileIDs), mediaGroupCount)
			return nil
		}
		log.Printf("INFO Admin gift notification sent: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=%d media_group_count=%d delivery=media_group", giftID, eventID, userID, len(photoFileIDs), mediaGroupCount)
		return nil
	}
}

func (b *Bot) sendAdminGiftTextNotification(ctx context.Context, gift *entity.Gift) error {
	text := b.adminGiftNotificationText(gift, telegramTextLimit)
	if markup, ok := b.adminGiftMiniappMarkup(); ok {
		_, err := b.SendMessageWithKeyboard(ctx, b.adminChatID, text, markup)
		if err != nil {
			return fmt.Errorf("send admin gift text notification: %w", err)
		}
		return nil
	}

	_, err := b.SendMessage(ctx, b.adminChatID, text)
	if err != nil {
		return fmt.Errorf("send admin gift text notification: %w", err)
	}

	return nil
}

func (b *Bot) sendAdminGiftPhotoNotification(ctx context.Context, gift *entity.Gift, photoFileID string) error {
	caption := b.adminGiftNotificationText(gift, telegramCaptionLimit)
	params := &telegrambot.SendPhotoParams{
		ChatID:  b.adminChatID,
		Photo:   &models.InputFileString{Data: photoFileID},
		Caption: caption,
	}
	if markup, ok := b.adminGiftMiniappMarkup(); ok {
		params.ReplyMarkup = markup
	}

	_, err := b.api.SendPhoto(ctx, params)
	if err != nil {
		return fmt.Errorf("send admin gift photo notification: %w", err)
	}

	return nil
}

func (b *Bot) sendAdminGiftMediaGroupNotification(ctx context.Context, gift *entity.Gift, photoFileIDs []string) (int, error) {
	chunks := adminGiftMediaGroupChunks(photoFileIDs)
	if len(chunks) > 1 {
		giftID, eventID, userID := adminGiftLogFields(gift)
		log.Printf("INFO Admin gift media group notification chunked: gift_id=%d event_id=%d user_id=%d chat=admin photo_count=%d media_group_count=%d", giftID, eventID, userID, len(photoFileIDs), len(chunks))
	}

	caption, parseMode := b.adminGiftMediaGroupCaption(gift)
	for chunkIndex, chunk := range chunks {
		media := make([]models.InputMedia, 0, len(chunk))
		for photoIndex, photoFileID := range chunk {
			item := &models.InputMediaPhoto{Media: photoFileID}
			if chunkIndex == 0 && photoIndex == 0 {
				item.Caption = caption
				item.ParseMode = parseMode
			}
			media = append(media, item)
		}

		if _, err := b.api.SendMediaGroup(ctx, &telegrambot.SendMediaGroupParams{
			ChatID: b.adminChatID,
			Media:  media,
		}); err != nil {
			return chunkIndex + 1, fmt.Errorf("send admin gift media group chunk %d of %d: %w", chunkIndex+1, len(chunks), err)
		}
	}

	return len(chunks), nil
}

func (b *Bot) adminGiftMiniappMarkup() (models.InlineKeyboardMarkup, bool) {
	if b == nil || strings.TrimSpace(b.miniappURL) == "" {
		return models.InlineKeyboardMarkup{}, false
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:   adminGiftMiniappLabel,
					WebApp: &models.WebAppInfo{URL: b.miniappURL},
				},
			},
		},
	}, true
}

func (b *Bot) adminGiftMediaGroupCaption(gift *entity.Gift) (string, models.ParseMode) {
	if b == nil || strings.TrimSpace(b.miniappURL) == "" {
		return b.adminGiftNotificationText(gift, telegramCaptionLimit), ""
	}

	miniappLink, ok := b.adminGiftMiniappTelegramLink()
	if !ok {
		return b.adminGiftNotificationText(gift, telegramCaptionLimit), ""
	}

	return b.adminGiftNotificationHTMLText(gift, telegramCaptionLimit, miniappLink), models.ParseModeHTML
}

func (b *Bot) adminGiftMiniappTelegramLink() (string, bool) {
	if b == nil || strings.TrimSpace(b.miniappURL) == "" {
		return "", false
	}

	username := strings.TrimPrefix(strings.TrimSpace(b.botUsername), "@")
	if username == "" {
		log.Printf("WARN Admin gift miniapp Telegram link unavailable: reason=missing_bot_username")
		return "", false
	}

	return fmt.Sprintf("https://t.me/%s?startapp", username), true
}

func giftPhotoFileIDs(gift *entity.Gift) []string {
	if gift == nil {
		return nil
	}

	fileIDs := make([]string, 0, len(gift.Attachments))
	for _, attachment := range gift.Attachments {
		if strings.TrimSpace(attachment.FileType) != "photo" {
			continue
		}
		fileID := strings.TrimSpace(attachment.TelegramFileID)
		if fileID == "" {
			continue
		}
		fileIDs = append(fileIDs, fileID)
	}

	return fileIDs
}

func adminGiftMediaGroupChunks(photoFileIDs []string) [][]string {
	if len(photoFileIDs) == 0 {
		return nil
	}

	chunks := make([][]string, 0, (len(photoFileIDs)+telegramMediaGroupLimit-1)/telegramMediaGroupLimit)
	for len(photoFileIDs) > 0 {
		if len(photoFileIDs) <= telegramMediaGroupLimit {
			chunks = append(chunks, photoFileIDs)
			break
		}

		chunkSize := telegramMediaGroupLimit
		if len(photoFileIDs)-chunkSize == 1 {
			chunkSize = telegramMediaGroupLimit - 1
		}
		chunks = append(chunks, photoFileIDs[:chunkSize])
		photoFileIDs = photoFileIDs[chunkSize:]
	}

	return chunks
}

func adminGiftLogFields(gift *entity.Gift) (uint, uint, int64) {
	if gift == nil {
		return 0, 0, 0
	}

	return gift.ID, gift.EventID, gift.UserID
}

func (b *Bot) adminGiftNotificationText(gift *entity.Gift, limit int) string {
	if limit <= 0 {
		limit = telegramTextLimit
	}
	if gift == nil {
		return truncateTelegramText("Данные приза недоступны.", limit)
	}

	description := strings.TrimSpace(gift.Description)
	if description == "" {
		description = "не указано"
	}

	suffix := fmt.Sprintf(
		"\n\nОт: %s\nГендер: %s\nВелосипед: %s",
		adminGiftDonorLabel(gift),
		adminGiftGenderLabel(gift.GenderFilter),
		adminGiftBikeTypeLabel(gift.BikeTypeFilter),
	)

	descriptionLimit := limit - runeLen(suffix)
	if descriptionLimit < 1 {
		return truncateTelegramText(suffix, limit)
	}

	description = truncateTelegramText(description, descriptionLimit)
	return description + suffix
}

func (b *Bot) adminGiftNotificationHTMLText(gift *entity.Gift, limit int, miniappLink string) string {
	if limit <= 0 {
		limit = telegramTextLimit
	}

	if gift == nil {
		text := "Данные приза недоступны."
		if miniappLink != "" {
			text += b.adminGiftMiniappHTMLSuffix(miniappLink)
		}
		return truncateTelegramText(text, limit)
	}

	description := strings.TrimSpace(gift.Description)
	if description == "" {
		description = "не указано"
	}

	suffix := fmt.Sprintf(
		"\n\nОт: %s\nГендер: %s\nВелосипед: %s",
		html.EscapeString(adminGiftDonorLabel(gift)),
		html.EscapeString(adminGiftGenderLabel(gift.GenderFilter)),
		html.EscapeString(adminGiftBikeTypeLabel(gift.BikeTypeFilter)),
	)
	if miniappLink != "" {
		suffix += b.adminGiftMiniappHTMLSuffix(miniappLink)
	}

	descriptionLimit := limit - runeLen(suffix)
	if descriptionLimit < 1 {
		return truncateTelegramText(suffix, limit)
	}

	description = truncateEscapedHTMLText(description, descriptionLimit)
	return description + suffix
}

func (b *Bot) adminGiftMiniappHTMLSuffix(miniappLink string) string {
	if strings.TrimSpace(miniappLink) == "" {
		return ""
	}

	return fmt.Sprintf(
		"\n\n<a href=\"%s\">%s</a>",
		html.EscapeString(miniappLink),
		adminGiftMiniappLabel,
	)
}

func adminGiftDonorLabel(gift *entity.Gift) string {
	if gift == nil {
		return "неизвестен"
	}

	if gift.User != nil {
		firstName := strings.TrimSpace(gift.User.FirstName)
		lastName := strings.TrimSpace(gift.User.LastName)
		username := strings.TrimPrefix(strings.TrimSpace(gift.User.Username), "@")
		fullName := strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))

		switch {
		case fullName != "" && username != "":
			return fmt.Sprintf("%s (@%s)", fullName, username)
		case username != "":
			return "@" + username
		case fullName != "":
			return fullName
		}
	}

	if gift.UserID != 0 {
		return fmt.Sprintf("user_id: %d", gift.UserID)
	}

	return "неизвестен"
}

func adminGiftGenderLabel(gender string) string {
	switch strings.TrimSpace(gender) {
	case "male":
		return "👨 Мужской"
	case "female":
		return "👩 Женский"
	case "all", "":
		return "👥 Любой"
	default:
		return gender
	}
}

func adminGiftBikeTypeLabel(bikeType string) string {
	switch strings.TrimSpace(bikeType) {
	case "gravel":
		return "🚵 Гравийник"
	case "mtb":
		return "🏔 МТБ"
	case "road":
		return "🚴 Шоссе"
	case "single_speed":
		return "⚡️ Фикс"
	case "tandem":
		return "👥 Тандем"
	case "all", "":
		return "🚲 Любой"
	default:
		return bikeType
	}
}

func truncateTelegramText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if runeLen(text) <= limit {
		return text
	}
	runes := []rune(text)
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func truncateEscapedHTMLText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}

	escaped := html.EscapeString(text)
	if runeLen(escaped) <= limit {
		return escaped
	}
	if limit <= 3 {
		return truncateTelegramText(escaped, limit)
	}

	var builder strings.Builder
	written := 0
	for _, r := range text {
		escapedRune := html.EscapeString(string(r))
		escapedLen := runeLen(escapedRune)
		if written+escapedLen > limit-3 {
			break
		}
		builder.WriteString(escapedRune)
		written += escapedLen
	}

	return builder.String() + "..."
}

func runeLen(text string) int {
	return len([]rune(text))
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
