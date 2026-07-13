package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
)

func TestAssignRandomManualGiftRecipientHandlerSelectsOnlyUnawardedParticipant(t *testing.T) {
	eligibleRecipientID := uint(30)
	setRecipientHandler := NewSetManualGiftRecipientHandler(
		&manualRecipientGiftRepoFake{gift: manualRecipientGift(nil)},
		&manualRecipientParticipantRepoFake{participant: &entity.Participant{ID: eligibleRecipientID, EventID: 77, Result: &entity.Result{}}},
	)
	participantIDsReader := &randomManualGiftRecipientIDsReaderFake{participantIDs: []uint{eligibleRecipientID}}
	handler := newAssignRandomManualGiftRecipientHandler(participantIDsReader, setRecipientHandler, func(max int) (int, error) {
		if max != 1 {
			t.Fatalf("random candidate count = %d, want 1", max)
		}
		return 0, nil
	})

	recipientID, err := handler.Handle(context.Background(), AssignRandomManualGiftRecipientCommand{
		GiftID:  1,
		EventID: 77,
		Actor:   ManualGiftRecipientActor{TelegramUserID: 100},
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if recipientID != eligibleRecipientID {
		t.Fatalf("recipient ID = %d, want %d", recipientID, eligibleRecipientID)
	}
}

func TestAssignRandomManualGiftRecipientHandlerRejectsWhenEveryoneHasPrize(t *testing.T) {
	handler := newAssignRandomManualGiftRecipientHandler(
		&randomManualGiftRecipientIDsReaderFake{},
		NewSetManualGiftRecipientHandler(&manualRecipientGiftRepoFake{gift: manualRecipientGift(nil)}, &manualRecipientParticipantRepoFake{}),
		func(int) (int, error) { return 0, nil },
	)

	_, err := handler.Handle(context.Background(), AssignRandomManualGiftRecipientCommand{
		GiftID:  1,
		EventID: 77,
		Actor:   ManualGiftRecipientActor{TelegramUserID: 100},
	})
	if !errors.Is(err, ErrManualGiftNoUnawardedParticipants) {
		t.Fatalf("Handle error = %v, want %v", err, ErrManualGiftNoUnawardedParticipants)
	}
}

type randomManualGiftRecipientIDsReaderFake struct {
	participantIDs []uint
	err            error
}

func (r *randomManualGiftRecipientIDsReaderFake) Handle(context.Context, query.GetEligibleUnawardedParticipantIDsQuery) ([]uint, error) {
	return r.participantIDs, r.err
}
