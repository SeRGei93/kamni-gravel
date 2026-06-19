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

var (
	ErrInvalidResultTime   = errors.New("invalid result time")
	ErrInvalidResultMetric = errors.New("invalid result metric")
)

// CreateManualResultCommand представляет команду ручного добавления результата администратором.
// Все поля метрик опциональны. Общее время берётся из старт/финиш, если оба заданы,
// иначе из ElapsedTimeSec.
type CreateManualResultCommand struct {
	ParticipantID  uint
	ResultLink     string
	ElapsedTimeSec *int
	MovingTimeSec  *int

	StartedAt      *time.Time
	FinishedAt     *time.Time
	DistanceMeters *int
	AvgHeartRate   *int
	MaxHeartRate   *int
	PeakSpeedKmh   *float64
	AvgCadence     *int
	Calories       *int
}

// resolveResultElapsed определяет общее время результата: разница финиша и старта,
// если оба заданы; иначе переданное elapsed. Возвращает (секунды, причина_отказа, ok).
func resolveResultElapsed(startedAt, finishedAt *time.Time, elapsedTimeSec *int) (int, string, bool) {
	if startedAt != nil && finishedAt != nil {
		if !finishedAt.After(*startedAt) {
			return 0, "invalid_time_order", false
		}
		return int(finishedAt.Sub(*startedAt).Seconds()), "", true
	}
	if elapsedTimeSec != nil {
		return *elapsedTimeSec, "", true
	}
	return 0, "missing_total", false
}

// validateResultMetrics проверяет чистое время и числовые метрики относительно
// общего времени. Возвращает (причина_отказа, ok).
func validateResultMetrics(elapsed int, movingTimeSec, distanceMeters, avgHeartRate, maxHeartRate, avgCadence, calories *int, peakSpeedKmh *float64) (string, bool) {
	if movingTimeSec != nil {
		if *movingTimeSec < 0 {
			return "moving_negative", false
		}
		if *movingTimeSec > elapsed {
			return "moving_gt_total", false
		}
	}
	for _, v := range []*int{distanceMeters, avgHeartRate, maxHeartRate, avgCadence, calories} {
		if v != nil && *v < 0 {
			return "negative_metric", false
		}
	}
	if peakSpeedKmh != nil && *peakSpeedKmh < 0 {
		return "negative_metric", false
	}
	return "", true
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
		"INFO Manual result creation requested: participant_id=%d start_finish_present=%t elapsed_present=%t moving_time_present=%t result_link_present=%t",
		cmd.ParticipantID,
		cmd.StartedAt != nil && cmd.FinishedAt != nil,
		cmd.ElapsedTimeSec != nil,
		cmd.MovingTimeSec != nil,
		strings.TrimSpace(cmd.ResultLink) != "",
	)

	elapsed, reason, ok := resolveResultElapsed(cmd.StartedAt, cmd.FinishedAt, cmd.ElapsedTimeSec)
	if !ok {
		log.Printf("WARN Manual result creation failed: participant_id=%d stage=resolve_elapsed reason=%s", cmd.ParticipantID, reason)
		return nil, ErrInvalidResultTime
	}
	if elapsed <= 0 {
		log.Printf("WARN Manual result creation failed: participant_id=%d stage=resolve_elapsed reason=non_positive", cmd.ParticipantID)
		return nil, ErrInvalidResultTime
	}
	if reason, ok := validateResultMetrics(elapsed, cmd.MovingTimeSec, cmd.DistanceMeters, cmd.AvgHeartRate, cmd.MaxHeartRate, cmd.AvgCadence, cmd.Calories, cmd.PeakSpeedKmh); !ok {
		log.Printf("WARN Manual result creation failed: participant_id=%d stage=validate_metrics reason=%s", cmd.ParticipantID, reason)
		if reason == "negative_metric" {
			return nil, ErrInvalidResultMetric
		}
		return nil, ErrInvalidResultTime
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

	elapsedTimeSec := elapsed
	result := &entity.Result{
		ParticipantID:  participant.ID,
		ResultLink:     resultLink,
		ElapsedTimeSec: &elapsedTimeSec,
		MovingTimeSec:  cmd.MovingTimeSec,
		IsCurrent:      true,
		SubmittedAt:    h.now(),
		StartedAt:      cmd.StartedAt,
		FinishedAt:     cmd.FinishedAt,
		DistanceMeters: cmd.DistanceMeters,
		AvgHeartRate:   cmd.AvgHeartRate,
		MaxHeartRate:   cmd.MaxHeartRate,
		PeakSpeedKmh:   cmd.PeakSpeedKmh,
		AvgCadence:     cmd.AvgCadence,
		Calories:       cmd.Calories,
	}

	if err := h.resultRepo.Create(ctx, result); err != nil {
		log.Printf("ERROR Manual result creation failed: participant_id=%d stage=create_result error=%v", cmd.ParticipantID, err)
		return nil, fmt.Errorf("failed to create manual result: %w", err)
	}

	log.Printf("INFO Manual result creation completed: participant_id=%d result_id=%d elapsed_time_sec=%d moving_time_present=%t result_link_present=%t", cmd.ParticipantID, result.ID, elapsedTimeSec, result.MovingTimeSec != nil, result.ResultLink != nil)
	return result, nil
}
