package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/domain/entity"
)

const (
	defaultGiftNotificationAttempts     = 3
	defaultGiftNotificationInitialDelay = 300 * time.Millisecond
	defaultGiftNotificationMaxDelay     = 2 * time.Second
)

// GiftNotificationRetryConfig задаёт retry/backoff для Telegram-уведомления.
type GiftNotificationRetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// GiftNotifierConfig задаёт получателя и оформление Telegram-уведомлений о призах.
type GiftNotifierConfig struct {
	ChatID      int64
	ChatName    string
	BotUsername string
	MiniappURL  string
	Retry       GiftNotificationRetryConfig
}

// GiftNotifier отправляет Telegram-уведомления о призах в один настроенный чат.
type GiftNotifier struct {
	api             giftNotificationAPI
	chatID          int64
	chatName        string
	botUsername     string
	miniappURL      string
	retry           GiftNotificationRetryConfig
	sleep           func(context.Context, time.Duration) error
	disabledLogOnce sync.Once
}

type giftNotificationAPI interface {
	SendMessage(ctx context.Context, params *telegrambot.SendMessageParams) (*models.Message, error)
	SendPhoto(ctx context.Context, params *telegrambot.SendPhotoParams) (*models.Message, error)
	SendMediaGroup(ctx context.Context, params *telegrambot.SendMediaGroupParams) ([]*models.Message, error)
}

// NewGiftNotifier создаёт notifier поверх готового Telegram API клиента.
func NewGiftNotifier(api giftNotificationAPI, cfg GiftNotifierConfig) *GiftNotifier {
	return &GiftNotifier{
		api:         api,
		chatID:      cfg.ChatID,
		chatName:    normalizeGiftNotificationChatName(cfg.ChatName),
		botUsername: strings.TrimSpace(cfg.BotUsername),
		miniappURL:  validateMiniappURL(cfg.MiniappURL),
		retry:       normalizeGiftNotificationRetry(cfg.Retry),
		sleep:       sleepWithContext,
	}
}

// NewGiftNotifierFromToken создаёт Telegram API клиент для notifier.
func NewGiftNotifierFromToken(token string, cfg GiftNotifierConfig) (*GiftNotifier, error) {
	if cfg.ChatID == 0 {
		return NewGiftNotifier(nil, cfg), nil
	}

	api, err := telegrambot.New(token, telegrambot.WithSkipGetMe())
	if err != nil {
		return nil, fmt.Errorf("create Telegram gift notifier API: %w", err)
	}

	if strings.TrimSpace(cfg.BotUsername) == "" {
		botUsername, err := lookupBotUsername(api)
		if err != nil {
			log.Printf("WARN Could not resolve Telegram gift notifier bot username: chat=%s reason=%v", normalizeGiftNotificationChatName(cfg.ChatName), err)
		} else {
			cfg.BotUsername = botUsername
			log.Printf("INFO Telegram gift notifier bot username resolved: chat=%s", normalizeGiftNotificationChatName(cfg.ChatName))
		}
	}

	return NewGiftNotifier(api, cfg), nil
}

// Notify отправляет уведомление без retry.
func (n *GiftNotifier) Notify(ctx context.Context, gift *entity.Gift) error {
	if n == nil {
		return nil
	}
	if n.chatID == 0 {
		n.disabledLogOnce.Do(func() {
			log.Printf("WARN Gift notification skipped: chat=%s reason=chat_disabled", n.chatName)
		})
		return nil
	}
	if n.api == nil {
		return fmt.Errorf("telegram api is nil")
	}

	return sendGiftNotification(ctx, n.api, n.chatID, n.chatName, n.botUsername, n.miniappURL, gift)
}

// NotifyWithRetry отправляет уведомление с ограниченным exponential backoff.
func (n *GiftNotifier) NotifyWithRetry(ctx context.Context, gift *entity.Gift) error {
	if n == nil {
		return nil
	}

	retry := normalizeGiftNotificationRetry(n.retry)
	var lastErr error
	for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return fmt.Errorf("gift notification cancelled after attempt %d: %w; last error: %v", attempt-1, err, lastErr)
			}
			return err
		}

		if err := n.Notify(ctx, gift); err != nil {
			lastErr = err
		} else {
			if attempt > 1 {
				giftID, eventID, userID := adminGiftLogFields(gift)
				log.Printf("INFO Gift notification retry succeeded: gift_id=%d event_id=%d user_id=%d chat=%s attempt=%d", giftID, eventID, userID, n.chatName, attempt)
			}
			return nil
		}

		if attempt == retry.MaxAttempts {
			break
		}

		delay := giftNotificationBackoffDelay(retry, attempt)
		giftID, eventID, userID := adminGiftLogFields(gift)
		log.Printf("WARN Gift notification retry scheduled: gift_id=%d event_id=%d user_id=%d chat=%s attempt=%d next_attempt=%d delay=%s error=%v", giftID, eventID, userID, n.chatName, attempt, attempt+1, delay, lastErr)
		if err := n.sleep(ctx, delay); err != nil {
			return fmt.Errorf("gift notification retry interrupted after attempt %d: %w; last error: %v", attempt, err, lastErr)
		}
	}

	return fmt.Errorf("gift notification failed after %d attempts: %w", retry.MaxAttempts, lastErr)
}

func sendGiftNotification(ctx context.Context, api giftNotificationAPI, chatID int64, chatName string, botUsername string, miniappURL string, gift *entity.Gift) error {
	giftID, eventID, userID := adminGiftLogFields(gift)
	if gift == nil {
		log.Printf("WARN Gift notification uses fallback text: gift_id=%d event_id=%d user_id=%d chat=%s reason=gift_nil", giftID, eventID, userID, chatName)
	}

	photoFileIDs := giftPhotoFileIDs(gift)
	if gift != nil && len(gift.Attachments) > 0 && len(photoFileIDs) == 0 {
		log.Printf("WARN Gift notification has no usable photos: gift_id=%d event_id=%d user_id=%d chat=%s attachment_count=%d", giftID, eventID, userID, chatName, len(gift.Attachments))
	}

	switch len(photoFileIDs) {
	case 0:
		if err := sendGiftTextNotification(ctx, api, chatID, chatName, botUsername, miniappURL, gift); err != nil {
			log.Printf("ERROR Gift notification failed: gift_id=%d event_id=%d user_id=%d chat=%s delivery=text error=%v", giftID, eventID, userID, chatName, err)
			return err
		}
		log.Printf("INFO Gift notification sent: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=0 media_group_count=0 delivery=text", giftID, eventID, userID, chatName)
		return nil
	case 1:
		if err := sendGiftPhotoNotification(ctx, api, chatID, chatName, botUsername, miniappURL, gift, photoFileIDs[0]); err != nil {
			log.Printf("WARN Gift photo notification failed: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=1 error=%v", giftID, eventID, userID, chatName, err)
			if fallbackErr := sendGiftTextNotification(ctx, api, chatID, chatName, botUsername, miniappURL, gift); fallbackErr != nil {
				log.Printf("ERROR Gift notification failed: gift_id=%d event_id=%d user_id=%d chat=%s delivery=text_fallback error=%v", giftID, eventID, userID, chatName, fallbackErr)
				return fmt.Errorf("send gift photo notification: %w; fallback: %w", err, fallbackErr)
			}
			log.Printf("INFO Gift notification sent: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=1 media_group_count=0 delivery=text_fallback", giftID, eventID, userID, chatName)
			return nil
		}
		log.Printf("INFO Gift notification sent: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=1 media_group_count=0 delivery=photo", giftID, eventID, userID, chatName)
		return nil
	default:
		mediaGroupCount, err := sendGiftMediaGroupNotification(ctx, api, chatID, chatName, botUsername, miniappURL, gift, photoFileIDs)
		if err != nil {
			log.Printf("WARN Gift media group notification failed: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=%d media_group_count=%d error=%v", giftID, eventID, userID, chatName, len(photoFileIDs), mediaGroupCount, err)
			if fallbackErr := sendGiftTextNotification(ctx, api, chatID, chatName, botUsername, miniappURL, gift); fallbackErr != nil {
				log.Printf("ERROR Gift notification failed: gift_id=%d event_id=%d user_id=%d chat=%s delivery=text_fallback error=%v", giftID, eventID, userID, chatName, fallbackErr)
				return fmt.Errorf("send gift media group notification: %w; fallback: %w", err, fallbackErr)
			}
			log.Printf("INFO Gift notification sent: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=%d media_group_count=%d delivery=text_fallback", giftID, eventID, userID, chatName, len(photoFileIDs), mediaGroupCount)
			return nil
		}
		log.Printf("INFO Gift notification sent: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=%d media_group_count=%d delivery=media_group", giftID, eventID, userID, chatName, len(photoFileIDs), mediaGroupCount)
		return nil
	}
}

func sendGiftTextNotification(ctx context.Context, api giftNotificationAPI, chatID int64, chatName string, botUsername string, miniappURL string, gift *entity.Gift) error {
	giftID, eventID, userID := adminGiftLogFields(gift)
	formatter := &Bot{}
	text := formatter.adminGiftNotificationText(gift, telegramTextLimit)
	params := &telegrambot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}
	markupMode := "none"
	if markup, ok := giftNotificationMiniappMarkup(botUsername, miniappURL); ok {
		params.ReplyMarkup = markup
		markupMode = "miniapp_url_button"
	}

	log.Printf("INFO Gift text notification sending: gift_id=%d event_id=%d user_id=%d chat=%s text_len=%d markup=%s", giftID, eventID, userID, chatName, len([]rune(text)), markupMode)
	if _, err := api.SendMessage(ctx, params); err != nil {
		log.Printf("Telegram API call failed: operation=send_message chat=%s error=%v", chatName, err)
		return fmt.Errorf("send gift text notification: %w", err)
	}

	return nil
}

func sendGiftPhotoNotification(ctx context.Context, api giftNotificationAPI, chatID int64, chatName string, botUsername string, miniappURL string, gift *entity.Gift, photoFileID string) error {
	giftID, eventID, userID := adminGiftLogFields(gift)
	formatter := &Bot{}
	caption := formatter.adminGiftNotificationText(gift, telegramCaptionLimit)
	params := &telegrambot.SendPhotoParams{
		ChatID:  chatID,
		Photo:   &models.InputFileString{Data: photoFileID},
		Caption: caption,
	}
	markupMode := "none"
	if markup, ok := giftNotificationMiniappMarkup(botUsername, miniappURL); ok {
		params.ReplyMarkup = markup
		markupMode = "miniapp_url_button"
	}

	log.Printf("INFO Gift photo notification sending: gift_id=%d event_id=%d user_id=%d chat=%s caption_len=%d markup=%s", giftID, eventID, userID, chatName, len([]rune(caption)), markupMode)
	if _, err := api.SendPhoto(ctx, params); err != nil {
		log.Printf("Telegram API call failed: operation=send_photo chat=%s error=%v", chatName, err)
		return fmt.Errorf("send gift photo notification: %w", err)
	}

	return nil
}

func sendGiftMediaGroupNotification(ctx context.Context, api giftNotificationAPI, chatID int64, chatName string, botUsername string, miniappURL string, gift *entity.Gift, photoFileIDs []string) (int, error) {
	chunks := adminGiftMediaGroupChunks(photoFileIDs)
	giftID, eventID, userID := adminGiftLogFields(gift)
	if len(chunks) > 1 {
		log.Printf("INFO Gift media group notification chunked: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=%d media_group_count=%d", giftID, eventID, userID, chatName, len(photoFileIDs), len(chunks))
	}

	caption, parseMode := giftNotificationMediaGroupCaption(botUsername, miniappURL, gift)
	log.Printf("INFO Gift media group notification sending: gift_id=%d event_id=%d user_id=%d chat=%s photo_count=%d media_group_count=%d caption_len=%d parse_mode=%q", giftID, eventID, userID, chatName, len(photoFileIDs), len(chunks), len([]rune(caption)), parseMode)
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

		if _, err := api.SendMediaGroup(ctx, &telegrambot.SendMediaGroupParams{
			ChatID: chatID,
			Media:  media,
		}); err != nil {
			log.Printf("Telegram API call failed: operation=send_media_group chat=%s chunk=%d/%d media_count=%d error=%v", chatName, chunkIndex+1, len(chunks), len(media), err)
			return chunkIndex + 1, fmt.Errorf("send gift media group chunk %d of %d: %w", chunkIndex+1, len(chunks), err)
		}
	}

	return len(chunks), nil
}

func giftNotificationMiniappMarkup(botUsername string, miniappURL string) (models.InlineKeyboardMarkup, bool) {
	miniappLink, ok := giftNotificationMiniappTelegramLink(botUsername, miniappURL)
	if !ok {
		return models.InlineKeyboardMarkup{}, false
	}

	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text: adminGiftMiniappLabel,
					URL:  miniappLink,
				},
			},
		},
	}, true
}

func giftNotificationMediaGroupCaption(botUsername string, miniappURL string, gift *entity.Gift) (string, models.ParseMode) {
	formatter := &Bot{}
	miniappLink, ok := giftNotificationMiniappTelegramLink(botUsername, miniappURL)
	if !ok {
		return formatter.adminGiftNotificationText(gift, telegramCaptionLimit), ""
	}

	return formatter.adminGiftNotificationHTMLText(gift, telegramCaptionLimit, miniappLink), models.ParseModeHTML
}

func giftNotificationMiniappTelegramLink(botUsername string, miniappURL string) (string, bool) {
	if strings.TrimSpace(miniappURL) == "" {
		return "", false
	}

	username := strings.TrimPrefix(strings.TrimSpace(botUsername), "@")
	if username == "" {
		log.Printf("WARN Gift notification miniapp Telegram link unavailable: reason=missing_bot_username")
		return "", false
	}

	return fmt.Sprintf("https://t.me/%s?startapp", username), true
}

func normalizeGiftNotificationRetry(cfg GiftNotificationRetryConfig) GiftNotificationRetryConfig {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultGiftNotificationAttempts
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = defaultGiftNotificationInitialDelay
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = defaultGiftNotificationMaxDelay
	}
	if cfg.MaxDelay < cfg.InitialDelay {
		cfg.MaxDelay = cfg.InitialDelay
	}
	return cfg
}

func giftNotificationBackoffDelay(cfg GiftNotificationRetryConfig, failedAttempt int) time.Duration {
	delay := cfg.InitialDelay
	for i := 1; i < failedAttempt; i++ {
		delay *= 2
		if delay >= cfg.MaxDelay {
			return cfg.MaxDelay
		}
	}
	if delay > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return delay
}

func normalizeGiftNotificationChatName(chatName string) string {
	chatName = strings.TrimSpace(chatName)
	if chatName == "" {
		return "telegram"
	}
	return chatName
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
