package command

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

type updateParticipantRepoFake struct {
	repository.ParticipantRepository
	stored  *entity.Participant
	updated *entity.Participant
}

func (f *updateParticipantRepoFake) FindByID(_ context.Context, _ uint) (*entity.Participant, error) {
	return f.stored, nil
}

func (f *updateParticipantRepoFake) Update(_ context.Context, p *entity.Participant) error {
	f.updated = p
	return nil
}

func TestUpdateParticipantSetsStatus(t *testing.T) {
	repo := &updateParticipantRepoFake{stored: &entity.Participant{ID: 1, Status: valueobject.ParticipantStatusActive}}
	h := NewUpdateParticipantHandler(repo)

	dnf := "dnf"
	got, err := h.Handle(context.Background(), UpdateParticipantCommand{ParticipantID: 1, Status: &dnf})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Status != valueobject.ParticipantStatusDNF {
		t.Fatalf("returned status = %q, want dnf", got.Status)
	}
	if repo.updated == nil || repo.updated.Status != valueobject.ParticipantStatusDNF {
		t.Fatalf("persisted status not dnf: %+v", repo.updated)
	}
}

func TestUpdateParticipantRejectsInvalidStatus(t *testing.T) {
	repo := &updateParticipantRepoFake{stored: &entity.Participant{ID: 1}}
	h := NewUpdateParticipantHandler(repo)

	bad := "retired"
	_, err := h.Handle(context.Background(), UpdateParticipantCommand{ParticipantID: 1, Status: &bad})
	if err != ErrInvalidStatus {
		t.Fatalf("err = %v, want ErrInvalidStatus", err)
	}
	if repo.updated != nil {
		t.Fatalf("must not persist on invalid status")
	}
}

func TestUpdateParticipantSetsPrevElapsedTime(t *testing.T) {
	repo := &updateParticipantRepoFake{stored: &entity.Participant{ID: 1}}
	h := NewUpdateParticipantHandler(repo)

	sec := 25500
	got, err := h.Handle(context.Background(), UpdateParticipantCommand{ParticipantID: 1, PrevElapsedTimeSec: &sec})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.PrevElapsedTimeSec == nil || *got.PrevElapsedTimeSec != sec {
		t.Fatalf("prev elapsed = %v, want %d", got.PrevElapsedTimeSec, sec)
	}
	if repo.updated == nil || repo.updated.PrevElapsedTimeSec == nil || *repo.updated.PrevElapsedTimeSec != sec {
		t.Fatalf("persisted prev elapsed mismatch: %+v", repo.updated)
	}
}

func TestUpdateParticipantClearsPrevElapsedTimeOnZero(t *testing.T) {
	existing := 25500
	repo := &updateParticipantRepoFake{stored: &entity.Participant{ID: 1, PrevElapsedTimeSec: &existing}}
	h := NewUpdateParticipantHandler(repo)

	zero := 0
	got, err := h.Handle(context.Background(), UpdateParticipantCommand{ParticipantID: 1, PrevElapsedTimeSec: &zero})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.PrevElapsedTimeSec != nil {
		t.Fatalf("prev elapsed should be cleared, got %v", *got.PrevElapsedTimeSec)
	}
}

func TestUpdateParticipantRejectsNegativePrevElapsedTime(t *testing.T) {
	repo := &updateParticipantRepoFake{stored: &entity.Participant{ID: 1}}
	h := NewUpdateParticipantHandler(repo)

	neg := -60
	_, err := h.Handle(context.Background(), UpdateParticipantCommand{ParticipantID: 1, PrevElapsedTimeSec: &neg})
	if err != ErrInvalidPrevElapsedTime {
		t.Fatalf("err = %v, want ErrInvalidPrevElapsedTime", err)
	}
	if repo.updated != nil {
		t.Fatalf("must not persist on invalid prev elapsed time")
	}
}

func TestUpdateParticipantLeavesPrevElapsedTimeWhenNotProvided(t *testing.T) {
	existing := 25500
	repo := &updateParticipantRepoFake{stored: &entity.Participant{ID: 1, PrevElapsedTimeSec: &existing}}
	h := NewUpdateParticipantHandler(repo)

	notes := "проверка"
	got, err := h.Handle(context.Background(), UpdateParticipantCommand{ParticipantID: 1, Notes: &notes})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.PrevElapsedTimeSec == nil || *got.PrevElapsedTimeSec != existing {
		t.Fatalf("prev elapsed changed unexpectedly: %v", got.PrevElapsedTimeSec)
	}
}

func TestUpdateParticipantLeavesStatusWhenNotProvided(t *testing.T) {
	repo := &updateParticipantRepoFake{stored: &entity.Participant{ID: 1, Status: valueobject.ParticipantStatusDisqualified}}
	h := NewUpdateParticipantHandler(repo)

	notes := "проверка"
	got, err := h.Handle(context.Background(), UpdateParticipantCommand{ParticipantID: 1, Notes: &notes})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Status != valueobject.ParticipantStatusDisqualified {
		t.Fatalf("status changed unexpectedly: %q", got.Status)
	}
}
