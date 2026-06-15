package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
)

func TestCreateManualResultHandlerCreatesCurrentResultWithoutStravaLink(t *testing.T) {
	now := testMinskNow()
	participant := &entity.Participant{ID: 11, EventID: 77}
	resultRepo := &submitResultRepoFake{}
	h := NewCreateManualResultHandler(
		&submitParticipantRepoFake{participant: participant},
		resultRepo,
		WithCreateManualResultClock(func() time.Time { return now }),
	)

	result, err := h.Handle(context.Background(), CreateManualResultCommand{
		ParticipantID:  participant.ID,
		ElapsedTimeSec: 3600,
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if resultRepo.created == nil {
		t.Fatal("result was not created")
	}
	if result != resultRepo.created {
		t.Fatalf("result mismatch: got %#v want %#v", result, resultRepo.created)
	}
	if result.ResultLink != nil {
		t.Fatalf("result link should be nil, got %#v", result.ResultLink)
	}
	if result.ElapsedTimeSec == nil || *result.ElapsedTimeSec != 3600 {
		t.Fatalf("elapsed_time_sec mismatch: got %#v want 3600", result.ElapsedTimeSec)
	}
	if !result.IsCurrent {
		t.Fatal("created result should be current")
	}
	if result.SubmittedAt != now {
		t.Fatalf("submitted_at mismatch: got %s want %s", result.SubmittedAt, now)
	}
}

func TestCreateManualResultHandlerRejectsInvalidTime(t *testing.T) {
	tests := []struct {
		name    string
		elapsed int
		moving  *int
	}{
		{name: "missing elapsed", elapsed: 0},
		{name: "negative moving", elapsed: 3600, moving: intPtr(-1)},
		{name: "moving greater than elapsed", elapsed: 3600, moving: intPtr(3601)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultRepo := &submitResultRepoFake{}
			h := NewCreateManualResultHandler(
				&submitParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
				resultRepo,
			)

			_, err := h.Handle(context.Background(), CreateManualResultCommand{
				ParticipantID:  11,
				ElapsedTimeSec: tt.elapsed,
				MovingTimeSec:  tt.moving,
			})
			if !errors.Is(err, ErrInvalidResultTime) {
				t.Fatalf("error mismatch: got %v, want %v", err, ErrInvalidResultTime)
			}
			if resultRepo.created != nil {
				t.Fatal("result should not be created for invalid time")
			}
		})
	}
}

func TestCreateManualResultHandlerRejectsDuplicateCurrentResult(t *testing.T) {
	resultRepo := &submitResultRepoFake{}
	h := NewCreateManualResultHandler(
		&submitParticipantRepoFake{
			participant: &entity.Participant{
				ID:      11,
				EventID: 77,
				Result:  &entity.Result{ID: 99, IsCurrent: true},
			},
		},
		resultRepo,
	)

	_, err := h.Handle(context.Background(), CreateManualResultCommand{
		ParticipantID:  11,
		ElapsedTimeSec: 3600,
	})
	if !errors.Is(err, ErrResultAlreadyExists) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrResultAlreadyExists)
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created when current result exists")
	}
}

func TestCreateManualResultHandlerRejectsInvalidOptionalResultLink(t *testing.T) {
	resultRepo := &submitResultRepoFake{}
	h := NewCreateManualResultHandler(
		&submitParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
		resultRepo,
	)

	_, err := h.Handle(context.Background(), CreateManualResultCommand{
		ParticipantID:  11,
		ElapsedTimeSec: 3600,
		ResultLink:     "https://www.komoot.com/tour/2308024419",
	})
	if !errors.Is(err, ErrInvalidResultLink) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrInvalidResultLink)
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created for invalid optional link")
	}
}

func intPtr(value int) *int {
	return &value
}
