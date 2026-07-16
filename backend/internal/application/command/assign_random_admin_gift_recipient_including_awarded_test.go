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

func TestAssignRandomAdminGiftRecipientIncludingAwardedHandlerAssignsAlreadyAwardedEligibleParticipant(t *testing.T) {
	writer := &randomAdminManualGiftRecipientIncludingAwardedWriterFake{}
	handler := newAssignRandomAdminGiftRecipientIncludingAwardedHandler(
		&randomAdminManualGiftRecipientIncludingAwardedGiftRepoFake{gift: randomAdminManualGiftIncludingAwardedGift(nil)},
		writer,
		&randomAdminManualGiftRecipientIncludingAwardedParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Result: &entity.Result{}}},
		&randomAdminManualGiftRecipientIncludingAwardedCandidateReaderFake{participantIDs: []uint{10}},
		func(int) (int, error) { return 0, nil },
	)

	result, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientIncludingAwardedCommand{GiftID: 1, AdminID: 5})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if result == nil || result.GiftID != 1 || result.EventID != 77 || result.RecipientParticipantID != 10 {
		t.Fatalf("result = %+v, want manual assignment", result)
	}
	if writer.calls != 1 || writer.giftID != 1 || writer.recipientID != 10 {
		t.Fatalf("writer call = %+v, want gift=1 recipient=10 once", writer)
	}
}

func TestAssignRandomAdminGiftRecipientIncludingAwardedHandlerRejectsUnavailableGiftStates(t *testing.T) {
	recipientID := uint(10)
	tests := []struct {
		name    string
		gift    *entity.Gift
		wantErr error
	}{
		{
			name:    "pending review",
			gift:    &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview, ManualDistribution: true},
			wantErr: ErrAdminRandomGiftNotApproved,
		},
		{
			name:    "automatic gift",
			gift:    &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved},
			wantErr: ErrManualGiftNotManual,
		},
		{
			name:    "recipient already assigned",
			gift:    randomAdminManualGiftIncludingAwardedGift(&recipientID),
			wantErr: ErrAdminRandomGiftAlreadyAssigned,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			writer := &randomAdminManualGiftRecipientIncludingAwardedWriterFake{}
			handler := newAssignRandomAdminGiftRecipientIncludingAwardedHandler(
				&randomAdminManualGiftRecipientIncludingAwardedGiftRepoFake{gift: testCase.gift},
				writer,
				&randomAdminManualGiftRecipientIncludingAwardedParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Result: &entity.Result{}}},
				&randomAdminManualGiftRecipientIncludingAwardedCandidateReaderFake{participantIDs: []uint{10}},
				func(int) (int, error) { return 0, nil },
			)

			_, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientIncludingAwardedCommand{GiftID: 1, AdminID: 5})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Handle error = %v, want %v", err, testCase.wantErr)
			}
			if writer.calls != 0 {
				t.Fatalf("writer calls = %d, want 0", writer.calls)
			}
		})
	}
}

func TestAssignRandomAdminGiftRecipientIncludingAwardedHandlerRejectsInvalidSelectedParticipant(t *testing.T) {
	tests := []struct {
		name        string
		participant *entity.Participant
		wantErr     error
	}{
		{
			name:        "cross event participant",
			participant: &entity.Participant{ID: 10, EventID: 88, Result: &entity.Result{}},
			wantErr:     ErrManualGiftRecipientEvent,
		},
		{
			name:        "dns participant",
			participant: &entity.Participant{ID: 10, EventID: 77},
			wantErr:     ErrManualGiftRecipientIneligible,
		},
		{
			name:        "disqualified participant",
			participant: &entity.Participant{ID: 10, EventID: 77, Status: valueobject.ParticipantStatusDisqualified, Result: &entity.Result{}},
			wantErr:     ErrManualGiftRecipientIneligible,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			writer := &randomAdminManualGiftRecipientIncludingAwardedWriterFake{}
			handler := newAssignRandomAdminGiftRecipientIncludingAwardedHandler(
				&randomAdminManualGiftRecipientIncludingAwardedGiftRepoFake{gift: randomAdminManualGiftIncludingAwardedGift(nil)},
				writer,
				&randomAdminManualGiftRecipientIncludingAwardedParticipantRepoFake{participant: testCase.participant},
				&randomAdminManualGiftRecipientIncludingAwardedCandidateReaderFake{participantIDs: []uint{10}},
				func(int) (int, error) { return 0, nil },
			)

			_, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientIncludingAwardedCommand{GiftID: 1, AdminID: 5})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Handle error = %v, want %v", err, testCase.wantErr)
			}
			if writer.calls != 0 {
				t.Fatalf("writer calls = %d, want 0", writer.calls)
			}
		})
	}
}

func TestAssignRandomAdminGiftRecipientIncludingAwardedHandlerRejectsNoCandidatesRandomFailureAndRace(t *testing.T) {
	tests := []struct {
		name           string
		participantIDs []uint
		randomIndex    func(int) (int, error)
		writerErr      error
		wantErr        error
	}{
		{
			name:           "no eligible participants",
			participantIDs: nil,
			randomIndex:    func(int) (int, error) { return 0, nil },
			wantErr:        ErrManualGiftNoEligibleParticipants,
		},
		{
			name:           "secure random failure",
			participantIDs: []uint{10},
			randomIndex: func(int) (int, error) {
				return 0, errors.New("entropy source unavailable")
			},
			wantErr: errors.New("choose random manual gift recipient including awarded participants"),
		},
		{
			name:           "compare and set race",
			participantIDs: []uint{10},
			randomIndex:    func(int) (int, error) { return 0, nil },
			writerErr:      repository.ErrRandomGiftRecipientAlreadyAssigned,
			wantErr:        ErrAdminRandomGiftAlreadyAssigned,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			writer := &randomAdminManualGiftRecipientIncludingAwardedWriterFake{err: testCase.writerErr}
			handler := newAssignRandomAdminGiftRecipientIncludingAwardedHandler(
				&randomAdminManualGiftRecipientIncludingAwardedGiftRepoFake{gift: randomAdminManualGiftIncludingAwardedGift(nil)},
				writer,
				&randomAdminManualGiftRecipientIncludingAwardedParticipantRepoFake{participant: &entity.Participant{ID: 10, EventID: 77, Result: &entity.Result{}}},
				&randomAdminManualGiftRecipientIncludingAwardedCandidateReaderFake{participantIDs: testCase.participantIDs},
				testCase.randomIndex,
			)

			_, err := handler.Handle(context.Background(), AssignRandomAdminGiftRecipientIncludingAwardedCommand{GiftID: 1, AdminID: 5})
			if testCase.name == "secure random failure" {
				if err == nil || !containsErrorText(err, testCase.wantErr.Error()) {
					t.Fatalf("Handle error = %v, want random-selection error", err)
				}
				return
			}
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Handle error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func randomAdminManualGiftIncludingAwardedGift(recipientID *uint) *entity.Gift {
	return &entity.Gift{
		ID:                           1,
		EventID:                      77,
		ReviewStatus:                 entity.GiftReviewStatusApproved,
		ManualDistribution:           true,
		ManualRecipientParticipantID: recipientID,
	}
}

type randomAdminManualGiftRecipientIncludingAwardedGiftRepoFake struct {
	repository.GiftRepository
	gift *entity.Gift
	err  error
}

func (r *randomAdminManualGiftRecipientIncludingAwardedGiftRepoFake) FindByID(context.Context, uint) (*entity.Gift, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.gift == nil {
		return nil, repository.ErrGiftNotFound
	}
	copy := *r.gift
	return &copy, nil
}

type randomAdminManualGiftRecipientIncludingAwardedWriterFake struct {
	calls       int
	giftID      uint
	recipientID uint
	err         error
}

func (r *randomAdminManualGiftRecipientIncludingAwardedWriterFake) AssignRandomManualRecipientIncludingAwarded(_ context.Context, giftID, recipientID uint) error {
	r.calls++
	r.giftID = giftID
	r.recipientID = recipientID
	return r.err
}

type randomAdminManualGiftRecipientIncludingAwardedParticipantRepoFake struct {
	repository.ParticipantRepository
	participant *entity.Participant
	err         error
}

func (r *randomAdminManualGiftRecipientIncludingAwardedParticipantRepoFake) FindByID(context.Context, uint) (*entity.Participant, error) {
	return r.participant, r.err
}

type randomAdminManualGiftRecipientIncludingAwardedCandidateReaderFake struct {
	participantIDs []uint
	err            error
}

func (r *randomAdminManualGiftRecipientIncludingAwardedCandidateReaderFake) Handle(context.Context, query.GetEligibleManualGiftParticipantIDsQuery) ([]uint, error) {
	return r.participantIDs, r.err
}
