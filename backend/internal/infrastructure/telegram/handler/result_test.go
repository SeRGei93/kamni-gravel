package handler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/telegram/session"
)

const (
	stravaActivityLink = "https://www.strava.com/activities/18929617181"
	stravaAppLink      = "https://strava.app.link/luP9ipxj13b"
)

func newResultHandlerWithSubmit(manager *session.Manager, participant *entity.Participant, event *entity.Event, resultRepo *resultResultRepoFake, now time.Time) *ResultHandler {
	participantRepo := &resultParticipantRepoFake{participant: participant}
	eventRepo := &resultEventRepoFake{event: event}
	submitHandler := command.NewSubmitResultHandler(
		participantRepo,
		eventRepo,
		resultRepo,
		command.WithSubmitResultClock(func() time.Time { return now }),
	)
	return NewResultHandler(
		manager,
		eventRepo,
		participantRepo,
		submitHandler,
		WithResultHandlerClock(func() time.Time { return now }),
	)
}

func mustResultLink(t *testing.T, raw string) *valueobject.ResultLink {
	t.Helper()
	link, err := valueobject.NewResultLink(raw)
	if err != nil {
		t.Fatalf("NewResultLink(%q) failed: %v", raw, err)
	}
	return link
}

func TestResultHandlerStartSubmitResultEndedEventStaysOpen(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 7, 7, 0, 0, 1, 0, valueobject.MinskLocation())
	start := now.Add(-24 * time.Hour)
	end := now.Add(-time.Second)
	h := newResultHandlerWithSubmit(
		manager,
		&entity.Participant{ID: 11, EventID: 77},
		&entity.Event{ID: 77, Name: "Тестовый заезд", Active: true, StartDate: &start, EndDate: &end},
		&resultResultRepoFake{},
		now,
	)

	// Окончание события без галочки StopResults не закрывает приём результатов.
	_, markup := h.StartSubmitResult(context.Background(), 123)

	if markup == nil {
		t.Fatal("markup mismatch: got nil, want cancel keyboard")
	}
	if got := manager.GetState(123); got != session.StateAwaitingResultLink {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingResultLink)
	}
}

func TestResultHandlerStartSubmitResultStoppedResults(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, valueobject.MinskLocation())
	start := now.Add(-24 * time.Hour)
	h := newResultHandlerWithSubmit(
		manager,
		&entity.Participant{ID: 11, EventID: 77},
		&entity.Event{ID: 77, Active: true, StartDate: &start, StopResults: true},
		&resultResultRepoFake{},
		now,
	)

	text, markup := h.StartSubmitResult(context.Background(), 123)

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if !strings.Contains(text, "Приём результатов завершён") {
		t.Fatalf("text should mention closed result intake, got %q", text)
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
	}
}

func TestResultHandlerSubmitOrConfirmStoppedResults(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 7, 7, 0, 0, 1, 0, valueobject.MinskLocation())
	start := now.Add(-24 * time.Hour)
	resultRepo := &resultResultRepoFake{}
	h := newResultHandlerWithSubmit(
		manager,
		&entity.Participant{ID: 11, EventID: 77},
		&entity.Event{ID: 77, Active: true, StartDate: &start, StopResults: true},
		resultRepo,
		now,
	)

	text, markup, participant := h.SubmitOrConfirm(context.Background(), 123, stravaActivityLink)

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if participant != nil {
		t.Fatalf("participant mismatch: got %#v, want nil", participant)
	}
	if !strings.Contains(text, "Приём результатов завершён") {
		t.Fatalf("text should mention closed result intake, got %q", text)
	}
	if resultRepo.created != nil {
		t.Fatal("result must not be created when intake is stopped")
	}
}

func TestResultHandlerSubmitOrConfirmSavesWhenNoResult(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	start := now.Add(-time.Minute)
	texts := entity.DefaultEventTelegramTexts()
	texts.ResultSuccess = "saved {result_link}"
	resultRepo := &resultResultRepoFake{}
	h := newResultHandlerWithSubmit(
		manager,
		&entity.Participant{ID: 11, EventID: 77},
		&entity.Event{ID: 77, Active: true, StartDate: &start, TelegramTexts: texts},
		resultRepo,
		now,
	)

	text, markup, participant := h.SubmitOrConfirm(context.Background(), 123, stravaActivityLink)

	if markup != nil {
		t.Fatalf("markup mismatch: got %#v, want nil", markup)
	}
	if participant == nil {
		t.Fatal("participant mismatch: got nil, want submitted participant")
	}
	if text != "saved "+stravaActivityLink {
		t.Fatalf("text mismatch: got %q", text)
	}
	if resultRepo.created == nil {
		t.Fatal("result was not created")
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
	}
}

func TestResultHandlerSubmitOrConfirmRequestsConfirmationThenReplaces(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	start := now.Add(-time.Minute)
	texts := entity.DefaultEventTelegramTexts()
	texts.ResultSuccess = "saved {result_link}"
	resultRepo := &resultResultRepoFake{}
	participant := &entity.Participant{
		ID:      11,
		EventID: 77,
		Result:  &entity.Result{ResultLink: mustResultLink(t, stravaActivityLink)},
	}
	h := newResultHandlerWithSubmit(
		manager,
		participant,
		&entity.Event{ID: 77, Active: true, StartDate: &start, TelegramTexts: texts},
		resultRepo,
		now,
	)

	text, markup, submitted := h.SubmitOrConfirm(context.Background(), 123, stravaAppLink)

	if markup == nil {
		t.Fatal("markup mismatch: got nil, want confirmation keyboard")
	}
	if submitted != nil {
		t.Fatalf("participant mismatch: got %#v, want nil before confirmation", submitted)
	}
	if resultRepo.created != nil {
		t.Fatal("result must not be created before confirmation")
	}
	if !strings.Contains(text, stravaAppLink) {
		t.Fatalf("confirmation text should mention new link: got %q", text)
	}
	if got := manager.GetState(123); got != session.StateAwaitingResultReplaceConfirmation {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingResultReplaceConfirmation)
	}

	confirmText, confirmed := h.ConfirmReplacement(context.Background(), 123)

	if confirmed == nil {
		t.Fatal("confirmed participant mismatch: got nil")
	}
	if resultRepo.created == nil {
		t.Fatal("result was not created after confirmation")
	}
	if got := resultRepo.created.ResultLink.String(); got != stravaAppLink {
		t.Fatalf("stored link mismatch: got %q, want %q", got, stravaAppLink)
	}
	if confirmText != "saved "+stravaAppLink {
		t.Fatalf("confirm text mismatch: got %q", confirmText)
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch after confirmation: got %s, want %s", got, session.StateIdle)
	}
}

func TestResultHandlerCancelReplacementResetsState(t *testing.T) {
	manager := session.NewManager(time.Minute)
	manager.SetState(123, session.StateAwaitingResultReplaceConfirmation)
	manager.SetData(123, "pending_result_link", stravaAppLink)
	h := NewResultHandler(manager, &resultEventRepoFake{}, &resultParticipantRepoFake{}, nil)

	text := h.CancelReplacement(123)

	if text != resultReplaceCancelText {
		t.Fatalf("text mismatch: got %q, want %q", text, resultReplaceCancelText)
	}
	if got := manager.GetState(123); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
	}
	if _, ok := manager.GetData(123, "pending_result_link"); ok {
		t.Fatal("pending link should be cleared after cancel")
	}
}

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

func TestResultHandlerStartSubmitResultAllowsReplaceWhenFinished(t *testing.T) {
	manager := session.NewManager(time.Minute)
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	start := now.Add(-time.Minute)
	texts := entity.DefaultEventTelegramTexts()
	texts.ResultPrompt = "custom result prompt"
	participant := &entity.Participant{
		ID:      11,
		EventID: 77,
		Result:  &entity.Result{ResultLink: mustResultLink(t, stravaActivityLink)},
	}
	h := NewResultHandler(
		manager,
		&resultEventRepoFake{event: &entity.Event{ID: 77, Active: true, StartDate: &start, TelegramTexts: texts}},
		&resultParticipantRepoFake{participant: participant},
		nil,
		WithResultHandlerClock(func() time.Time { return now }),
	)

	text, markup := h.StartSubmitResult(context.Background(), 123)

	// Финишировавший участник больше не блокируется — ему предлагают прислать ссылку,
	// замену он подтвердит на следующем шаге.
	if text != "custom result prompt" {
		t.Fatalf("text mismatch: got %q, want prompt", text)
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil, want cancel keyboard")
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

type resultResultRepoFake struct {
	created *entity.Result
}

func (r *resultResultRepoFake) Create(ctx context.Context, result *entity.Result) error {
	r.created = result
	return nil
}
func (r *resultResultRepoFake) FindByID(ctx context.Context, id uint) (*entity.Result, error) {
	return nil, nil
}
func (r *resultResultRepoFake) FindCurrentByParticipant(ctx context.Context, participantID uint) (*entity.Result, error) {
	return nil, nil
}
func (r *resultResultRepoFake) FindByParticipant(ctx context.Context, participantID uint) ([]*entity.Result, error) {
	return nil, nil
}
func (r *resultResultRepoFake) UpdateTime(ctx context.Context, id uint, elapsedSec, movingSec *int) error {
	return nil
}
func (r *resultResultRepoFake) UpdateMetrics(ctx context.Context, result *entity.Result) error {
	return nil
}
func (r *resultResultRepoFake) MarkAsNotCurrent(ctx context.Context, id uint) error { return nil }
func (r *resultResultRepoFake) Delete(ctx context.Context, id uint) error           { return nil }
func (r *resultResultRepoFake) AddCriteria(ctx context.Context, resultID, criteriaID uint) error {
	return nil
}
func (r *resultResultRepoFake) RemoveCriteria(ctx context.Context, resultID, criteriaID uint) error {
	return nil
}
func (r *resultResultRepoFake) FindWithCriteria(ctx context.Context, resultID uint) (*entity.Result, error) {
	return nil, nil
}
func (r *resultResultRepoFake) FindByEventWithPlaces(ctx context.Context, eventID uint) ([]*repository.ResultWithPlace, error) {
	return nil, nil
}
func (r *resultResultRepoFake) FindPrevEventElapsedByUser(ctx context.Context, eventID uint) (map[int64]int, error) {
	return nil, nil
}
