package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestAssignRandomManualGiftRecipientIncludingAwardedHandlerAssignsEligibleParticipant(t *testing.T) {
	recipientID := uint(30)
	setRecipientHandler := NewSetManualGiftRecipientHandler(
		&manualRecipientGiftRepoFake{gift: manualRecipientGift(nil)},
		&manualRecipientParticipantRepoFake{participant: &entity.Participant{ID: recipientID, EventID: 77, Result: &entity.Result{}}},
	)
	writer := &initialManualGiftRecipientWriterFake{}
	handler := newAssignRandomManualGiftRecipientIncludingAwardedHandler(
		&eligibleRandomManualGiftRecipientIDsReaderFake{participantIDs: []uint{recipientID}},
		setRecipientHandler,
		writer,
		func(max int) (int, error) {
			if max != 1 {
				t.Fatalf("random candidate count = %d, want 1", max)
			}
			return 0, nil
		},
	)

	actualRecipientID, err := handler.Handle(context.Background(), AssignRandomManualGiftRecipientIncludingAwardedCommand{
		GiftID:  1,
		EventID: 77,
		Actor:   ManualGiftRecipientActor{TelegramUserID: 100},
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if actualRecipientID != recipientID {
		t.Fatalf("recipient ID = %d, want %d", actualRecipientID, recipientID)
	}
	if writer.calls != 1 || writer.giftID != 1 || writer.recipientID != recipientID {
		t.Fatalf("initial assignment = calls:%d gift:%d recipient:%d, want one assignment of recipient %d", writer.calls, writer.giftID, writer.recipientID, recipientID)
	}
}

func TestAssignRandomManualGiftRecipientIncludingAwardedHandlerRejectsGiftInvariantViolations(t *testing.T) {
	assignedRecipientID := uint(30)
	tests := []struct {
		name    string
		gift    *entity.Gift
		actorID int64
		eventID uint
		wantErr error
	}{
		{
			name:    "foreign owner",
			gift:    manualRecipientGift(nil),
			actorID: 101,
			eventID: 77,
			wantErr: ErrManualGiftOwnerForbidden,
		},
		{
			name:    "other event",
			gift:    manualRecipientGift(nil),
			actorID: 100,
			eventID: 88,
			wantErr: ErrManualGiftOwnerForbidden,
		},
		{
			name:    "automatic gift",
			gift:    &entity.Gift{ID: 1, UserID: 100, EventID: 77},
			actorID: 100,
			eventID: 77,
			wantErr: ErrManualGiftNotManual,
		},
		{
			name:    "recipient already assigned",
			gift:    manualRecipientGift(&assignedRecipientID),
			actorID: 100,
			eventID: 77,
			wantErr: ErrManualGiftRecipientAlreadyAssigned,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			handler := newAssignRandomManualGiftRecipientIncludingAwardedHandler(
				&eligibleRandomManualGiftRecipientIDsReaderFake{participantIDs: []uint{30}},
				NewSetManualGiftRecipientHandler(
					&manualRecipientGiftRepoFake{gift: testCase.gift},
					&manualRecipientParticipantRepoFake{participant: &entity.Participant{ID: 30, EventID: 77, Result: &entity.Result{}}},
				),
				&initialManualGiftRecipientWriterFake{},
				func(int) (int, error) { return 0, nil },
			)

			_, err := handler.Handle(context.Background(), AssignRandomManualGiftRecipientIncludingAwardedCommand{
				GiftID:  1,
				EventID: testCase.eventID,
				Actor:   ManualGiftRecipientActor{TelegramUserID: testCase.actorID},
			})
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Handle error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestAssignRandomManualGiftRecipientIncludingAwardedHandlerRejectsNoCandidatesAndRandomFailure(t *testing.T) {
	tests := []struct {
		name           string
		participantIDs []uint
		randomIndex    func(int) (int, error)
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
			participantIDs: []uint{30},
			randomIndex: func(int) (int, error) {
				return 0, errors.New("entropy source unavailable")
			},
			wantErr: errors.New("choose random manual gift recipient including awarded participants"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			handler := newAssignRandomManualGiftRecipientIncludingAwardedHandler(
				&eligibleRandomManualGiftRecipientIDsReaderFake{participantIDs: testCase.participantIDs},
				NewSetManualGiftRecipientHandler(
					&manualRecipientGiftRepoFake{gift: manualRecipientGift(nil)},
					&manualRecipientParticipantRepoFake{participant: &entity.Participant{ID: 30, EventID: 77, Result: &entity.Result{}}},
				),
				&initialManualGiftRecipientWriterFake{},
				testCase.randomIndex,
			)

			_, err := handler.Handle(context.Background(), AssignRandomManualGiftRecipientIncludingAwardedCommand{
				GiftID:  1,
				EventID: 77,
				Actor:   ManualGiftRecipientActor{TelegramUserID: 100},
			})
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

func TestAssignRandomManualGiftRecipientIncludingAwardedHandlerWrapsCandidateReaderError(t *testing.T) {
	readerError := errors.New("participants unavailable")
	handler := newAssignRandomManualGiftRecipientIncludingAwardedHandler(
		&eligibleRandomManualGiftRecipientIDsReaderFake{err: readerError},
		NewSetManualGiftRecipientHandler(
			&manualRecipientGiftRepoFake{gift: manualRecipientGift(nil)},
			&manualRecipientParticipantRepoFake{},
		),
		&initialManualGiftRecipientWriterFake{},
		func(int) (int, error) { return 0, nil },
	)

	_, err := handler.Handle(context.Background(), AssignRandomManualGiftRecipientIncludingAwardedCommand{
		GiftID:  1,
		EventID: 77,
		Actor:   ManualGiftRecipientActor{TelegramUserID: 100},
	})
	if !errors.Is(err, readerError) {
		t.Fatalf("Handle error = %v, want wrapped candidate-reader error", err)
	}
}

func TestAssignRandomManualGiftRecipientIncludingAwardedHandlerMapsClaimRace(t *testing.T) {
	recipientID := uint(30)
	writer := &initialManualGiftRecipientWriterFake{err: repository.ErrRandomGiftRecipientAlreadyAssigned}
	handler := newAssignRandomManualGiftRecipientIncludingAwardedHandler(
		&eligibleRandomManualGiftRecipientIDsReaderFake{participantIDs: []uint{recipientID}},
		NewSetManualGiftRecipientHandler(
			&manualRecipientGiftRepoFake{gift: manualRecipientGift(nil)},
			&manualRecipientParticipantRepoFake{participant: &entity.Participant{ID: recipientID, EventID: 77, Result: &entity.Result{}}},
		),
		writer,
		func(int) (int, error) { return 0, nil },
	)

	_, err := handler.Handle(context.Background(), AssignRandomManualGiftRecipientIncludingAwardedCommand{
		GiftID:  1,
		EventID: 77,
		Actor:   ManualGiftRecipientActor{TelegramUserID: 100},
	})
	if !errors.Is(err, ErrManualGiftRecipientAlreadyAssigned) {
		t.Fatalf("Handle error = %v, want %v", err, ErrManualGiftRecipientAlreadyAssigned)
	}
	if writer.calls != 1 {
		t.Fatalf("claim calls = %d, want 1", writer.calls)
	}
}

type eligibleRandomManualGiftRecipientIDsReaderFake struct {
	participantIDs []uint
	err            error
}

func (r *eligibleRandomManualGiftRecipientIDsReaderFake) Handle(context.Context, query.GetEligibleManualGiftParticipantIDsQuery) ([]uint, error) {
	return r.participantIDs, r.err
}

type initialManualGiftRecipientWriterFake struct {
	err         error
	calls       int
	giftID      uint
	recipientID uint
}

func (w *initialManualGiftRecipientWriterFake) AssignInitialManualRecipient(_ context.Context, giftID uint, recipientID uint) error {
	w.calls++
	w.giftID = giftID
	w.recipientID = recipientID
	return w.err
}
