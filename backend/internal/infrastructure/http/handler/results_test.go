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

func TestResultsHandlerCreateManualResult(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	participantRepo := &resultsParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, CreateResultRequest{
		ElapsedTimeSec: intPointer(3600),
		MovingTimeSec:  intPointer(3500),
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if resultRepo.created == nil {
		t.Fatal("result was not created through manual command")
	}
	if resultRepo.created.ResultLink != nil {
		t.Fatalf("result link should be nil, got %#v", resultRepo.created.ResultLink)
	}

	var got struct {
		ElapsedTimeSec int `json:"elapsed_time_sec"`
		MovingTimeSec  int `json:"moving_time_sec"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ElapsedTimeSec != 3600 || got.MovingTimeSec != 3500 {
		t.Fatalf("time mismatch: got elapsed=%d moving=%d", got.ElapsedTimeSec, got.MovingTimeSec)
	}
}

func TestResultsHandlerCreateReturnsBadRequestWithoutElapsedTime(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	participantRepo := &resultsParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, CreateResultRequest{
		ResultLink: "https://www.strava.com/activities/14758223172",
	})

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created without elapsed_time_sec")
	}
}

func TestResultsHandlerCreateReturnsConflictWhenCurrentResultExists(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	participantRepo := &resultsParticipantRepoFake{
		participant: &entity.Participant{
			ID:      11,
			EventID: 77,
			Result:  &entity.Result{ID: 88, IsCurrent: true},
		},
	}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, CreateResultRequest{
		ElapsedTimeSec: intPointer(3600),
	})

	if rr.Code != http.StatusConflict {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created when current result exists")
	}
}

func TestResultsHandlerCreateReturnsBadRequestForKomoot(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	participantRepo := &resultsParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, CreateResultRequest{
		ElapsedTimeSec: intPointer(3600),
		ResultLink:     "https://www.komoot.com/tour/2308024419",
	})

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if resultRepo.created != nil {
		t.Fatal("result should not be created for Komoot link")
	}
}

func TestResultsHandlerCreateAcceptsStartFinishWithoutElapsed(t *testing.T) {
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, valueobject.MinskLocation())
	participantRepo := &resultsParticipantRepoFake{participant: &entity.Participant{ID: 11, EventID: 77}}
	resultRepo := &resultsResultRepoFake{}
	eventRepo := &resultsEventRepoFake{event: &entity.Event{ID: 77, Active: true}}
	h := newResultsTestHandler(participantRepo, eventRepo, resultRepo, now)

	rr := resultsCreateRequest(t, h, 11, CreateResultRequest{
		StartedAt:      stringPointer("2025-06-15T09:00:00+03:00"),
		FinishedAt:     stringPointer("2025-06-15T15:15:00+03:00"),
		DistanceMeters: intPointer(202000),
		AvgHeartRate:   intPointer(140),
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if resultRepo.created == nil {
		t.Fatal("result was not created from start/finish")
	}
	if resultRepo.created.ElapsedTimeSec == nil || *resultRepo.created.ElapsedTimeSec != 6*3600+15*60 {
		t.Fatalf("elapsed not computed from start/finish: %v", resultRepo.created.ElapsedTimeSec)
	}

	var got struct {
		AvgSpeedKmh *float64 `json:"avg_speed_kmh"`
		RideDate    *string  `json:"ride_date"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.AvgSpeedKmh == nil {
		t.Fatal("response should include computed avg_speed_kmh")
	}
	if got.RideDate == nil || *got.RideDate != "2025-06-15" {
		t.Fatalf("ride_date mismatch: got %v", got.RideDate)
	}
}

func stringPointer(value string) *string { return &value }

func newResultsTestHandler(participantRepo *resultsParticipantRepoFake, eventRepo *resultsEventRepoFake, resultRepo *resultsResultRepoFake, now time.Time) *ResultsHandler {
	submitHandler := command.NewSubmitResultHandler(
		participantRepo,
		eventRepo,
		resultRepo,
		command.WithSubmitResultClock(func() time.Time { return now }),
	)
	manualHandler := command.NewCreateManualResultHandler(
		participantRepo,
		resultRepo,
		command.WithCreateManualResultClock(func() time.Time { return now }),
	)
	updateHandler := command.NewUpdateResultHandler(resultRepo)
	return NewResultsHandler(resultRepo, participantRepo, &resultsCriteriaRepoFake{}, submitHandler, manualHandler, updateHandler)
}

func resultsCreateRequest(t *testing.T, h *ResultsHandler, participantID uint, bodyData CreateResultRequest) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(bodyData)
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

func intPointer(value int) *int {
	return &value
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
	stored  *entity.Result
	updated *entity.Result
}

func (r *resultsResultRepoFake) Create(ctx context.Context, result *entity.Result) error {
	result.ID = 101
	r.created = result
	return nil
}
func (r *resultsResultRepoFake) FindByID(ctx context.Context, id uint) (*entity.Result, error) {
	return r.stored, nil
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
func (r *resultsResultRepoFake) UpdateMetrics(ctx context.Context, result *entity.Result) error {
	r.updated = result
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
func (r *resultsCriteriaRepoFake) ListPaged(ctx context.Context, criteriaType *valueobject.CriteriaType, limit, offset int) ([]*entity.Criteria, int, error) {
	return nil, 0, nil
}
func (r *resultsCriteriaRepoFake) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *resultsCriteriaRepoFake) FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error) {
	return nil, nil
}
