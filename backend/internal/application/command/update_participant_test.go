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
