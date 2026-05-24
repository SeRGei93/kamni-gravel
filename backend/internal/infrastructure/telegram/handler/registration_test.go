package handler

import (
	"context"
	"testing"
	"time"

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
