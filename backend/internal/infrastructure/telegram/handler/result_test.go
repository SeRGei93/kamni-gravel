package handler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestResultHandlerStartSubmitResultUsesEditablePromptAfterStart(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	start := now.Add(-time.Minute)
	texts := entity.DefaultEventTelegramTexts()
	texts.ResultPrompt = "custom result prompt"
	h := NewResultHandler(
		manager,
		&resultEventRepoFake{event: &entity.Event{ID: 77, Active: true, StartDate: &start, TelegramTexts: texts}},
		&resultParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
		nil,
		WithResultHandlerClock(func() time.Time { return now }),
	)

	text, markup := h.StartSubmitResult(context.Background(), 123)

	if text != "custom result prompt" {
		t.Fatalf("text mismatch: got %q", text)
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if got := manager.GetState(123); got != session.StateAwaitingResultLink {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingResultLink)
	}
	if participantID, _ := manager.GetData(123, "participant_id"); participantID != uint(11) {
		t.Fatalf("participant_id mismatch: got %#v", participantID)
	}
}

func TestResultHandlerStartSubmitResultBlocksBeforeStart(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 5, 27, 11, 0, 0, 0, valueobject.MinskLocation())
	start := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	texts := entity.DefaultEventTelegramTexts()
	texts.ResultNotStarted = "open at {start_time}"
	h := NewResultHandler(
		manager,
		&resultEventRepoFake{event: &entity.Event{ID: 77, Active: true, StartDate: &start, TelegramTexts: texts}},
		&resultParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
		nil,
		WithResultHandlerClock(func() time.Time { return now }),
	)

	text, markup := h.StartSubmitResult(context.Background(), 123)

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if text != "open at 27.05.2026 12:00" {
		t.Fatalf("text mismatch: got %q", text)
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
	}
	if _, ok := manager.GetData(123, "participant_id"); ok {
		t.Fatal("participant_id should not be stored before event start")
	}
}

func TestResultHandlerStartSubmitResultBlocksMissingStart(t *testing.T) {
	manager := session.NewManager(time.Minute)
	texts := entity.DefaultEventTelegramTexts()
	texts.ResultStartMissing = "start is missing"
	h := NewResultHandler(
		manager,
		&resultEventRepoFake{event: &entity.Event{ID: 77, Active: true, TelegramTexts: texts}},
		&resultParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}},
		nil,
	)

	text, markup := h.StartSubmitResult(context.Background(), 123)

	if text != "start is missing" {
		t.Fatalf("text mismatch: got %q", text)
	}
	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
	}
}

func TestResultHandlerStartSubmitResultMapsNoActiveEvent(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewResultHandler(
		manager,
		&resultEventRepoFake{findActiveErr: errors.New("no active event found")},
		&resultParticipantRepoFake{},
		nil,
	)

	text, markup := h.StartSubmitResult(context.Background(), 123)

	if !strings.Contains(text, "нет активных событий") {
		t.Fatalf("text mismatch: got %q", text)
	}
	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
}

type resultEventRepoFake struct {
	event         *entity.Event
	findActiveErr error
}

func (r *resultEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *resultEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *resultEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *resultEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *resultEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	if r.findActiveErr != nil {
		return nil, r.findActiveErr
	}
	return r.event, nil
}
func (r *resultEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *resultEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type resultParticipantRepoFake struct {
	participant *entity.Participant
}

func (r *resultParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *resultParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *resultParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	return r.participant, nil
}
func (r *resultParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return r.participant, nil
}
func (r *resultParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
func (r *resultParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *resultParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *resultParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *resultParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
