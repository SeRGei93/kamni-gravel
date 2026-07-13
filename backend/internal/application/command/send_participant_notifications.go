package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/repository"
)

const (
	// Telegram ограничивает сообщение 4096 символами без форматирования.
	maxParticipantNotificationTextLength = 4096
	participantNotificationInterval      = 40 * time.Millisecond
)

var (
	// ErrParticipantNotificationsNotConfigured означает отсутствие Telegram-клиента.
	ErrParticipantNotificationsNotConfigured = errors.New("participant notifications are not configured")
	// ErrParticipantNotificationTextEmpty означает пустой текст уведомления.
	ErrParticipantNotificationTextEmpty = errors.New("notification text is empty")
	// ErrParticipantNotificationTextTooLong означает превышение лимита Telegram.
	ErrParticipantNotificationTextTooLong = errors.New("notification text exceeds Telegram limit")
)

// ParticipantNotifier отправляет одно личное Telegram-уведомление участнику.
type ParticipantNotifier interface {
	Send(ctx context.Context, userID int64, text string) error
}

// SendParticipantNotificationsCommand описывает подтверждённую администратором рассылку.
type SendParticipantNotificationsCommand struct {
	EventID uint
	UserIDs []int64
	Text    string
}

// SendParticipantNotificationsResult — итог попытки отправить рассылку.
type SendParticipantNotificationsResult struct {
	Sent    int
	Failed  int
	Skipped int
}

// SendParticipantNotificationsHandler доставляет текст выбранным участникам.
// Перед отправкой он сверяет каждый user_id с участниками события, чтобы клиент
// не мог разослать сообщение произвольным Telegram-пользователям.
type SendParticipantNotificationsHandler struct {
	participantRepo repository.ParticipantRepository
	notifier        ParticipantNotifier
	sleep           func(context.Context, time.Duration) bool
}

// NewSendParticipantNotificationsHandler создаёт handler рассылки.
func NewSendParticipantNotificationsHandler(
	participantRepo repository.ParticipantRepository,
	notifier ParticipantNotifier,
) *SendParticipantNotificationsHandler {
	return &SendParticipantNotificationsHandler{
		participantRepo: participantRepo,
		notifier:        notifier,
		sleep:           sleepParticipantNotification,
	}
}

// Handle отправляет уведомление каждому уникальному выбранному участнику.
func (h *SendParticipantNotificationsHandler) Handle(
	ctx context.Context,
	cmd SendParticipantNotificationsCommand,
) (SendParticipantNotificationsResult, error) {
	var result SendParticipantNotificationsResult
	text, err := h.validate(cmd.Text)
	if err != nil {
		return result, err
	}

	participants, err := h.participantRepo.FindByEvent(ctx, cmd.EventID)
	if err != nil {
		return result, fmt.Errorf("find event participants: %w", err)
	}
	allowedUserIDs := make(map[int64]struct{}, len(participants))
	for _, participant := range participants {
		if participant != nil && participant.UserID > 0 {
			allowedUserIDs[participant.UserID] = struct{}{}
		}
	}

	seen := make(map[int64]struct{}, len(cmd.UserIDs))
	first := true
	for _, userID := range cmd.UserIDs {
		if userID <= 0 {
			result.Skipped++
			continue
		}
		if _, duplicate := seen[userID]; duplicate {
			result.Skipped++
			continue
		}
		seen[userID] = struct{}{}
		if _, allowed := allowedUserIDs[userID]; !allowed {
			result.Skipped++
			log.Printf("INFO participant notification skipped: event_id=%d target_user_id=%d reason=not_event_participant", cmd.EventID, userID)
			continue
		}

		if !first && !h.sleep(ctx, participantNotificationInterval) {
			log.Printf("WARN participant notifications cancelled: event_id=%d sent=%d failed=%d skipped=%d error=%v", cmd.EventID, result.Sent, result.Failed, result.Skipped, ctx.Err())
			return result, fmt.Errorf("wait between participant notifications: %w", ctx.Err())
		}
		first = false

		if err := h.notifier.Send(ctx, userID, text); err != nil {
			result.Failed++
			log.Printf("WARN participant notification delivery failed: event_id=%d target_user_id=%d text_length=%d error=%v", cmd.EventID, userID, len([]rune(text)), err)
			continue
		}
		result.Sent++
	}

	log.Printf("INFO participant notifications delivered: event_id=%d requested=%d sent=%d failed=%d skipped=%d text_length=%d", cmd.EventID, len(cmd.UserIDs), result.Sent, result.Failed, result.Skipped, len([]rune(text)))
	return result, nil
}

func (h *SendParticipantNotificationsHandler) validate(text string) (string, error) {
	if h == nil || h.notifier == nil {
		return "", ErrParticipantNotificationsNotConfigured
	}
	return normalizeParticipantNotificationText(text)
}

func sleepParticipantNotification(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
