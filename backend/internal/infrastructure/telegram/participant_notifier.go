package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
)

const defaultParticipantNotificationRetryWait = time.Second

// participantNotificationAPI — минимальный контракт Telegram-клиента для
// отправки личных уведомлений; он позволяет тестировать адаптер без сети.
type participantNotificationAPI interface {
	SendMessage(ctx context.Context, params *telegrambot.SendMessageParams) (*models.Message, error)
}

// ParticipantNotifier — Telegram-адаптер application-порта рассылки.
type ParticipantNotifier struct {
	api participantNotificationAPI
}

// NewParticipantNotifier создаёт адаптер поверх готового Telegram-клиента.
func NewParticipantNotifier(api participantNotificationAPI) *ParticipantNotifier {
	return &ParticipantNotifier{api: api}
}

// NewParticipantNotifierFromToken создаёт Telegram-клиент для админской
// рассылки. Nil означает, что токен не настроен.
func NewParticipantNotifierFromToken(token string) (*ParticipantNotifier, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	api, err := telegrambot.New(token, telegrambot.WithSkipGetMe())
	if err != nil {
		return nil, fmt.Errorf("create telegram client for participant notifications: %w", err)
	}
	return NewParticipantNotifier(api), nil
}

// Send доставляет текст в личный чат и один раз повторяет запрос после 429.
func (n *ParticipantNotifier) Send(ctx context.Context, userID int64, text string) error {
	if n == nil || n.api == nil {
		return command.ErrParticipantNotificationsNotConfigured
	}
	if _, err := n.api.SendMessage(ctx, &telegrambot.SendMessageParams{ChatID: userID, Text: text}); err != nil {
		var rateLimitErr *telegrambot.TooManyRequestsError
		if !errors.As(err, &rateLimitErr) {
			return fmt.Errorf("send Telegram message: %w", err)
		}

		retryAfter := time.Duration(rateLimitErr.RetryAfter) * time.Second
		if retryAfter <= 0 {
			retryAfter = defaultParticipantNotificationRetryWait
		}
		if !waitParticipantNotificationRetry(ctx, retryAfter) {
			return fmt.Errorf("wait Telegram rate limit: %w", ctx.Err())
		}
		if _, retryErr := n.api.SendMessage(ctx, &telegrambot.SendMessageParams{ChatID: userID, Text: text}); retryErr != nil {
			return fmt.Errorf("retry Telegram message after rate limit: %w", retryErr)
		}
	}
	return nil
}

func waitParticipantNotificationRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
