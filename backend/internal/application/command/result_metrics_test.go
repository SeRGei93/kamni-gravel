package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
)

func TestCreateManualResultComputesTotalFromStartFinish(t *testing.T) {
	now := testMinskNow()
	resultRepo := &submitResultRepoFake{}
	h := NewCreateManualResultHandler(
		&submitParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
		resultRepo,
		WithCreateManualResultClock(func() time.Time { return now }),
	)

	start := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)
	finish := start.Add(6*time.Hour + 15*time.Minute)

	result, err := h.Handle(context.Background(), CreateManualResultCommand{
		ParticipantID:  11,
		StartedAt:      &start,
		FinishedAt:     &finish,
		MovingTimeSec:  intPtr(18000),
		DistanceMeters: intPtr(202000),
		AvgHeartRate:   intPtr(140),
		PeakSpeedKmh:   float64Ptr(51.4),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.ElapsedTimeSec == nil || *result.ElapsedTimeSec != 6*3600+15*60 {
		t.Fatalf("computed elapsed mismatch: got %v want %d", result.ElapsedTimeSec, 6*3600+15*60)
	}
	if result.DistanceMeters == nil || *result.DistanceMeters != 202000 {
		t.Fatalf("distance not persisted: %v", result.DistanceMeters)
	}
	if result.AvgSpeedKmh() == nil {
		t.Fatal("avg speed should be computable from distance and total")
	}
}

func TestCreateManualResultRejectsFinishBeforeStart(t *testing.T) {
	resultRepo := &submitResultRepoFake{}
	h := NewCreateManualResultHandler(
		&submitParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
		resultRepo,
	)

	start := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	finish := start.Add(-time.Hour) // финиш раньше старта

	_, err := h.Handle(context.Background(), CreateManualResultCommand{
		ParticipantID: 11,
		StartedAt:     &start,
		FinishedAt:    &finish,
	})
	if !errors.Is(err, ErrInvalidResultTime) {
		t.Fatalf("error mismatch: got %v want %v", err, ErrInvalidResultTime)
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created when finish precedes start")
	}
}

func TestCreateManualResultRejectsNegativeMetric(t *testing.T) {
	resultRepo := &submitResultRepoFake{}
	h := NewCreateManualResultHandler(
		&submitParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
		resultRepo,
	)

	_, err := h.Handle(context.Background(), CreateManualResultCommand{
		ParticipantID:  11,
		ElapsedTimeSec: intPtr(3600),
		DistanceMeters: intPtr(-5),
	})
	if !errors.Is(err, ErrInvalidResultMetric) {
		t.Fatalf("error mismatch: got %v want %v", err, ErrInvalidResultMetric)
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created for negative metric")
	}
}

func TestUpdateResultHandlerRecomputesAndPersists(t *testing.T) {
	existing := &entity.Result{ID: 55, ParticipantID: 11, IsCurrent: true, ElapsedTimeSec: intPtr(3600)}
	resultRepo := &submitResultRepoFake{stored: existing}
	h := NewUpdateResultHandler(resultRepo)

	start := time.Date(2025, 6, 15, 8, 0, 0, 0, time.UTC)
	finish := start.Add(5 * time.Hour)

	result, err := h.Handle(context.Background(), UpdateResultCommand{
		ResultID:       55,
		StartedAt:      &start,
		FinishedAt:     &finish,
		MovingTimeSec:  intPtr(16000),
		DistanceMeters: intPtr(202000),
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.ElapsedTimeSec == nil || *result.ElapsedTimeSec != 5*3600 {
		t.Fatalf("elapsed not recomputed: got %v want %d", result.ElapsedTimeSec, 5*3600)
	}
	if resultRepo.updated == nil {
		t.Fatal("UpdateMetrics was not called")
	}
	if resultRepo.updated.DistanceMeters == nil || *resultRepo.updated.DistanceMeters != 202000 {
		t.Fatalf("distance not persisted on update: %v", resultRepo.updated.DistanceMeters)
	}
}

func TestUpdateResultHandlerRejectsMovingGreaterThanTotal(t *testing.T) {
	existing := &entity.Result{ID: 55, ParticipantID: 11, IsCurrent: true}
	resultRepo := &submitResultRepoFake{stored: existing}
	h := NewUpdateResultHandler(resultRepo)

	_, err := h.Handle(context.Background(), UpdateResultCommand{
		ResultID:       55,
		ElapsedTimeSec: intPtr(3600),
		MovingTimeSec:  intPtr(4000),
	})
	if !errors.Is(err, ErrInvalidResultTime) {
		t.Fatalf("error mismatch: got %v want %v", err, ErrInvalidResultTime)
	}
	if resultRepo.updated != nil {
		t.Fatal("result should not be persisted when moving exceeds total")
	}
}

func TestUpdateResultHandlerNotFound(t *testing.T) {
	resultRepo := &submitResultRepoFake{} // stored == nil
	h := NewUpdateResultHandler(resultRepo)

	_, err := h.Handle(context.Background(), UpdateResultCommand{
		ResultID:       404,
		ElapsedTimeSec: intPtr(3600),
	})
	if !errors.Is(err, ErrResultNotFound) {
		t.Fatalf("error mismatch: got %v want %v", err, ErrResultNotFound)
	}
}

func float64Ptr(v float64) *float64 { return &v }
