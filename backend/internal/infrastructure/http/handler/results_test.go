package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestResultsHandlerCreateUsesSubmitResultCommand(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	start := now.Add(-time.Minute)
	participantRepo := &resultsParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true, StartDate: &start}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, "https://www.strava.com/activities/14758223172")

	if rr.Code != http.StatusCreated {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if resultRepo.created == nil {
		t.Fatal("result was not created through command")
	}

	var got struct {
		ResultLink string `json:"result_link"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ResultLink != "https://www.strava.com/activities/14758223172" {
		t.Fatalf("result_link mismatch: got %q", got.ResultLink)
	}
}

func TestResultsHandlerCreateReturnsConflictBeforeEventStart(t *testing.T) {
	now := time.Date(2026, 5, 27, 11, 0, 0, 0, valueobject.MinskLocation())
	start := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	participantRepo := &resultsParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true, StartDate: &start}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, "https://www.strava.com/activities/14758223172")

	if rr.Code != http.StatusConflict {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created before event start")
	}
}

func TestResultsHandlerCreateReturnsBadRequestForKomoot(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	start := now.Add(-time.Minute)
	participantRepo := &resultsParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true, StartDate: &start}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, "https://www.komoot.com/tour/2308024419")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created for Komoot link")
	}
}

func newResultsTestHandler(participantRepo *resultsParticipantRepoFake, eventRepo *resultsEventRepoFake, resultRepo *resultsResultRepoFake, now time.Time) *ResultsHandler {
	submitHandler := command.NewSubmitResultHandler(
		participantRepo,
		eventRepo,
		resultRepo,
		command.WithSubmitResultClock(func() time.Time { return now }),
	)
	return NewResultsHandler(resultRepo, participantRepo, &resultsCriteriaRepoFake{}, submitHandler)
}

func resultsCreateRequest(t *testing.T, h *ResultsHandler, participantID uint, resultLink string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(CreateResultRequest{ResultLink: resultLink})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	router := chi.NewRouter()
	router.Post("/api/participants/{participantId}/results", h.Create)
	req := httptest.NewRequest(http.MethodPost, "/api/participants/"+uintString(participantID)+"/results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func uintString(value uint) string {
	return strconv.FormatUint(uint64(value), 10)
}

type resultsParticipantRepoFake struct {
	participant *entity.Participant
}

func (r *resultsParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *resultsParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *resultsParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	if r.participant == nil {
		return nil, repository.ErrParticipantNotFound
	}
	return r.participant, nil
}
func (r *resultsParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *resultsParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
func (r *resultsParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *resultsParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *resultsParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *resultsParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}

type resultsEventRepoFake struct {
	event *entity.Event
}

func (r *resultsEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *resultsEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *resultsEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *resultsEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *resultsEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *resultsEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *resultsEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type resultsResultRepoFake struct {
	created *entity.Result
}

func (r *resultsResultRepoFake) Create(ctx context.Context, result *entity.Result) error {
	result.ID = 101
	r.created = result
	return nil
}
func (r *resultsResultRepoFake) FindByID(ctx context.Context, id uint) (*entity.Result, error) {
	return nil, nil
}
func (r *resultsResultRepoFake) FindCurrentByParticipant(ctx context.Context, participantID uint) (*entity.Result, error) {
	return nil, nil
}
func (r *resultsResultRepoFake) FindByParticipant(ctx context.Context, participantID uint) ([]*entity.Result, error) {
	return nil, nil
}
func (r *resultsResultRepoFake) UpdateTime(ctx context.Context, id uint, elapsedSec, movingSec *int) error {
	return nil
}
func (r *resultsResultRepoFake) MarkAsNotCurrent(ctx context.Context, id uint) error {
	return nil
}
func (r *resultsResultRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *resultsResultRepoFake) AddCriteria(ctx context.Context, resultID, criteriaID uint) error {
	return nil
}
func (r *resultsResultRepoFake) RemoveCriteria(ctx context.Context, resultID, criteriaID uint) error {
	return nil
}
func (r *resultsResultRepoFake) FindWithCriteria(ctx context.Context, resultID uint) (*entity.Result, error) {
	return nil, nil
}
func (r *resultsResultRepoFake) FindByEventWithPlaces(ctx context.Context, eventID uint) ([]*repository.ResultWithPlace, error) {
	return nil, nil
}

type resultsCriteriaRepoFake struct{}

func (r *resultsCriteriaRepoFake) Create(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *resultsCriteriaRepoFake) Update(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *resultsCriteriaRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *resultsCriteriaRepoFake) FindByID(ctx context.Context, id uint) (*entity.Criteria, error) {
	return nil, nil
}
func (r *resultsCriteriaRepoFake) FindAll(ctx context.Context) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *resultsCriteriaRepoFake) FindByType(ctx context.Context, criteriaType valueobject.CriteriaType) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *resultsCriteriaRepoFake) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *resultsCriteriaRepoFake) FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error) {
	return nil, nil
}
