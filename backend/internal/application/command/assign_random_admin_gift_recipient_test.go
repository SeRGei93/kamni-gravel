package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestAssignRandomAdminGiftRecipientHandlerAssignsAutomaticUnassignedGift(t *testing.T) {
	writer := &randomAdminGiftRecipientWriterFake{}
	handler := newAssignRandomAdminGiftRecipientHandler(
		&randomAdminGiftRepoFake{gift: randomAdminGift(false, nil)},
		writer,
		&randomAdminParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Result: &entity.Result{}}},
		&randomAdminCandidateIDsReaderFake{participantIDs: []uint{10}},
		&randomAdminPrizeDistributionReaderFake{},
		func(max int) (int, error) { return 0, nil },
	)

	result, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientCommand{GiftID: 1})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if result == nil || !result.BecameManual || result.RecipientParticipantID != 10 || result.EventID != 77 {
		t.Fatalf("result = %+v, want automatic-to-manual assignment", result)
	}
	if writer.calls != 1 || writer.giftID != 1 || writer.recipientID != 10 {
		t.Fatalf("writer call = %+v, want gift=1 recipient=10 once", writer)
	}
}

func TestAssignRandomAdminGiftRecipientHandlerAssignsManualUnassignedGiftWithoutDistributionLookup(t *testing.T) {
	writer := &randomAdminGiftRecipientWriterFake{}
	distributionReader := &randomAdminPrizeDistributionReaderFake{}
	handler := newAssignRandomAdminGiftRecipientHandler(
		&randomAdminGiftRepoFake{gift: randomAdminGift(true, nil)},
		writer,
		&randomAdminParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Status: valueobject.ParticipantStatusDNF}},
		&randomAdminCandidateIDsReaderFake{participantIDs: []uint{10}},
		distributionReader,
		func(max int) (int, error) { return 0, nil },
	)

	result, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientCommand{GiftID: 1})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if result == nil || result.BecameManual {
		t.Fatalf("result = %+v, want existing manual gift assignment", result)
	}
	if distributionReader.calls != 0 {
		t.Fatalf("distribution calls = %d, want 0 for manual gift", distributionReader.calls)
	}
}

func TestAssignRandomAdminGiftRecipientHandlerRejectsUnavailableGiftStates(t *testing.T) {
	recipientID := uint(10)
	cases := []struct {
		name         string
		gift         *entity.Gift
		distribution []*query.PrizeDistributionResult
		wantErr      error
	}{
		{
			name:    "pending review",
			gift:    &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview},
			wantErr: ErrAdminRandomGiftNotApproved,
		},
		{
			name:    "manual recipient already assigned",
			gift:    randomAdminGift(true, &recipientID),
			wantErr: ErrAdminRandomGiftAlreadyAssigned,
		},
		{
			name: "automatic recipient recorded in assignments",
			gift: randomAdminGift(false, nil),
			distribution: []*query.PrizeDistributionResult{{
				ParticipantID:          99,
				MatchedGiftAssignments: []*query.PrizeGiftAssignment{{Gift: &entity.Gift{ID: 1}}},
			}},
			wantErr: ErrAdminRandomGiftAlreadyAssigned,
		},
		{
			name: "automatic recipient recorded in legacy gifts",
			gift: randomAdminGift(false, nil),
			distribution: []*query.PrizeDistributionResult{{
				ParticipantID: 99,
				MatchedGifts:  []*entity.Gift{{ID: 1}},
			}},
			wantErr: ErrAdminRandomGiftAlreadyAssigned,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			writer := &randomAdminGiftRecipientWriterFake{}
			handler := newAssignRandomAdminGiftRecipientHandler(
				&randomAdminGiftRepoFake{gift: testCase.gift},
				writer,
				&randomAdminParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Result: &entity.Result{}}},
				&randomAdminCandidateIDsReaderFake{participantIDs: []uint{10}},
				&randomAdminPrizeDistributionReaderFake{results: testCase.distribution},
				func(int) (int, error) { return 0, nil },
			)

			_, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientCommand{GiftID: 1})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Handle error = %v, want %v", err, testCase.wantErr)
			}
			if writer.calls != 0 {
				t.Fatalf("writer calls = %d, want 0", writer.calls)
			}
		})
	}
}

func TestAssignRandomAdminGiftRecipientHandlerRejectsNoCandidatesAndRandomError(t *testing.T) {
	for _, testCase := range []struct {
		name           string
		participantIDs []uint
		randomIndex    func(int) (int, error)
		wantErr        error
	}{
		{
			name:           "no unawarded participants",
			participantIDs: nil,
			randomIndex:    func(int) (int, error) { return 0, nil },
			wantErr:        ErrManualGiftNoUnawardedParticipants,
		},
		{
			name:           "secure random failure",
			participantIDs: []uint{10},
			randomIndex: func(int) (int, error) {
				return 0, errors.New("entropy source unavailable")
			},
			wantErr: errors.New("choose random gift recipient"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			writer := &randomAdminGiftRecipientWriterFake{}
			handler := newAssignRandomAdminGiftRecipientHandler(
				&randomAdminGiftRepoFake{gift: randomAdminGift(true, nil)},
				writer,
				&randomAdminParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Result: &entity.Result{}}},
				&randomAdminCandidateIDsReaderFake{participantIDs: testCase.participantIDs},
				&randomAdminPrizeDistributionReaderFake{},
				testCase.randomIndex,
			)

			_, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientCommand{GiftID: 1})
			if testCase.name == "secure random failure" {
				if err == nil || !containsErrorText(err, "choose random gift recipient") {
					t.Fatalf("Handle error = %v, want random-selection error", err)
				}
			} else if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Handle error = %v, want %v", err, testCase.wantErr)
			}
			if writer.calls != 0 {
				t.Fatalf("writer calls = %d, want 0", writer.calls)
			}
		})
	}
}

func TestAssignRandomAdminGiftRecipientHandlerMapsCompareAndSetConflict(t *testing.T) {
	handler := newAssignRandomAdminGiftRecipientHandler(
		&randomAdminGiftRepoFake{gift: randomAdminGift(true, nil)},
		&randomAdminGiftRecipientWriterFake{err: repository.ErrRandomGiftRecipientAlreadyAssigned},
		&randomAdminParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Result: &entity.Result{}}},
		&randomAdminCandidateIDsReaderFake{participantIDs: []uint{10}},
		&randomAdminPrizeDistributionReaderFake{},
		func(int) (int, error) { return 0, nil },
	)

	_, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientCommand{GiftID: 1})
	if !errors.Is(err, ErrAdminRandomGiftAlreadyAssigned) {
		t.Fatalf("Handle error = %v, want compare-and-set conflict", err)
	}
}

func randomAdminGift(manualDistribution bool, recipientID *uint) *entity.Gift {
	return &entity.Gift{
		ID:                           1,
		EventID:                      77,
		ReviewStatus:                 entity.GiftReviewStatusApproved,
		ManualDistribution:           manualDistribution,
		ManualRecipientParticipantID: recipientID,
	}
}

type randomAdminGiftRepoFake struct {
	repository.GiftRepository
	gift *entity.Gift
	err  error
}

func (r *randomAdminGiftRepoFake) FindByID(context.Context, uint) (*entity.Gift, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.gift == nil {
		return nil, repository.ErrGiftNotFound
	}
	copy := *r.gift
	return &copy, nil
}

type randomAdminGiftRecipientWriterFake struct {
	calls       int
	giftID      uint
	recipientID uint
	err         error
}

func (r *randomAdminGiftRecipientWriterFake) AssignRandomManualRecipient(_ context.Context, giftID, recipientID uint) error {
	r.calls++
	r.giftID = giftID
	r.recipientID = recipientID
	return r.err
}

type randomAdminParticipantRepoFake struct {
	repository.ParticipantRepository
	participant *entity.Participant
	err         error
}

func (r *randomAdminParticipantRepoFake) FindByID(context.Context, uint) (*entity.Participant, error) {
	return r.participant, r.err
}

type randomAdminCandidateIDsReaderFake struct {
	participantIDs []uint
	err            error
}

func (r *randomAdminCandidateIDsReaderFake) Handle(context.Context, query.GetEligibleUnawardedParticipantIDsQuery) ([]uint, error) {
	return r.participantIDs, r.err
}

type randomAdminPrizeDistributionReaderFake struct {
	results []*query.PrizeDistributionResult
	err     error
	calls   int
}

func (r *randomAdminPrizeDistributionReaderFake) Handle(context.Context, query.GetPrizeDistributionQuery) ([]*query.PrizeDistributionResult, error) {
	r.calls++
	return r.results, r.err
}

func containsErrorText(err error, want string) bool {
	for err != nil {
		if err.Error() == want || len(err.Error()) >= len(want) && err.Error()[:len(want)] == want {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
