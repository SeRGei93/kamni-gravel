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

func TestRegistrationHandlerStartRegistrationEndedEvent(t *testing.T) {
	manager := session.NewManager(time.Minute)
	end := time.Now().Add(-time.Hour)
	eventRepo := &registrationEventRepoFake{
		event: &entity.Event{ID: 77, Name: "Тестовый заезд", Active: true, EndDate: &end},
	}
	h := NewRegistrationHandler(manager, eventRepo, nil, nil)

	text, markup := h.StartRegistration(context.Background(), 123)

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if !strings.Contains(text, "завершено") {
		t.Fatalf("text should mention ended event, got %q", text)
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
	}
}

func TestRegistrationHandlerConfirmRegistrationEndedEvent(t *testing.T) {
	manager := session.NewManager(time.Minute)
	end := time.Now().Add(-time.Hour)
	eventRepo := &registrationEventRepoFake{
		event: &entity.Event{ID: 77, Active: true, EndDate: &end},
	}
	h := NewRegistrationHandler(manager, eventRepo, nil, nil)

	manager.SetData(123, "event_id", uint(77))
	manager.SetData(123, "bike_type", "gravel")
	manager.SetData(123, "gender", "male")
	manager.SetState(123, session.StateAwaitingRegistrationConsent)

	text, err := h.ConfirmRegistration(context.Background(), 123)

	if err != nil {
		t.Fatalf("ConfirmRegistration error: %v", err)
	}
	if !strings.Contains(text, "завершено") {
		t.Fatalf("text should mention ended event, got %q", text)
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
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
