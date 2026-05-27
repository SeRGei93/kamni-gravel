package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var (
	ErrInvalidResultLink       = errors.New("invalid result link")
	ErrResultAlreadyExists     = errors.New("result already submitted")
	ErrResultSubmissionNotOpen = errors.New("result submission is not open")
	ErrEventStartNotConfigured = errors.New("event start time is not configured")
	ErrEventNotStarted         = errors.New("event has not started")
)

// SubmitResultCommand представляет команду отправки результата
type SubmitResultCommand struct {
	ParticipantID uint
	ResultLink    string
}

// SubmitResultHandler обрабатывает отправку результата
type SubmitResultHandler struct {
	participantRepo repository.ParticipantRepository
	eventRepo       repository.EventRepository
	resultRepo      repository.ResultRepository
	now             func() time.Time
}

// SubmitResultHandlerOption настраивает handler отправки результата.
type SubmitResultHandlerOption func(*SubmitResultHandler)

// WithSubmitResultClock задаёт источник текущего времени для тестов.
func WithSubmitResultClock(now func() time.Time) SubmitResultHandlerOption {
	return func(h *SubmitResultHandler) {
		if now != nil {
			h.now = now
		}
	}
}

// NewSubmitResultHandler создаёт новый handler
func NewSubmitResultHandler(
	participantRepo repository.ParticipantRepository,
	eventRepo repository.EventRepository,
	resultRepo repository.ResultRepository,
	options ...SubmitResultHandlerOption,
) *SubmitResultHandler {
	handler := &SubmitResultHandler{
		participantRepo: participantRepo,
		eventRepo:       eventRepo,
		resultRepo:      resultRepo,
		now:             time.Now,
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

// Handle выполняет команду отправки результата
func (h *SubmitResultHandler) Handle(ctx context.Context, cmd SubmitResultCommand) (*entity.Participant, error) {
	// Находим участника
	participant, err := h.participantRepo.FindByID(ctx, cmd.ParticipantID)
	if err != nil {
		return nil, ErrParticipantNotFound
	}

	event, err := h.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil || event == nil {
		log.Printf("INFO Result submission blocked: participant_id=%d event_id=%d reason=event_not_found", participant.ID, participant.EventID)
		return nil, ErrEventNotFound
	}

	if !event.Active {
		log.Printf("INFO Result submission blocked: participant_id=%d event_id=%d reason=event_inactive", participant.ID, event.ID)
		return nil, ErrResultSubmissionNotOpen
	}

	now := h.now().In(valueobject.MinskLocation())
	startTime, ok := event.SubmissionStartTimeInMinsk()
	if !ok {
		log.Printf(
			"INFO Result submission blocked: participant_id=%d event_id=%d current_minsk_time=%q reason=start_not_configured",
			participant.ID,
			event.ID,
			valueobject.FormatMinskDateTime(now),
		)
		return nil, ErrEventStartNotConfigured
	}

	if !event.HasStartedAt(now) {
		log.Printf(
			"INFO Result submission blocked: participant_id=%d event_id=%d current_minsk_time=%q start_minsk_time=%q reason=event_not_started",
			participant.ID,
			event.ID,
			valueobject.FormatMinskDateTime(now),
			valueobject.FormatMinskDateTime(startTime),
		)
		return nil, ErrEventNotStarted
	}

	// Валидируем ссылку на результат
	resultLink, err := valueobject.NewResultLink(cmd.ResultLink)
	if err != nil {
		return nil, ErrInvalidResultLink
	}

	// Создаём новый результат
	result := &entity.Result{
		ParticipantID: participant.ID,
		ResultLink:    resultLink,
		IsCurrent:     true,
		SubmittedAt:   h.now(),
	}

	// Сохраняем результат (он автоматически пометит старые как неактуальные)
	if err := h.resultRepo.Create(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to create result: %w", err)
	}

	// Привязываем результат к участнику для возврата
	participant.Result = result

	return participant, nil
}
