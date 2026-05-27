package command

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestRegisterParticipantHandlerRejectsBlacklistedUserBeforeUserLookup(t *testing.T) {
	userRepo := &registerUserRepoFake{user: &entity.User{ID: 123}}
	participantRepo := &participantRepoFake{}
	h := NewRegisterParticipantHandler(
		userRepo,
		&registerEventRepoFake{event: &entity.Event{ID: 77, Active: true}},
		participantRepo,
		&userBlacklistRepoFake{blacklisted: true},
	)

	_, err := h.Handle(context.Background(), RegisterParticipantCommand{
		UserID:   123,
		EventID:  77,
		BikeType: "gravel",
		Gender:   "male",
	})
	if !errors.Is(err, ErrUserBlacklisted) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrUserBlacklisted)
	}
	if userRepo.findCalled {
		t.Fatal("user lookup should not happen after blacklist rejection")
	}
	if participantRepo.createCalled {
		t.Fatal("participant should not be created for blacklisted user")
	}
}

func TestDeleteParticipantHandlerUsesSafeRepositoryCleanup(t *testing.T) {
	participantRepo := &participantRepoFake{
		participant: &entity.Participant{ID: 55, EventID: 77},
	}
	h := NewDeleteParticipantHandler(participantRepo)

	if err := h.Handle(context.Background(), DeleteParticipantCommand{ParticipantID: 55}); err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if !participantRepo.deleteWithResultCriteriaCalled {
		t.Fatal("DeleteWithResultCriteria was not called")
	}
}

func TestDeleteParticipantHandlerMapsMissingParticipant(t *testing.T) {
	h := NewDeleteParticipantHandler(&participantRepoFake{})

	err := h.Handle(context.Background(), DeleteParticipantCommand{ParticipantID: 55})
	if !errors.Is(err, ErrParticipantNotFound) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrParticipantNotFound)
	}
}

func TestDeleteParticipantHandlerPropagatesFindError(t *testing.T) {
	h := NewDeleteParticipantHandler(&participantRepoFake{findErr: fmt.Errorf("database unavailable")})

	err := h.Handle(context.Background(), DeleteParticipantCommand{ParticipantID: 55})
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrParticipantNotFound) {
		t.Fatalf("error should not be mapped to not found: %v", err)
	}
}

func TestDeleteParticipantHandlerMapsConcurrentMissingParticipant(t *testing.T) {
	h := NewDeleteParticipantHandler(&participantRepoFake{
		participant: &entity.Participant{ID: 55, EventID: 77},
		deleteErr:   fmt.Errorf("%w: 55", repository.ErrParticipantNotFound),
	})

	err := h.Handle(context.Background(), DeleteParticipantCommand{ParticipantID: 55})
	if !errors.Is(err, ErrParticipantNotFound) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrParticipantNotFound)
	}
}

type registerUserRepoFake struct {
	user       *entity.User
	findCalled bool
}

func (r *registerUserRepoFake) Create(ctx context.Context, user *entity.User) error { return nil }
func (r *registerUserRepoFake) Update(ctx context.Context, user *entity.User) error { return nil }
func (r *registerUserRepoFake) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	r.findCalled = true
	if r.user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return r.user, nil
}
func (r *registerUserRepoFake) Delete(ctx context.Context, id int64) error { return nil }
func (r *registerUserRepoFake) GetAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

type registerEventRepoFake struct {
	event *entity.Event
}

func (r *registerEventRepoFake) Create(ctx context.Context, event *entity.Event) error {
	return nil
}
func (r *registerEventRepoFake) Update(ctx context.Context, event *entity.Event) error {
	return nil
}
func (r *registerEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	if r.event == nil {
		return nil, fmt.Errorf("event not found")
	}
	return r.event, nil
}
func (r *registerEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *registerEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *registerEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *registerEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type participantRepoFake struct {
	participant                    *entity.Participant
	findErr                        error
	deleteErr                      error
	createCalled                   bool
	deleteWithResultCriteriaCalled bool
}

func (r *participantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	r.createCalled = true
	participant.ID = 1
	return nil
}
func (r *participantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *participantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.participant == nil {
		return nil, repository.ErrParticipantNotFound
	}
	return r.participant, nil
}
func (r *participantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return nil, fmt.Errorf("participant not found")
}
func (r *participantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	bikeType, _ := valueobject.NewBikeType("gravel")
	gender, _ := valueobject.NewGender("male")
	return []*entity.Participant{{
		ID:           1,
		UserID:       123,
		EventID:      eventID,
		BikeType:     bikeType,
		Gender:       gender,
		RegisteredAt: time.Now(),
	}}, nil
}
func (r *participantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *participantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *participantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	r.deleteWithResultCriteriaCalled = true
	return r.deleteErr
}
func (r *participantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
