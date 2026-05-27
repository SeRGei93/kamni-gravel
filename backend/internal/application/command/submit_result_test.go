package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestSubmitResultHandlerBlocksWhenEventStartMissing(t *testing.T) {
	participant := &entity.Participant{ID: 11, EventID: 77}
	resultRepo := &submitResultRepoFake{}
	h := newSubmitResultTestHandler(participant, &entity.Event{ID: 77, Active: true}, resultRepo, testMinskNow())

	_, err := h.Handle(context.Background(), SubmitResultCommand{
		ParticipantID: participant.ID,
		ResultLink:    "https://www.strava.com/activities/14758223172",
	})

	if !errors.Is(err, ErrEventStartNotConfigured) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrEventStartNotConfigured)
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created before event start is configured")
	}
}

func TestSubmitResultHandlerBlocksBeforeEventStart(t *testing.T) {
	participant := &entity.Participant{ID: 11, EventID: 77}
	start := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	resultRepo := &submitResultRepoFake{}
	h := newSubmitResultTestHandler(
		participant,
		&entity.Event{ID: 77, Active: true, StartDate: &start},
		resultRepo,
		time.Date(2026, 5, 27, 11, 59, 0, 0, valueobject.MinskLocation()),
	)

	_, err := h.Handle(context.Background(), SubmitResultCommand{
		ParticipantID: participant.ID,
		ResultLink:    "https://www.strava.com/activities/14758223172",
	})

	if !errors.Is(err, ErrEventNotStarted) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrEventNotStarted)
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created before event start")
	}
}

func TestSubmitResultHandlerAcceptsAfterEventStart(t *testing.T) {
	now := testMinskNow()
	participant := &entity.Participant{ID: 11, EventID: 77}
	start := now.Add(-time.Minute)
	resultRepo := &submitResultRepoFake{}
	h := newSubmitResultTestHandler(
		participant,
		&entity.Event{ID: 77, Active: true, StartDate: &start},
		resultRepo,
		now,
	)

	updated, err := h.Handle(context.Background(), SubmitResultCommand{
		ParticipantID: participant.ID,
		ResultLink:    "https://www.strava.com/activities/14758223172?utm_source=telegram#comments",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if resultRepo.created == nil {
		t.Fatal("result was not created")
	}
	if updated.Result == nil || updated.Result != resultRepo.created {
		t.Fatalf("participant result mismatch: got %#v want %#v", updated.Result, resultRepo.created)
	}
	if resultRepo.created.SubmittedAt != now {
		t.Fatalf("submitted_at mismatch: got %s want %s", resultRepo.created.SubmittedAt, now)
	}
	if got := resultRepo.created.ResultLink.String(); got != "https://www.strava.com/activities/14758223172?utm_source=telegram#comments" {
		t.Fatalf("result link mismatch: got %q", got)
	}
}

func TestSubmitResultHandlerBlocksInactiveEvent(t *testing.T) {
	participant := &entity.Participant{ID: 11, EventID: 77}
	start := testMinskNow().Add(-time.Minute)
	resultRepo := &submitResultRepoFake{}
	h := newSubmitResultTestHandler(participant, &entity.Event{ID: 77, Active: false, StartDate: &start}, resultRepo, testMinskNow())

	_, err := h.Handle(context.Background(), SubmitResultCommand{
		ParticipantID: participant.ID,
		ResultLink:    "https://www.strava.com/activities/14758223172",
	})

	if !errors.Is(err, ErrResultSubmissionNotOpen) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrResultSubmissionNotOpen)
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created for inactive event")
	}
}

func newSubmitResultTestHandler(participant *entity.Participant, event *entity.Event, resultRepo *submitResultRepoFake, now time.Time) *SubmitResultHandler {
	return NewSubmitResultHandler(
		&submitParticipantRepoFake{participant: participant},
		&submitEventRepoFake{event: event},
		resultRepo,
		WithSubmitResultClock(func() time.Time { return now }),
	)
}

func testMinskNow() time.Time {
	return time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
}

type submitParticipantRepoFake struct {
	participant *entity.Participant
}

func (r *submitParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *submitParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *submitParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	if r.participant == nil {
		return nil, repository.ErrParticipantNotFound
	}
	return r.participant, nil
}
func (r *submitParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *submitParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
func (r *submitParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *submitParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *submitParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *submitParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}

type submitEventRepoFake struct {
	event *entity.Event
}

func (r *submitEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *submitEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *submitEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	if r.event == nil {
		return nil, errors.New("event not found")
	}
	return r.event, nil
}
func (r *submitEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *submitEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *submitEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *submitEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type submitResultRepoFake struct {
	created *entity.Result
}

func (r *submitResultRepoFake) Create(ctx context.Context, result *entity.Result) error {
	r.created = result
	return nil
}
func (r *submitResultRepoFake) FindByID(ctx context.Context, id uint) (*entity.Result, error) {
	return nil, nil
}
func (r *submitResultRepoFake) FindCurrentByParticipant(ctx context.Context, participantID uint) (*entity.Result, error) {
	return nil, nil
}
func (r *submitResultRepoFake) FindByParticipant(ctx context.Context, participantID uint) ([]*entity.Result, error) {
	return nil, nil
}
func (r *submitResultRepoFake) UpdateTime(ctx context.Context, id uint, elapsedSec, movingSec *int) error {
	return nil
}
func (r *submitResultRepoFake) MarkAsNotCurrent(ctx context.Context, id uint) error { return nil }
func (r *submitResultRepoFake) Delete(ctx context.Context, id uint) error           { return nil }
func (r *submitResultRepoFake) AddCriteria(ctx context.Context, resultID, criteriaID uint) error {
	return nil
}
func (r *submitResultRepoFake) RemoveCriteria(ctx context.Context, resultID, criteriaID uint) error {
	return nil
}
func (r *submitResultRepoFake) FindWithCriteria(ctx context.Context, resultID uint) (*entity.Result, error) {
	return nil, nil
}
func (r *submitResultRepoFake) FindByEventWithPlaces(ctx context.Context, eventID uint) ([]*repository.ResultWithPlace, error) {
	return nil, nil
}
