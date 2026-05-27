package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// ResultHandler обрабатывает отправку результатов
type ResultHandler struct {
	sessionManager      *session.Manager
	eventRepo           repository.EventRepository
	participantRepo     repository.ParticipantRepository
	submitResultHandler *command.SubmitResultHandler
	now                 func() time.Time
}

// ResultHandlerOption настраивает Telegram handler результатов.
type ResultHandlerOption func(*ResultHandler)

// WithResultHandlerClock задаёт источник текущего времени для тестов.
func WithResultHandlerClock(now func() time.Time) ResultHandlerOption {
	return func(h *ResultHandler) {
		if now != nil {
			h.now = now
		}
	}
}

// ResultLinkPromptText возвращает текст запроса ссылки результата.
func ResultLinkPromptText(texts entity.EventTelegramTexts) string {
	return entity.NormalizeEventTelegramTexts(texts).ResultPrompt
}

// ResultLinkInvalidInputText возвращает текст для повторного запроса ссылки результата.
func ResultLinkInvalidInputText(texts entity.EventTelegramTexts) string {
	return entity.NormalizeEventTelegramTexts(texts).ResultInvalidLink
}

// NewResultHandler создаёт новый handler
func NewResultHandler(
	sessionManager *session.Manager,
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	submitResultHandler *command.SubmitResultHandler,
	options ...ResultHandlerOption,
) *ResultHandler {
	handler := &ResultHandler{
		sessionManager:      sessionManager,
		eventRepo:           eventRepo,
		participantRepo:     participantRepo,
		submitResultHandler: submitResultHandler,
		now:                 time.Now,
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

// StartSubmitResult начинает процесс отправки результата
func (h *ResultHandler) StartSubmitResult(ctx context.Context, userID int64) (string, *models.InlineKeyboardMarkup) {
	// Получаем активное событие
	event, err := h.eventRepo.FindActive(ctx)
	if err != nil {
		if isNoActiveEventError(err) {
			log.Printf("INFO Telegram result submission blocked: user_id=%d reason=no_active_event", userID)
			return "В данный момент нет активных событий.", nil
		}

		log.Printf("Error finding active event: %v", err)
		return "Произошла ошибка. Попробуйте позже.", nil
	}

	if event == nil {
		log.Printf("INFO Telegram result submission blocked: user_id=%d reason=no_active_event", userID)
		return "В данный момент нет активных событий.", nil
	}

	texts := entity.NormalizeEventTelegramTexts(event.TelegramTexts)
	startTime, ok := event.SubmissionStartTimeInMinsk()
	if !ok {
		log.Printf("INFO Telegram result submission blocked: user_id=%d event_id=%d reason=start_not_configured", userID, event.ID)
		return texts.ResultStartMissing, nil
	}

	if !event.HasStartedAt(h.now()) {
		log.Printf(
			"INFO Telegram result submission blocked: user_id=%d event_id=%d start_minsk_time=%q reason=event_not_started",
			userID,
			event.ID,
			valueobject.FormatMinskDateTime(startTime),
		)
		return applyResultTextPlaceholders(texts.ResultNotStarted, map[string]string{
			"start_time": valueobject.FormatMinskDateTime(startTime),
		}), nil
	}

	// Проверяем, зарегистрирован ли участник
	participant, err := h.participantRepo.FindByUserAndEvent(ctx, userID, event.ID)
	if err != nil || participant == nil {
		// Участник не найден или ошибка при поиске
		log.Printf("Participant not found for user %d, event %d", userID, event.ID)
		return texts.ResultNotRegistered, nil
	}

	// Проверяем, не отправлен ли уже результат
	if participant.IsFinished() {
		return texts.ResultAlreadySent, nil
	}

	// Сохраняем ID участника в сессии
	h.sessionManager.SetData(userID, "event_telegram_texts", texts)
	h.sessionManager.SetData(userID, "event_id", event.ID)
	h.sessionManager.SetData(userID, "participant_id", participant.ID)
	h.sessionManager.SetState(userID, session.StateAwaitingResultLink)

	keyboard := keyboard.CancelMenu()

	return ResultLinkPromptText(texts), &keyboard
}

// HandleResultLink обрабатывает ссылку на результат
func (h *ResultHandler) HandleResultLink(ctx context.Context, userID int64, resultLink string) (string, error) {
	// Получаем сохранённые данные
	participantIDRaw, ok := h.sessionManager.GetData(userID, "participant_id")
	if !ok {
		return "Ошибка: данные сессии не найдены. Начните отправку результата заново.", nil
	}
	participantID, ok := participantIDRaw.(uint)
	if !ok {
		log.Printf("WARN Invalid result session data: user_id=%d key=participant_id type=%T", userID, participantIDRaw)
		return "Ошибка: данные сессии некорректны. Начните отправку результата заново.", nil
	}

	// Выполняем команду отправки результата
	cmd := command.SubmitResultCommand{
		ParticipantID: participantID,
		ResultLink:    resultLink,
	}

	participant, err := h.submitResultHandler.Handle(ctx, cmd)
	if err != nil {
		if errors.Is(err, command.ErrInvalidResultLink) {
			eventID := resultSessionEventID(h.sessionManager, userID)
			log.Printf(
				"INFO Invalid result submission attempt: user_id=%d participant_id=%d event_id=%d reason=invalid_strava_format",
				userID,
				participantID,
				eventID,
			)
			return ResultLinkInvalidInputText(resultSessionTelegramTexts(h.sessionManager, userID)), nil
		}

		log.Printf("Error submitting result: user_id=%d participant_id=%d error=%v", userID, participantID, err)
		return fmt.Sprintf("Ошибка при отправке результата: %v", err), err
	}

	// Очищаем сессию
	h.sessionManager.ResetState(userID)

	return applyResultTextPlaceholders(
		resultSessionTelegramTexts(h.sessionManager, userID).ResultSuccess,
		map[string]string{"result_link": participant.GetResultLink()},
	), nil
}

func applyResultTextPlaceholders(text string, values map[string]string) string {
	for key, value := range values {
		text = strings.ReplaceAll(text, "{"+key+"}", value)
	}
	return text
}

func resultSessionTelegramTexts(manager *session.Manager, userID int64) entity.EventTelegramTexts {
	textsRaw, ok := manager.GetData(userID, "event_telegram_texts")
	if !ok {
		return entity.NormalizeEventTelegramTexts(entity.EventTelegramTexts{})
	}

	texts, ok := textsRaw.(entity.EventTelegramTexts)
	if !ok {
		log.Printf("WARN Invalid result session data: user_id=%d key=event_telegram_texts type=%T", userID, textsRaw)
		return entity.NormalizeEventTelegramTexts(entity.EventTelegramTexts{})
	}

	return entity.NormalizeEventTelegramTexts(texts)
}

func resultSessionEventID(manager *session.Manager, userID int64) uint {
	eventIDRaw, ok := manager.GetData(userID, "event_id")
	if !ok {
		return 0
	}

	eventID, ok := eventIDRaw.(uint)
	if !ok {
		log.Printf("WARN Invalid result session data: user_id=%d key=event_id type=%T", userID, eventIDRaw)
		return 0
	}

	return eventID
}

// CancelSubmitResult отменяет отправку результата
func (h *ResultHandler) CancelSubmitResult(userID int64) string {
	h.sessionManager.ResetState(userID)
	return "Отправка результата отменена."
}

func isNoActiveEventError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no active event")
}
