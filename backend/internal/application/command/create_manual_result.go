package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var ErrInvalidResultTime = errors.New("invalid result time")

// CreateManualResultCommand представляет команду ручного добавления результата администратором.
type CreateManualResultCommand struct {
	ParticipantID  uint
	ResultLink     string
	ElapsedTimeSec int
	MovingTimeSec  *int
}

// CreateManualResultHandler обрабатывает ручное добавление результата.
type CreateManualResultHandler struct {
	participantRepo repository.ParticipantRepository
	resultRepo      repository.ResultRepository
	now             func() time.Time
}

// CreateManualResultHandlerOption настраивает handler ручного результата.
type CreateManualResultHandlerOption func(*CreateManualResultHandler)

// WithCreateManualResultClock задаёт источник текущего времени для тестов.
func WithCreateManualResultClock(now func() time.Time) CreateManualResultHandlerOption {
	return func(h *CreateManualResultHandler) {
		if now != nil {
			h.now = now
		}
	}
}

// NewCreateManualResultHandler создаёт handler ручного результата.
func NewCreateManualResultHandler(
	participantRepo repository.ParticipantRepository,
	resultRepo repository.ResultRepository,
	options ...CreateManualResultHandlerOption,
) *CreateManualResultHandler {
	handler := &CreateManualResultHandler{
		participantRepo: participantRepo,
		resultRepo:      resultRepo,
		now:             time.Now,
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

// Handle выполняет ручное добавление результата.
func (h *CreateManualResultHandler) Handle(ctx context.Context, cmd CreateManualResultCommand) (*entity.Result, error) {
	log.Printf(
		"INFO Manual result creation requested: participant_id=%d elapsed_time_sec=%d moving_time_present=%t result_link_present=%t",
		cmd.ParticipantID,
		cmd.ElapsedTimeSec,
		cmd.MovingTimeSec != nil,
		strings.TrimSpace(cmd.ResultLink) != "",
	)

	if cmd.ElapsedTimeSec <= 0 {
		log.Printf("WARN Manual result creation failed: participant_id=%d stage=validate_elapsed_time reason=non_positive", cmd.ParticipantID)
		return nil, ErrInvalidResultTime
	}
	if cmd.MovingTimeSec != nil {
		if *cmd.MovingTimeSec < 0 {
			log.Printf("WARN Manual result creation failed: participant_id=%d stage=validate_moving_time reason=negative", cmd.ParticipantID)
			return nil, ErrInvalidResultTime
		}
		if *cmd.MovingTimeSec > cmd.ElapsedTimeSec {
			log.Printf("WARN Manual result creation failed: participant_id=%d stage=validate_moving_time reason=greater_than_elapsed", cmd.ParticipantID)
			return nil, ErrInvalidResultTime
		}
	}

	participant, err := h.participantRepo.FindByID(ctx, cmd.ParticipantID)
	if err != nil {
		log.Printf("WARN Manual result creation failed: participant_id=%d stage=find_participant error=%v", cmd.ParticipantID, err)
		return nil, ErrParticipantNotFound
	}
	if participant == nil {
		log.Printf("WARN Manual result creation failed: participant_id=%d stage=find_participant reason=nil_participant", cmd.ParticipantID)
		return nil, ErrParticipantNotFound
	}
	if participant.Result != nil && participant.Result.IsCurrent {
		log.Printf("WARN Manual result creation failed: participant_id=%d stage=check_current_result result_id=%d reason=already_exists", cmd.ParticipantID, participant.Result.ID)
		return nil, ErrResultAlreadyExists
	}

	var resultLink *valueobject.ResultLink
	if trimmedLink := strings.TrimSpace(cmd.ResultLink); trimmedLink != "" {
		parsedLink, err := valueobject.NewResultLink(trimmedLink)
		if err != nil {
			log.Printf("WARN Manual result creation failed: participant_id=%d stage=validate_result_link reason=invalid", cmd.ParticipantID)
			return nil, ErrInvalidResultLink
		}
		resultLink = parsedLink
	}

	elapsedTimeSec := cmd.ElapsedTimeSec
	result := &entity.Result{
		ParticipantID:  participant.ID,
		ResultLink:     resultLink,
		ElapsedTimeSec: &elapsedTimeSec,
		MovingTimeSec:  cmd.MovingTimeSec,
		IsCurrent:      true,
		SubmittedAt:    h.now(),
	}

	if err := h.resultRepo.Create(ctx, result); err != nil {
		log.Printf("ERROR Manual result creation failed: participant_id=%d stage=create_result error=%v", cmd.ParticipantID, err)
		return nil, fmt.Errorf("failed to create manual result: %w", err)
	}

	log.Printf("INFO Manual result creation completed: participant_id=%d result_id=%d elapsed_time_sec=%d moving_time_present=%t result_link_present=%t", cmd.ParticipantID, result.ID, elapsedTimeSec, result.MovingTimeSec != nil, result.ResultLink != nil)
	return result, nil
}
