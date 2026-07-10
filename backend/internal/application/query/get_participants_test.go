package query

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

// При равном общем времени места определяются по чистому времени (время в
// движении): меньше чистое время — выше место.
func TestGetParticipantsTieBreaksEqualElapsedByMovingTime(t *testing.T) {
	const elapsed = 37440 // 10:24:00 — одинаковое у всех троих

	participants := []*entity.Participant{
		makeFinisher(1, "Alexey", valueobject.GenderMale, elapsed, 36547),       // 10:09:07
		makeFinisher(2, "Anastasiya", valueobject.GenderFemale, elapsed, 35872), // 09:57:52
		makeFinisher(3, "Zmiter", valueobject.GenderMale, elapsed, 35626),       // 09:53:46
	}

	handler := NewGetParticipantsHandler(&tieBreakParticipantRepoFake{participants: participants})

	result, err := handler.Handle(context.Background(), GetParticipantsQuery{EventID: 1})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(result))
	}

	// Меньше чистое время — выше место: Zmiter(35626) < Anastasiya(35872) < Alexey(36547).
	wantOrder := []struct {
		id    uint
		place int
	}{
		{3, 1},
		{2, 2},
		{1, 3},
	}
	for i, want := range wantOrder {
		got := result[i]
		if got.Participant.ID != want.id || got.Place != want.place {
			t.Fatalf("position %d: got participant %d place %d, want participant %d place %d",
				i, got.Participant.ID, got.Place, want.id, want.place)
		}
	}
}

// Отсутствующее чистое время при равном общем уходит в конец зачёта.
func TestGetParticipantsTieBreakPutsMissingMovingTimeLast(t *testing.T) {
	const elapsed = 37440

	withMoving := makeFinisher(1, "WithMoving", valueobject.GenderMale, elapsed, 35626)
	noMoving := makeFinisher(2, "NoMoving", valueobject.GenderMale, elapsed, 0)
	noMoving.Result.MovingTimeSec = nil // чистого времени нет

	handler := NewGetParticipantsHandler(&tieBreakParticipantRepoFake{
		participants: []*entity.Participant{noMoving, withMoving},
	})

	result, err := handler.Handle(context.Background(), GetParticipantsQuery{EventID: 1})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(result))
	}
	if result[0].Participant.ID != 1 || result[0].Place != 1 {
		t.Fatalf("participant with moving time must rank first, got %#v", result[0])
	}
	if result[1].Participant.ID != 2 || result[1].Place != 2 {
		t.Fatalf("participant without moving time must rank last, got %#v", result[1])
	}
}

func TestGetParticipantByUserAndEventReturnsOnlyRequestedParticipant(t *testing.T) {
	participants := []*entity.Participant{
		{ID: 1, UserID: 11, EventID: 77},
		{ID: 2, UserID: 22, EventID: 77},
	}
	handler := NewGetParticipantByUserAndEventHandler(
		&tieBreakParticipantRepoFake{participants: participants},
	)

	participant, err := handler.Handle(context.Background(), GetParticipantByUserAndEventQuery{
		UserID:  22,
		EventID: 77,
	})

	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if participant == nil || participant.ID != 2 {
		t.Fatalf("participant mismatch: %#v", participant)
	}
}

func makeFinisher(id uint, name string, gender valueobject.Gender, elapsedSec, movingSec int) *entity.Participant {
	elapsed := elapsedSec
	moving := movingSec
	return &entity.Participant{
		ID:       id,
		UserID:   int64(id),
		EventID:  1,
		Gender:   gender,
		BikeType: valueobject.BikeTypeMTB,
		Status:   valueobject.ParticipantStatusActive,
		User:     &entity.User{ID: int64(id), FirstName: name},
		Result: &entity.Result{
			ID:             id,
			ParticipantID:  id,
			IsCurrent:      true,
			ElapsedTimeSec: &elapsed,
			MovingTimeSec:  &moving,
		},
	}
}

type tieBreakParticipantRepoFake struct {
	participants []*entity.Participant
}

func (r *tieBreakParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *tieBreakParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *tieBreakParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *tieBreakParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	for _, participant := range r.participants {
		if participant.UserID == userID && participant.EventID == eventID {
			return participant, nil
		}
	}
	return nil, nil
}
func (r *tieBreakParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return r.participants, nil
}
func (r *tieBreakParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *tieBreakParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *tieBreakParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *tieBreakParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return r.participants, nil
}
