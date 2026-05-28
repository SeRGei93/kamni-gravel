package handler

import (
	"context"
	"strings"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestRegistrationHandlerHandleBikeTypeSelectionSetsSession(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewRegistrationHandler(manager, nil, nil, nil)

	text, markup := h.HandleBikeTypeSelection(context.Background(), 123, "gravel")

	if text != "Выберите пол:" {
		t.Fatalf("text mismatch: got %q", text)
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if got := manager.GetState(123); got != session.StateAwaitingGender {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGender)
	}

	bikeTypeRaw, ok := manager.GetData(123, "bike_type")
	if !ok {
		t.Fatal("bike_type missing from session")
	}
	if bikeTypeRaw != "gravel" {
		t.Fatalf("bike_type mismatch: got %v, want gravel", bikeTypeRaw)
	}
}

func TestRegistrationHandlerHandleGenderSelectionShowsConditions(t *testing.T) {
	manager := session.NewManager(time.Minute)
	eventRepo := &registrationEventRepoFake{
		event: &entity.Event{ID: 77, ParticipationConditions: "Условия участия"},
	}
	h := NewRegistrationHandler(manager, eventRepo, nil, nil)

	manager.SetData(123, "event_id", uint(77))
	text, markup := h.HandleGenderSelection(context.Background(), 123, "male")

	if !strings.Contains(text, "Условия участия") {
		t.Fatalf("text should contain conditions, got %q", text)
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if got := manager.GetState(123); got != session.StateAwaitingRegistrationConsent {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingRegistrationConsent)
	}

	genderRaw, ok := manager.GetData(123, "gender")
	if !ok {
		t.Fatal("gender missing from session")
	}
	if genderRaw != "male" {
		t.Fatalf("gender mismatch: got %v, want male", genderRaw)
	}
}

type registrationEventRepoFake struct {
	event *entity.Event
}

func (r *registrationEventRepoFake) Create(ctx context.Context, event *entity.Event) error {
	return nil
}
func (r *registrationEventRepoFake) Update(ctx context.Context, event *entity.Event) error {
	return nil
}
func (r *registrationEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *registrationEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *registrationEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *registrationEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *registrationEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }
