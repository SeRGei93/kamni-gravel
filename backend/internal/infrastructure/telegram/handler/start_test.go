package handler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/domain/entity"
)

func TestStartHandlerHandleMissingSender(t *testing.T) {
	h := NewStartHandler(nil, nil, nil, "")

	text, markup := h.Handle(context.Background(), &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 20},
	})

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v", markup)
	}
	if !strings.Contains(text, "Не удалось определить пользователя") {
		t.Fatalf("text mismatch: got %q", text)
	}
}

func TestStartHandlerHandleConditionsPayload(t *testing.T) {
	h := NewStartHandler(&startUserRepoFake{}, &startEventRepoFake{event: &entity.Event{
		ID:                      11,
		Name:                    "Gran Fondo Test",
		Description:             "Описание тестового старта",
		ParticipationConditions: "Правила тестового старта",
	}}, nil, "")

	text, markup := h.Handle(context.Background(), &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 20},
		From: &models.User{ID: 123, FirstName: "Alex"},
		Text: " /start conditions",
	})

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if !strings.Contains(text, "Правила тестового старта") {
		t.Fatalf("text mismatch: got %q", text)
	}
}

func TestStartHandlerHandleStartShowsDescriptionOnly(t *testing.T) {
	h := NewStartHandler(&startUserRepoFake{}, &startEventRepoFake{event: &entity.Event{
		ID:                      11,
		Name:                    "Gran Fondo Test",
		Description:             "Описание тестового старта",
		ParticipationConditions: "Условия тестового старта",
	}}, &startParticipantRepoFake{}, "")

	text, markup := h.Handle(context.Background(), &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 20},
		From: &models.User{ID: 123, FirstName: "Alex"},
		Text: "/start",
	})

	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if !strings.Contains(text, "Описание тестового старта") {
		t.Fatalf("start text should contain description, got %q", text)
	}
	if strings.Contains(text, "Условия тестового старта") {
		t.Fatalf("start text should not contain participation conditions: %q", text)
	}
}

func TestStartHandlerHandleUnknownPayloadKeepsMenu(t *testing.T) {
	h := NewStartHandler(&startUserRepoFake{}, &startEventRepoFake{event: &entity.Event{
		ID:   11,
		Name: "Gran Fondo Test",
	}}, &startParticipantRepoFake{}, "")

	text, markup := h.Handle(context.Background(), &models.Message{
		ID:   10,
		Chat: models.Chat{ID: 20},
		From: &models.User{ID: 123, FirstName: "Alex"},
		Text: "/start unknown-payload",
	})

	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if !strings.Contains(text, "Что ты хочешь сделать?") {
		t.Fatalf("text mismatch: got %q", text)
	}
}

func TestEventConditionsTextFallback(t *testing.T) {
	if text := EventConditionsText(&entity.Event{}); !strings.Contains(text, "не заданы") {
		t.Fatalf("fallback mismatch: got %q", text)
	}

	text := EventConditionsText(&entity.Event{ParticipationConditions: "  Условия  "})
	if got, want := strings.TrimSpace(text), "Условия"; got != want {
		t.Fatalf("conditions mismatch: got %q, want %q", got, want)
	}
}

type startUserRepoFake struct{}

func (r *startUserRepoFake) Create(ctx context.Context, user *entity.User) error { return nil }
func (r *startUserRepoFake) Update(ctx context.Context, user *entity.User) error { return nil }
func (r *startUserRepoFake) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	return nil, fmt.Errorf("not found: %d", id)
}
func (r *startUserRepoFake) Delete(ctx context.Context, id int64) error { return nil }
func (r *startUserRepoFake) GetAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

type startEventRepoFake struct {
	event *entity.Event
}

func (r *startEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *startEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *startEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return nil, nil
}
func (r *startEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *startEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *startEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *startEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type startParticipantRepoFake struct{}

func (r *startParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *startParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *startParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *startParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *startParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
func (r *startParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *startParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *startParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *startParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
