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

// Ключи данных сессии сценария отправки результата.
const (
	sessionKeyResultTexts       = "event_telegram_texts"
	sessionKeyResultEventID     = "event_id"
	sessionKeyResultParticipant = "participant_id"
	sessionKeyPendingResultLink = "pending_result_link"
)

const (
	resultReplaceConfirmText = "У вас уже есть отправленный результат:\n{current_link}\n\nЗаменить его на новый?\n{new_link}"
	resultReplaceCancelText  = "Результат не изменён."
	resultReplaceStaleText   = "Не удалось определить ссылку для замены. Отправьте результат заново."
)

// submissionContext хранит данные, необходимые для отправки результата участником.
type submissionContext struct {
	event       *entity.Event
	participant *entity.Participant
	texts       entity.EventTelegramTexts
}

// resolveSubmission проверяет, может ли пользователь отправить результат
// (есть активное событие, окно открыто, участник зарегистрирован). При ошибке
// возвращает готовый для показа текст и ok=false.
func (h *ResultHandler) resolveSubmission(ctx context.Context, userID int64) (submissionContext, string, bool) {
	event, err := h.eventRepo.FindActive(ctx)
	if err != nil {
		if isNoActiveEventError(err) {
			log.Printf("INFO Telegram result submission blocked: user_id=%d reason=no_active_event", userID)
			return submissionContext{}, "В данный момент нет активных событий.", false
		}

		log.Printf("Error finding active event: %v", err)
		return submissionContext{}, "Произошла ошибка. Попробуйте позже.", false
	}

	if event == nil {
		log.Printf("INFO Telegram result submission blocked: user_id=%d reason=no_active_event", userID)
		return submissionContext{}, "В данный момент нет активных событий.", false
	}

	if event.ResultIntakeClosedAt(h.now()) {
		log.Printf("INFO Telegram result submission blocked: user_id=%d event_id=%d reason=result_intake_closed", userID, event.ID)
		return submissionContext{}, ResultIntakeClosedText(event, h.now()), false
	}

	texts := entity.NormalizeEventTelegramTexts(event.TelegramTexts)
	startTime, ok := event.SubmissionStartTimeInMinsk()
	if !ok {
		log.Printf("INFO Telegram result submission blocked: user_id=%d event_id=%d reason=start_not_configured", userID, event.ID)
		return submissionContext{}, texts.ResultStartMissing, false
	}

	if !event.HasStartedAt(h.now()) {
		log.Printf(
			"INFO Telegram result submission blocked: user_id=%d event_id=%d start_minsk_time=%q reason=event_not_started",
			userID,
			event.ID,
			valueobject.FormatMinskDateTime(startTime),
		)
		return submissionContext{}, applyResultTextPlaceholders(texts.ResultNotStarted, map[string]string{
			"start_time": valueobject.FormatMinskDateTime(startTime),
		}), false
	}

	// Проверяем, зарегистрирован ли участник
	participant, err := h.participantRepo.FindByUserAndEvent(ctx, userID, event.ID)
	if err != nil || participant == nil {
		// Участник не найден или ошибка при поиске
		log.Printf("Participant not found for user %d, event %d", userID, event.ID)
		return submissionContext{}, texts.ResultNotRegistered, false
	}

	return submissionContext{event: event, participant: participant, texts: texts}, "", true
}

// storeSubmissionSession сохраняет данные сценария отправки результата в сессии.
func (h *ResultHandler) storeSubmissionSession(userID int64, sc submissionContext) {
	h.sessionManager.SetData(userID, sessionKeyResultTexts, sc.texts)
	h.sessionManager.SetData(userID, sessionKeyResultEventID, sc.event.ID)
	h.sessionManager.SetData(userID, sessionKeyResultParticipant, sc.participant.ID)
}

// submit выполняет команду отправки результата и формирует ответ пользователю.
// При успехе сбрасывает сессию и возвращает участника с привязанным результатом.
func (h *ResultHandler) submit(ctx context.Context, userID int64, sc submissionContext, resultLink string) (string, *entity.Participant) {
	cmd := command.SubmitResultCommand{
		ParticipantID: sc.participant.ID,
		ResultLink:    resultLink,
	}

	participant, err := h.submitResultHandler.Handle(ctx, cmd)
	if err != nil {
		if errors.Is(err, command.ErrInvalidResultLink) {
			log.Printf(
				"INFO Invalid result submission attempt: user_id=%d participant_id=%d event_id=%d reason=invalid_strava_format",
				userID,
				sc.participant.ID,
				sc.event.ID,
			)
			return ResultLinkInvalidInputText(sc.texts), nil
		}

		log.Printf("Error submitting result: user_id=%d participant_id=%d error=%v", userID, sc.participant.ID, err)
		return fmt.Sprintf("Ошибка при отправке результата: %v", err), nil
	}

	h.sessionManager.ResetState(userID)

	return applyResultTextPlaceholders(
		sc.texts.ResultSuccess,
		map[string]string{"result_link": participant.GetResultLink()},
	), participant
}

// StartSubmitResult начинает процесс отправки результата. Участник с уже
// отправленным результатом тоже допускается — ссылку он подтвердит на шаге
// замены (см. SubmitOrConfirm), поэтому здесь статус финиша не блокирует сценарий.
func (h *ResultHandler) StartSubmitResult(ctx context.Context, userID int64) (string, *models.InlineKeyboardMarkup) {
	sc, errText, ok := h.resolveSubmission(ctx, userID)
	if !ok {
		return errText, nil
	}

	// Сохраняем данные сценария в сессии
	h.storeSubmissionSession(userID, sc)
	h.sessionManager.SetState(userID, session.StateAwaitingResultLink)

	keyboard := keyboard.CancelMenu()

	return ResultLinkPromptText(sc.texts), &keyboard
}

// SubmitOrConfirm обрабатывает ссылку Strava, присланную пользователем напрямую
// (вне сценария «Я уже проехал»). Если у участника уже есть результат — возвращает
// запрос подтверждения замены с клавиатурой (participant == nil). Иначе сразу
// сохраняет результат и возвращает участника для уведомления админов.
func (h *ResultHandler) SubmitOrConfirm(ctx context.Context, userID int64, resultLink string) (string, *models.InlineKeyboardMarkup, *entity.Participant) {
	sc, errText, ok := h.resolveSubmission(ctx, userID)
	if !ok {
		h.sessionManager.ResetState(userID)
		return errText, nil, nil
	}

	// Допускаем сообщения, где ссылка обёрнута текстом или прислана без схемы
	// https:// — извлекаем и нормализуем её перед сохранением.
	link, ok := valueobject.ExtractResultLink(resultLink)
	if !ok {
		log.Printf(
			"INFO Invalid result submission attempt: user_id=%d participant_id=%d event_id=%d reason=invalid_strava_format",
			userID,
			sc.participant.ID,
			sc.event.ID,
		)
		return ResultLinkInvalidInputText(sc.texts), nil, nil
	}
	resultLink = link.String()

	// Результат уже есть — запрашиваем подтверждение замены.
	if sc.participant.IsFinished() {
		h.storeSubmissionSession(userID, sc)
		h.sessionManager.SetData(userID, sessionKeyPendingResultLink, resultLink)
		h.sessionManager.SetState(userID, session.StateAwaitingResultReplaceConfirmation)

		markup := keyboard.ResultReplaceConfirmMenu()
		text := applyResultTextPlaceholders(resultReplaceConfirmText, map[string]string{
			"current_link": sc.participant.GetResultLink(),
			"new_link":     resultLink,
		})
		return text, &markup, nil
	}

	text, participant := h.submit(ctx, userID, sc, resultLink)
	return text, nil, participant
}

// ConfirmReplacement подтверждает замену ранее отправленного результата на ссылку,
// сохранённую в сессии. Возвращает участника при успешной замене.
func (h *ResultHandler) ConfirmReplacement(ctx context.Context, userID int64) (string, *entity.Participant) {
	defer h.sessionManager.ResetState(userID)

	linkRaw, ok := h.sessionManager.GetData(userID, sessionKeyPendingResultLink)
	if !ok {
		return resultReplaceStaleText, nil
	}
	link, ok := linkRaw.(string)
	if !ok || strings.TrimSpace(link) == "" {
		log.Printf("WARN Invalid result session data: user_id=%d key=%s type=%T", userID, sessionKeyPendingResultLink, linkRaw)
		return resultReplaceStaleText, nil
	}

	sc, errText, ok := h.resolveSubmission(ctx, userID)
	if !ok {
		return errText, nil
	}

	text, participant := h.submit(ctx, userID, sc, strings.TrimSpace(link))
	return text, participant
}

// CancelReplacement отменяет замену результата.
func (h *ResultHandler) CancelReplacement(userID int64) string {
	h.sessionManager.ResetState(userID)
	return resultReplaceCancelText
}

func applyResultTextPlaceholders(text string, values map[string]string) string {
	for key, value := range values {
		text = strings.ReplaceAll(text, "{"+key+"}", value)
	}
	return text
}

// CancelSubmitResult отменяет отправку результата
func (h *ResultHandler) CancelSubmitResult(userID int64) string {
	h.sessionManager.ResetState(userID)
	return "Отправка результата отменена."
}

func isNoActiveEventError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no active event")
}
