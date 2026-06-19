package command

import (
	"context"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// ErrResultNotFound (объявлен в add_result_criteria.go) возвращается, когда
// обновляемый результат не найден.

// UpdateResultCommand представляет команду обновления результата администратором.
// Семантика — полная замена метрик результата значениями из запроса (отсутствующее
// поле очищается). Общее время определяется как в create: из старт/финиш, иначе из ElapsedTimeSec.
type UpdateResultCommand struct {
	ResultID       uint
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

// UpdateResultHandler обрабатывает обновление результата с валидацией и пересчётом.
type UpdateResultHandler struct {
	resultRepo repository.ResultRepository
}

// NewUpdateResultHandler создаёт handler обновления результата.
func NewUpdateResultHandler(resultRepo repository.ResultRepository) *UpdateResultHandler {
	return &UpdateResultHandler{resultRepo: resultRepo}
}

// Handle выполняет обновление результата.
func (h *UpdateResultHandler) Handle(ctx context.Context, cmd UpdateResultCommand) (*entity.Result, error) {
	result, err := h.resultRepo.FindByID(ctx, cmd.ResultID)
	if err != nil {
		log.Printf("ERROR Result update failed: result_id=%d stage=find error=%v", cmd.ResultID, err)
		return nil, fmt.Errorf("failed to load result: %w", err)
	}
	if result == nil {
		log.Printf("WARN Result update rejected: result_id=%d error_class=result_not_found", cmd.ResultID)
		return nil, ErrResultNotFound
	}

	elapsed, reason, ok := resolveResultElapsed(cmd.StartedAt, cmd.FinishedAt, cmd.ElapsedTimeSec)
	if !ok {
		log.Printf("WARN Result update rejected: result_id=%d stage=resolve_elapsed reason=%s", cmd.ResultID, reason)
		return nil, ErrInvalidResultTime
	}
	if elapsed <= 0 {
		log.Printf("WARN Result update rejected: result_id=%d stage=resolve_elapsed reason=non_positive", cmd.ResultID)
		return nil, ErrInvalidResultTime
	}
	if reason, ok := validateResultMetrics(elapsed, cmd.MovingTimeSec, cmd.DistanceMeters, cmd.AvgHeartRate, cmd.MaxHeartRate, cmd.AvgCadence, cmd.Calories, cmd.PeakSpeedKmh); !ok {
		log.Printf("WARN Result update rejected: result_id=%d stage=validate_metrics reason=%s", cmd.ResultID, reason)
		if reason == "negative_metric" {
			return nil, ErrInvalidResultMetric
		}
		return nil, ErrInvalidResultTime
	}

	elapsedTimeSec := elapsed
	result.ElapsedTimeSec = &elapsedTimeSec
	result.MovingTimeSec = cmd.MovingTimeSec
	result.StartedAt = cmd.StartedAt
	result.FinishedAt = cmd.FinishedAt
	result.DistanceMeters = cmd.DistanceMeters
	result.AvgHeartRate = cmd.AvgHeartRate
	result.MaxHeartRate = cmd.MaxHeartRate
	result.PeakSpeedKmh = cmd.PeakSpeedKmh
	result.AvgCadence = cmd.AvgCadence
	result.Calories = cmd.Calories

	if err := h.resultRepo.UpdateMetrics(ctx, result); err != nil {
		log.Printf("ERROR Result update failed: result_id=%d stage=persist error=%v", cmd.ResultID, err)
		return nil, fmt.Errorf("failed to update result: %w", err)
	}

	log.Printf("INFO Result update completed: result_id=%d elapsed_time_sec=%d moving_time_present=%t", cmd.ResultID, elapsed, result.MovingTimeSec != nil)
	return result, nil
}
