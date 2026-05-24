package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// ResultHandler обрабатывает отправку результатов
type ResultHandler struct {
	sessionManager      *session.Manager
	eventRepo           repository.EventRepository
	participantRepo     repository.ParticipantRepository
	submitResultHandler *command.SubmitResultHandler
}

// NewResultHandler создаёт новый handler
func NewResultHandler(
	sessionManager *session.Manager,
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	submitResultHandler *command.SubmitResultHandler,
) *ResultHandler {
	return &ResultHandler{
		sessionManager:      sessionManager,
		eventRepo:           eventRepo,
		participantRepo:     participantRepo,
		submitResultHandler: submitResultHandler,
	}
}

// StartSubmitResult начинает процесс отправки результата
func (h *ResultHandler) StartSubmitResult(ctx context.Context, userID int64) (string, *models.InlineKeyboardMarkup) {
	// Получаем активное событие
	event, err := h.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("Error finding active event: %v", err)
		return "Произошла ошибка. Попробуйте позже.", nil
	}

	if event == nil {
		return "В данный момент нет активных событий.", nil
	}

	// Проверяем, зарегистрирован ли участник
	participant, err := h.participantRepo.FindByUserAndEvent(ctx, userID, event.ID)
	if err != nil || participant == nil {
		// Участник не найден или ошибка при поиске
		log.Printf("Participant not found for user %d, event %d", userID, event.ID)
		return "Вы не зарегистрированы на это событие. Сначала пройдите регистрацию.", nil
	}

	// Проверяем, не отправлен ли уже результат
	if participant.IsFinished() {
		return "Вы уже отправили результат!", nil
	}

	// Сохраняем ID участника в сессии
	h.sessionManager.SetData(userID, "participant_id", participant.ID)
	h.sessionManager.SetState(userID, session.StateAwaitingResultLink)

	text := `🏁 Отправка результата

Отправьте ссылку на ваш результат (Strava или Komoot).

Пример:
https://www.strava.com/activities/123456789
https://www.komoot.com/tour/123456789`

	keyboard := keyboard.CancelMenu()

	return text, &keyboard
}

// HandleResultLink обрабатывает ссылку на результат
func (h *ResultHandler) HandleResultLink(ctx context.Context, userID int64, resultLink string) (string, error) {
	// Получаем сохранённые данные
	participantIDRaw, ok := h.sessionManager.GetData(userID, "participant_id")
	if !ok {
		return "Ошибка: данные сессии не найдены. Начните отправку результата заново.", nil
	}
	participantID := participantIDRaw.(uint)

	// Выполняем команду отправки результата
	cmd := command.SubmitResultCommand{
		ParticipantID: participantID,
		ResultLink:    resultLink,
	}

	participant, err := h.submitResultHandler.Handle(ctx, cmd)
	if err != nil {
		log.Printf("Error submitting result: %v", err)
		return fmt.Sprintf("Ошибка при отправке результата: %v", err), err
	}

	// Очищаем сессию
	h.sessionManager.ResetState(userID)

	return fmt.Sprintf(`✅ Результат принят!

Ссылка: %s

Ваше время будет обработано администратором. Следите за обновлениями! 🏆`, participant.GetResultLink()), nil
}

// CancelSubmitResult отменяет отправку результата
func (h *ResultHandler) CancelSubmitResult(userID int64) string {
	h.sessionManager.ResetState(userID)
	return "Отправка результата отменена."
}
