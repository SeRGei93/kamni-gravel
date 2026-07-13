package telegram

import (
	"context"
	"testing"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func TestParticipantNotifierSendsPlainTextMessage(t *testing.T) {
	api := &participantNotificationAPIFake{}
	notifier := NewParticipantNotifier(api)

	if err := notifier.Send(context.Background(), 42, "Привет!"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if api.params == nil {
		t.Fatal("SendMessage() was not called")
	}
	if api.params.ChatID != int64(42) || api.params.Text != "Привет!" {
		t.Fatalf("params = %+v, want chat 42 and message text", api.params)
	}
}

type participantNotificationAPIFake struct {
	params *telegrambot.SendMessageParams
}

func (f *participantNotificationAPIFake) SendMessage(ctx context.Context, params *telegrambot.SendMessageParams) (*models.Message, error) {
	f.params = params
	return &models.Message{}, nil
}
