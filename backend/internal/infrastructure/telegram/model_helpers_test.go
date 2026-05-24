package telegram

import (
	"reflect"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestMessageCommand(t *testing.T) {
	tests := []struct {
		name string
		msg  *models.Message
		want string
	}{
		{
			name: "start command",
			msg: &models.Message{
				Text:     "/start",
				Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: 6}},
			},
			want: "start",
		},
		{
			name: "start command with bot username",
			msg: &models.Message{
				Text:     "/start@GravelBot",
				Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: 16}},
			},
			want: "start",
		},
		{
			name: "not a command entity",
			msg: &models.Message{
				Text:     "/start",
				Entities: []models.MessageEntity{{Type: models.MessageEntityTypeMention, Offset: 0, Length: 6}},
			},
			want: "",
		},
		{
			name: "nil message",
			msg:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageCommand(tt.msg); got != tt.want {
				t.Fatalf("command mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCallbackMessage(t *testing.T) {
	t.Run("accessible message", func(t *testing.T) {
		ref, ok := callbackMessage(&models.CallbackQuery{
			ID: "callback-id",
			Message: models.MaybeInaccessibleMessage{
				Message: &models.Message{
					ID:   77,
					Chat: models.Chat{ID: 99},
				},
			},
		})
		if !ok {
			t.Fatal("callback message mismatch: got inaccessible")
		}
		if ref.ChatID != 99 || ref.MessageID != 77 {
			t.Fatalf("callback message ref mismatch: got %#v", ref)
		}
	})

	t.Run("inaccessible message", func(t *testing.T) {
		_, ok := callbackMessage(&models.CallbackQuery{
			ID: "callback-id",
			Message: models.MaybeInaccessibleMessage{
				InaccessibleMessage: &models.InaccessibleMessage{
					Chat:      models.Chat{ID: 99},
					MessageID: 77,
				},
			},
		})
		if ok {
			t.Fatal("callback message mismatch: got accessible, want inaccessible")
		}
	})
}

func TestGiftMessageIDTracking(t *testing.T) {
	b := &Bot{sessionManager: session.NewManager(time.Minute)}

	b.appendGiftMessageID(123, 10)
	b.appendGiftMessageID(123, 11)

	want := []int{10, 11}
	if got := b.giftMessageIDs(123); !reflect.DeepEqual(got, want) {
		t.Fatalf("gift message IDs mismatch: got %v, want %v", got, want)
	}
}
