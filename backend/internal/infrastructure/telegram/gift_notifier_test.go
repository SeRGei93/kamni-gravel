package telegram

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/domain/entity"
)

func TestGiftNotifierNotifyWithRetryBacksOffAndSucceeds(t *testing.T) {
	api := &giftNotifierAPIFake{
		sendMessageErrors: []error{
			errors.New("temporary telegram error 1"),
			errors.New("temporary telegram error 2"),
			nil,
		},
	}
	notifier := NewGiftNotifier(api, GiftNotifierConfig{
		ChatID:   -100,
		ChatName: "public",
		Retry: GiftNotificationRetryConfig{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			MaxDelay:     20 * time.Millisecond,
		},
	})
	var delays []time.Duration
	notifier.sleep = func(ctx context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}

	err := notifier.NotifyWithRetry(context.Background(), &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		User:           &entity.User{ID: 123, FirstName: "Alex"},
	})
	if err != nil {
		t.Fatalf("NotifyWithRetry error: %v", err)
	}
	if api.sendMessageAttempts != 3 {
		t.Fatalf("send attempts mismatch: got %d, want 3", api.sendMessageAttempts)
	}
	if !reflect.DeepEqual(delays, []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}) {
		t.Fatalf("backoff delays mismatch: got %v", delays)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("successful message count mismatch: got %d, want 1", len(api.sentMessages))
	}
}

func TestGiftNotifierSendsPublicMediaGroupWithMiniappLink(t *testing.T) {
	api := &giftNotifierAPIFake{}
	notifier := NewGiftNotifier(api, GiftNotifierConfig{
		ChatID:      -100,
		ChatName:    "public",
		BotUsername: "GravelBot",
		MiniappURL:  "https://example.com/miniapp/gifts",
	})

	err := notifier.Notify(context.Background(), &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		User:           &entity.User{ID: 123, FirstName: "Alex"},
		Attachments: []entity.GiftAttachment{
			{ID: 1, FileType: "photo", TelegramFileID: "photo-1"},
			{ID: 2, FileType: "photo", TelegramFileID: "photo-2"},
		},
	})
	if err != nil {
		t.Fatalf("Notify error: %v", err)
	}
	if len(api.mediaGroups) != 1 {
		t.Fatalf("media group count mismatch: got %d, want 1", len(api.mediaGroups))
	}
	if got := chatIDFromAny(api.mediaGroups[0].ChatID); got != int64(-100) {
		t.Fatalf("media group chat mismatch: got %d, want -100", got)
	}
	first, ok := api.mediaGroups[0].Media[0].(*models.InputMediaPhoto)
	if !ok {
		t.Fatalf("first media type mismatch: got %T", api.mediaGroups[0].Media[0])
	}
	if first.ParseMode != models.ParseModeHTML {
		t.Fatalf("parse mode mismatch: got %q, want %q", first.ParseMode, models.ParseModeHTML)
	}
	if !strings.Contains(first.Caption, `<a href="https://t.me/GravelBot?startapp">призовой фонд</a>`) {
		t.Fatalf("caption missing miniapp link: %q", first.Caption)
	}
}

type giftNotifierAPIFake struct {
	sentMessages        []*telegrambot.SendMessageParams
	sentPhotos          []*telegrambot.SendPhotoParams
	mediaGroups         []*telegrambot.SendMediaGroupParams
	sendMessageErrors   []error
	sendPhotoErrors     []error
	mediaGroupErrors    []error
	sendMessageAttempts int
	sendPhotoAttempts   int
	mediaGroupAttempts  int
}

func (a *giftNotifierAPIFake) SendMessage(ctx context.Context, params *telegrambot.SendMessageParams) (*models.Message, error) {
	a.sendMessageAttempts++
	if err := nextGiftNotifierError(&a.sendMessageErrors); err != nil {
		return nil, err
	}
	a.sentMessages = append(a.sentMessages, params)
	return &models.Message{ID: len(a.sentMessages), Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}, Text: params.Text}, nil
}

func (a *giftNotifierAPIFake) SendPhoto(ctx context.Context, params *telegrambot.SendPhotoParams) (*models.Message, error) {
	a.sendPhotoAttempts++
	if err := nextGiftNotifierError(&a.sendPhotoErrors); err != nil {
		return nil, err
	}
	a.sentPhotos = append(a.sentPhotos, params)
	return &models.Message{ID: len(a.sentPhotos), Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}, Caption: params.Caption}, nil
}

func (a *giftNotifierAPIFake) SendMediaGroup(ctx context.Context, params *telegrambot.SendMediaGroupParams) ([]*models.Message, error) {
	a.mediaGroupAttempts++
	if err := nextGiftNotifierError(&a.mediaGroupErrors); err != nil {
		return nil, err
	}
	a.mediaGroups = append(a.mediaGroups, params)
	return []*models.Message{{ID: len(a.mediaGroups), Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}}}, nil
}

func nextGiftNotifierError(errorsQueue *[]error) error {
	if len(*errorsQueue) == 0 {
		return nil
	}
	err := (*errorsQueue)[0]
	*errorsQueue = (*errorsQueue)[1:]
	return err
}
