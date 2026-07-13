package query

import (
	"context"
	"reflect"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

func TestGetEligibleUnawardedParticipantIDsHandlerReturnsOnlyEligibleParticipantsWithoutPrizes(t *testing.T) {
	handler := NewGetEligibleUnawardedParticipantIDsHandler(
		&miniappParticipantsRepoFake{participants: []*entity.Participant{
			{ID: 1, EventID: 77, Status: valueobject.ParticipantStatusActive, Result: &entity.Result{}},
			{ID: 2, EventID: 77, Status: valueobject.ParticipantStatusDNF},
			{ID: 3, EventID: 77, Status: valueobject.ParticipantStatusActive, Result: &entity.Result{}},
			{ID: 4, EventID: 77, Status: valueobject.ParticipantStatusActive, Result: &entity.Result{}},
			{ID: 5, EventID: 77, Status: valueobject.ParticipantStatusDisqualified, Result: &entity.Result{}},
			{ID: 6, EventID: 77, Status: valueobject.ParticipantStatusActive},
			{ID: 7, EventID: 77, Status: valueobject.ParticipantStatusActive, Result: &entity.Result{}},
		}},
		&miniappManualRecipientCountRepoFake{counts: map[uint]int{3: 1}},
		&miniappPrizeDistributionReaderFake{results: []*PrizeDistributionResult{
			{ParticipantID: 4, MatchedGiftAssignments: []*PrizeGiftAssignment{{Gift: &entity.Gift{ID: 100}}}},
			{ParticipantID: 7, MatchedGifts: []*entity.Gift{{ID: 101}}},
		}},
	)

	participantIDs, err := handler.Handle(context.Background(), GetEligibleUnawardedParticipantIDsQuery{EventID: 77})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if want := []uint{1, 2}; !reflect.DeepEqual(participantIDs, want) {
		t.Fatalf("participant IDs = %v, want %v", participantIDs, want)
	}
}
