package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestWithdrawParticipantHandlerSuccess(t *testing.T) {
	repo := &withdrawParticipantRepoFake{participant: &entity.Participant{ID: 91, UserID: 123, EventID: 77}}
	handler := NewWithdrawParticipantHandler(repo)

	participant, err := handler.Handle(context.Background(), WithdrawParticipantCommand{
		UserID:  123,
		EventID: 77,
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if participant == nil {
		t.Fatal("participant is nil")
	}
	if got := participant.ID; got != 91 {
		t.Fatalf("participant id mismatch: got %d, want %d", got, 91)
	}
	if !repo.deleteCalled {
		t.Fatal("delete with result criteria was not called")
	}
}

func TestWithdrawParticipantHandlerNotFound(t *testing.T) {
	repo := &withdrawParticipantRepoFake{}
	handler := NewWithdrawParticipantHandler(repo)

	_, err := handler.Handle(context.Background(), WithdrawParticipantCommand{
		UserID:  123,
		EventID: 77,
	})

	if !errors.Is(err, ErrParticipantNotFound) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrParticipantNotFound)
	}
}

type withdrawParticipantRepoFake struct {
	participant  *entity.Participant
	deleteCalled bool
	errOnFind    error
	errOnDelete  error
}

func (r *withdrawParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *withdrawParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *withdrawParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	if r.participant == nil {
		if r.errOnFind != nil {
			return nil, r.errOnFind
		}
		return nil, repository.ErrParticipantNotFound
	}
	if r.participant.ID == id {
		return r.participant, nil
	}
	return nil, repository.ErrParticipantNotFound
}

func (r *withdrawParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	if r.participant == nil {
		if r.errOnFind != nil {
			return nil, r.errOnFind
		}
		return nil, repository.ErrParticipantNotFound
	}
	if r.participant.UserID == userID && r.participant.EventID == eventID {
		return r.participant, nil
	}
	return nil, repository.ErrParticipantNotFound
}

func (r *withdrawParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
func (r *withdrawParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}

func (r *withdrawParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }

func (r *withdrawParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	r.deleteCalled = true
	if r.errOnDelete != nil {
		return r.errOnDelete
	}
	return nil
}

func (r *withdrawParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
