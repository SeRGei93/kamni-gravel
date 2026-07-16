package query

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestGetEligibleManualGiftParticipantIDsHandlerReturnsEligibleParticipantsRegardlessOfExistingPrizes(t *testing.T) {
	handler := NewGetEligibleManualGiftParticipantIDsHandler(
		&eligibleManualGiftParticipantRepoFake{participants: []*entity.Participant{
			// These two candidates can already have automatic or manually assigned
			// prizes. This query intentionally has no prize-distribution dependency.
			{ID: 1, EventID: 77, Status: valueobject.ParticipantStatusActive, Result: &entity.Result{}},
			{ID: 2, EventID: 77, Status: valueobject.ParticipantStatusDNF},
			{ID: 3, EventID: 77, Status: valueobject.ParticipantStatusActive},
			{ID: 4, EventID: 77, Status: valueobject.ParticipantStatusDisqualified, Result: &entity.Result{}},
			nil,
		}},
	)

	participantIDs, err := handler.Handle(context.Background(), GetEligibleManualGiftParticipantIDsQuery{EventID: 77})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if want := []uint{1, 2}; !reflect.DeepEqual(participantIDs, want) {
		t.Fatalf("participant IDs = %v, want %v", participantIDs, want)
	}
}

func TestGetEligibleManualGiftParticipantIDsHandlerReturnsRepositoryError(t *testing.T) {
	repositoryError := errors.New("database unavailable")
	handler := NewGetEligibleManualGiftParticipantIDsHandler(
		&eligibleManualGiftParticipantRepoFake{err: repositoryError},
	)

	_, err := handler.Handle(context.Background(), GetEligibleManualGiftParticipantIDsQuery{EventID: 77})
	if !errors.Is(err, repositoryError) {
		t.Fatalf("Handle error = %v, want wrapped repository error", err)
	}
}

type eligibleManualGiftParticipantRepoFake struct {
	repository.ParticipantRepository
	participants []*entity.Participant
	err          error
}

func (r *eligibleManualGiftParticipantRepoFake) FindByEvent(context.Context, uint) ([]*entity.Participant, error) {
	return r.participants, r.err
}
