package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestSetManualGiftRecipientHandlerAssignsSelfForEveryParticipantStatus(t *testing.T) {
	statuses := []valueobject.ParticipantStatus{
		valueobject.ParticipantStatusActive,
		valueobject.ParticipantStatusDNF,
		valueobject.ParticipantStatusDisqualified,
	}
	for _, status := range statuses {
		t.Run(status.String(), func(t *testing.T) {
			recipientID := uint(20)
			giftRepo := &manualRecipientGiftRepoFake{gift: manualRecipientGift(nil)}
			participantRepo := &manualRecipientParticipantRepoFake{participant: &entity.Participant{
				ID:      recipientID,
				UserID:  100,
				EventID: 77,
				Status:  status,
			}}
			handler := NewSetManualGiftRecipientHandler(giftRepo, participantRepo)

			err := handler.Handle(context.Background(), SetManualGiftRecipientCommand{
				GiftID:                 1,
				Actor:                  ManualGiftRecipientActor{TelegramUserID: 100},
				RecipientParticipantID: &recipientID,
			})
			if err != nil {
				t.Fatalf("Handle error: %v", err)
			}
			if giftRepo.setCalls != 1 || giftRepo.recipientID == nil || *giftRepo.recipientID != recipientID {
				t.Fatalf("recipient update = calls:%d id:%v, want one self assignment", giftRepo.setCalls, giftRepo.recipientID)
			}
		})
	}
}

func TestSetManualGiftRecipientHandlerIsIdempotentAndAllowsClear(t *testing.T) {
	recipientID := uint(20)
	giftRepo := &manualRecipientGiftRepoFake{gift: manualRecipientGift(&recipientID)}
	participantRepo := &manualRecipientParticipantRepoFake{}
	handler := NewSetManualGiftRecipientHandler(giftRepo, participantRepo)

	command := SetManualGiftRecipientCommand{
		GiftID:                 1,
		Actor:                  ManualGiftRecipientActor{TelegramUserID: 100},
		RecipientParticipantID: &recipientID,
	}
	if err := handler.Handle(context.Background(), command); err != nil {
		t.Fatalf("idempotent assignment error: %v", err)
	}
	if giftRepo.setCalls != 0 {
		t.Fatalf("idempotent assignment should not write, calls=%d", giftRepo.setCalls)
	}

	command.RecipientParticipantID = nil
	if err := handler.Handle(context.Background(), command); err != nil {
		t.Fatalf("clear assignment error: %v", err)
	}
	if giftRepo.setCalls != 1 || giftRepo.recipientID != nil {
		t.Fatalf("clear recipient update = calls:%d id:%v", giftRepo.setCalls, giftRepo.recipientID)
	}
}

func TestSetManualGiftRecipientHandlerRejectsUnauthorizedAndInvalidRecipients(t *testing.T) {
	recipientID := uint(20)
	tests := []struct {
		name           string
		gift           *entity.Gift
		participant    *entity.Participant
		participantErr error
		actorID        int64
		want           error
	}{
		{
			name:    "foreign owner",
			gift:    manualRecipientGift(nil),
			actorID: 101,
			want:    ErrManualGiftOwnerForbidden,
		},
		{
			name:    "automatic gift",
			gift:    &entity.Gift{ID: 1, UserID: 100, EventID: 77},
			actorID: 100,
			want:    ErrManualGiftNotManual,
		},
		{
			name:           "missing participant",
			gift:           manualRecipientGift(nil),
			participantErr: repository.ErrParticipantNotFound,
			actorID:        100,
			want:           ErrManualGiftRecipientNotFound,
		},
		{
			name:        "cross-event participant",
			gift:        manualRecipientGift(nil),
			participant: &entity.Participant{ID: recipientID, EventID: 88},
			actorID:     100,
			want:        ErrManualGiftRecipientEvent,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			giftRepo := &manualRecipientGiftRepoFake{gift: testCase.gift}
			participantRepo := &manualRecipientParticipantRepoFake{participant: testCase.participant, err: testCase.participantErr}
			handler := NewSetManualGiftRecipientHandler(giftRepo, participantRepo)

			err := handler.Handle(context.Background(), SetManualGiftRecipientCommand{
				GiftID:                 1,
				Actor:                  ManualGiftRecipientActor{TelegramUserID: testCase.actorID},
				RecipientParticipantID: &recipientID,
			})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("Handle error = %v, want %v", err, testCase.want)
			}
			if giftRepo.setCalls != 0 {
				t.Fatalf("rejected command must not write, calls=%d", giftRepo.setCalls)
			}
		})
	}
}

func TestSetManualGiftRecipientHandlerMapsWriteTimeRaceErrors(t *testing.T) {
	recipientID := uint(20)
	giftRepo := &manualRecipientGiftRepoFake{
		gift:   manualRecipientGift(nil),
		setErr: repository.ErrManualRecipientEventMismatch,
	}
	handler := NewSetManualGiftRecipientHandler(giftRepo, &manualRecipientParticipantRepoFake{participant: &entity.Participant{ID: recipientID, EventID: 77}})

	err := handler.Handle(context.Background(), SetManualGiftRecipientCommand{
		GiftID:                 1,
		Actor:                  ManualGiftRecipientActor{TelegramUserID: 100},
		RecipientParticipantID: &recipientID,
	})
	if !errors.Is(err, ErrManualGiftRecipientEvent) {
		t.Fatalf("race error = %v, want ErrManualGiftRecipientEvent", err)
	}
}

func manualRecipientGift(recipientID *uint) *entity.Gift {
	return &entity.Gift{
		ID:                           1,
		UserID:                       100,
		EventID:                      77,
		ManualDistribution:           true,
		ManualRecipientParticipantID: recipientID,
	}
}

type manualRecipientGiftRepoFake struct {
	repository.ManualGiftRepository
	gift        *entity.Gift
	setErr      error
	setCalls    int
	recipientID *uint
}

func (r *manualRecipientGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	if r.gift == nil {
		return nil, repository.ErrGiftNotFound
	}
	copy := *r.gift
	return &copy, nil
}

func (r *manualRecipientGiftRepoFake) SetManualRecipient(ctx context.Context, giftID uint, recipientID *uint) error {
	r.setCalls++
	r.recipientID = recipientID
	return r.setErr
}

type manualRecipientParticipantRepoFake struct {
	repository.ParticipantRepository
	participant *entity.Participant
	err         error
}

func (r *manualRecipientParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.participant, nil
}
