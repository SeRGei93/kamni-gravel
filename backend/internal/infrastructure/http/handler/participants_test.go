package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestParticipantsHandlerGetAllMarksParticipantsWithGift(t *testing.T) {
	participantRepo := &participantListParticipantRepoFake{
		participants: []*entity.Participant{
			{
				ID:       1,
				UserID:   111,
				EventID:  77,
				BikeType: valueobject.BikeTypeGravel,
				Gender:   valueobject.GenderMale,
				User:     &entity.User{ID: 111, Username: "without_gift"},
			},
			{
				ID:       2,
				UserID:   222,
				EventID:  77,
				BikeType: valueobject.BikeTypeMTB,
				Gender:   valueobject.GenderFemale,
				User:     &entity.User{ID: 222, Username: "with_gift"},
			},
		},
	}
	giftRepo := &participantListGiftRepoFake{
		gifts: []*entity.Gift{
			{ID: 10, UserID: 222, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview},
		},
	}
	h := &ParticipantsHandler{
		participantRepo:        participantRepo,
		resultRepo:             &participantListResultRepoFake{},
		giftRepo:               giftRepo,
		getParticipantsHandler: query.NewGetParticipantsHandler(participantRepo),
	}

	router := chi.NewRouter()
	router.Get("/api/events/{eventId}/participants", h.GetAll)
	req := httptest.NewRequest(http.MethodGet, "/api/events/77/participants", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if giftRepo.eventID != 77 {
		t.Fatalf("gift event id mismatch: got %d, want 77", giftRepo.eventID)
	}

	var got dto.ParticipantListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	hasGiftByParticipantID := make(map[uint]bool, len(got.Participants))
	for _, participant := range got.Participants {
		hasGiftByParticipantID[participant.ID] = participant.HasGift
	}

	if hasGiftByParticipantID[1] {
		t.Fatal("participant without gift was marked as having gift")
	}
	if !hasGiftByParticipantID[2] {
		t.Fatal("participant with gift was not marked as having gift")
	}
}

func TestParticipantsHandlerCreateReturnsForbiddenForBlacklistedUser(t *testing.T) {
	h := &ParticipantsHandler{
		registerParticipantHandler: command.NewRegisterParticipantHandler(
			nil,
			nil,
			nil,
			&participantCreateBlacklistRepoFake{blacklisted: true},
		),
	}

	req := httptest.NewRequest(
		"POST",
		"/api/events/77/participants",
		strings.NewReader(`{"user_id":123,"bike_type":"gravel","gender":"male"}`),
	)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("eventId", "77")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
}

type participantCreateBlacklistRepoFake struct {
	blacklisted bool
}

func (r *participantCreateBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *participantCreateBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *participantCreateBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	return r.blacklisted, nil
}
func (r *participantCreateBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	return nil
}
func (r *participantCreateBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *participantCreateBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return nil
}

type participantListParticipantRepoFake struct {
	repository.ParticipantRepository
	participants []*entity.Participant
}

func (r *participantListParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return r.participants, nil
}

func (r *participantListParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	for _, p := range r.participants {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, repository.ErrParticipantNotFound
}

type participantListResultRepoFake struct {
	repository.ResultRepository
	prevElapsedByUser map[int64]int
}

func (r *participantListResultRepoFake) FindByEventWithPlaces(ctx context.Context, eventID uint) ([]*repository.ResultWithPlace, error) {
	return nil, nil
}

func (r *participantListResultRepoFake) FindPrevEventElapsedByUser(ctx context.Context, eventID uint) (map[int64]int, error) {
	return r.prevElapsedByUser, nil
}

type participantListGiftRepoFake struct {
	repository.GiftRepository
	eventID uint
	gifts   []*entity.Gift
}

func (r *participantListGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	r.eventID = eventID
	return r.gifts, nil
}

func (r *participantListGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, status entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}

func newParticipantsListTestHandler() *ParticipantsHandler {
	aliceElapsed := 7200
	participantRepo := &participantListParticipantRepoFake{
		participants: []*entity.Participant{
			{ID: 1, UserID: 111, EventID: 77, BikeType: valueobject.BikeTypeGravel, Gender: valueobject.GenderMale, User: &entity.User{ID: 111, Username: "alice"},
				Result: &entity.Result{ID: 9, ParticipantID: 1, IsCurrent: true, ElapsedTimeSec: &aliceElapsed}},
			{ID: 2, UserID: 222, EventID: 77, BikeType: valueobject.BikeTypeMTB, Gender: valueobject.GenderFemale, User: &entity.User{ID: 222, Username: "bob_with_gift"}},
			{ID: 3, UserID: 333, EventID: 77, BikeType: valueobject.BikeTypeGravel, Gender: valueobject.GenderMale, User: &entity.User{ID: 333, Username: "carol"}},
		},
	}
	giftRepo := &participantListGiftRepoFake{
		gifts: []*entity.Gift{{ID: 10, UserID: 222, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview}},
	}
	return &ParticipantsHandler{
		participantRepo: participantRepo,
		// user 111 имеет время на предыдущем событии (1:59:59), остальные — нет.
		resultRepo:             &participantListResultRepoFake{prevElapsedByUser: map[int64]int{111: 7199}},
		giftRepo:               giftRepo,
		getParticipantsHandler: query.NewGetParticipantsHandler(participantRepo),
	}
}

func getParticipantsList(t *testing.T, h *ParticipantsHandler, url string) dto.ParticipantListResponse {
	t.Helper()
	router := chi.NewRouter()
	router.Get("/api/events/{eventId}/participants", h.GetAll)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.ParticipantListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return got
}

func TestParticipantsHandlerPaginationEcho(t *testing.T) {
	got := getParticipantsList(t, newParticipantsListTestHandler(), "/api/events/77/participants?page=1&page_size=50")
	if got.Total != 3 {
		t.Fatalf("total mismatch: got %d, want 3", got.Total)
	}
	if got.Page != 1 || got.PageSize != 50 {
		t.Fatalf("page echo mismatch: page=%d page_size=%d", got.Page, got.PageSize)
	}
	if len(got.Participants) != 3 {
		t.Fatalf("participants mismatch: got %d, want 3", len(got.Participants))
	}
}

func TestParticipantsHandlerWithoutPaginationReturnsAll(t *testing.T) {
	got := getParticipantsList(t, newParticipantsListTestHandler(), "/api/events/77/participants")
	if got.Total != 3 || len(got.Participants) != 3 {
		t.Fatalf("expected all 3 participants, got total=%d len=%d", got.Total, len(got.Participants))
	}
	if got.Page != 0 || got.PageSize != 0 {
		t.Fatalf("page fields should be omitted when not paginating: page=%d page_size=%d", got.Page, got.PageSize)
	}
}

func TestParticipantsHandlerAllPageSizeReturnsAll(t *testing.T) {
	got := getParticipantsList(t, newParticipantsListTestHandler(), "/api/events/77/participants?page=3&page_size=all")
	if got.Total != 3 || len(got.Participants) != 3 {
		t.Fatalf("expected all 3 participants, got total=%d len=%d", got.Total, len(got.Participants))
	}
	if got.Page != 0 || got.PageSize != 0 {
		t.Fatalf("page fields should be omitted for all page size: page=%d page_size=%d", got.Page, got.PageSize)
	}
}

func TestParticipantsHandlerHasGiftFilter(t *testing.T) {
	got := getParticipantsList(t, newParticipantsListTestHandler(), "/api/events/77/participants?page=1&page_size=50&has_gift=true")
	if got.Total != 1 || len(got.Participants) != 1 {
		t.Fatalf("has_gift=true should yield 1, got total=%d len=%d", got.Total, len(got.Participants))
	}
	if got.Participants[0].UserID != 222 {
		t.Fatalf("has_gift filter returned wrong participant: user_id=%d", got.Participants[0].UserID)
	}
}

// Список участников должен отдавать время предыдущего события только тем,
// у кого оно есть (по user_id), в секундах и в формате ЧЧ:ММ:СС.
func TestParticipantsHandlerPrevEventElapsedTime(t *testing.T) {
	got := getParticipantsList(t, newParticipantsListTestHandler(), "/api/events/77/participants")

	byUserID := make(map[int64]*dto.ParticipantDTO, len(got.Participants))
	for _, p := range got.Participants {
		byUserID[p.UserID] = p
	}

	withPrev := byUserID[111]
	if withPrev == nil || withPrev.PrevElapsedTimeSec == nil || withPrev.PrevElapsedTime == nil {
		t.Fatalf("participant 111 should have prev event time, got %+v", withPrev)
	}
	if *withPrev.PrevElapsedTimeSec != 7199 || *withPrev.PrevElapsedTime != "01:59:59" {
		t.Fatalf("prev event time mismatch: sec=%d formatted=%s", *withPrev.PrevElapsedTimeSec, *withPrev.PrevElapsedTime)
	}
	// У 111 есть и прошлое (7199), и текущее (7200) время → дельта −1 сек.
	if withPrev.PrevElapsedDeltaSec == nil || *withPrev.PrevElapsedDeltaSec != -1 {
		t.Fatalf("prev delta sec mismatch: %v", withPrev.PrevElapsedDeltaSec)
	}
	if withPrev.PrevElapsedDelta == nil || *withPrev.PrevElapsedDelta != "-00:00:01" {
		t.Fatalf("prev delta formatted mismatch: %v", withPrev.PrevElapsedDelta)
	}
	// Вычисленное (не ручное) значение не должно попадать в manual-поле.
	if withPrev.PrevElapsedTimeManualSec != nil {
		t.Fatalf("computed prev time must not be exposed as manual: %v", *withPrev.PrevElapsedTimeManualSec)
	}

	withoutPrev := byUserID[222]
	if withoutPrev == nil || withoutPrev.PrevElapsedTimeSec != nil || withoutPrev.PrevElapsedTime != nil {
		t.Fatalf("participant 222 should not have prev event time, got %+v", withoutPrev)
	}
	if withoutPrev.PrevElapsedDeltaSec != nil || withoutPrev.PrevElapsedDelta != nil {
		t.Fatalf("participant 222 should not have prev delta, got %+v", withoutPrev)
	}
}

// Ручное «время прошлого года» на участнике имеет приоритет над значением,
// вычисленным по предыдущему событию, и отдаётся отдельным manual-полем.
func TestParticipantsHandlerManualPrevElapsedTimeOverridesComputed(t *testing.T) {
	manual := 28800 // 08:00:00 вручную
	elapsed := 25500
	participantRepo := &participantListParticipantRepoFake{
		participants: []*entity.Participant{
			{ID: 1, UserID: 111, EventID: 77, BikeType: valueobject.BikeTypeGravel, Gender: valueobject.GenderMale,
				User:               &entity.User{ID: 111, Username: "alice"},
				PrevElapsedTimeSec: &manual,
				Result:             &entity.Result{ID: 9, ParticipantID: 1, IsCurrent: true, ElapsedTimeSec: &elapsed}},
		},
	}
	h := &ParticipantsHandler{
		participantRepo: participantRepo,
		// Вычисленное значение с прошлого события отличается — должно игнорироваться.
		resultRepo:             &participantListResultRepoFake{prevElapsedByUser: map[int64]int{111: 7199}},
		giftRepo:               &participantListGiftRepoFake{},
		getParticipantsHandler: query.NewGetParticipantsHandler(participantRepo),
	}

	got := getParticipantsList(t, h, "/api/events/77/participants")
	if len(got.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(got.Participants))
	}
	p := got.Participants[0]

	if p.PrevElapsedTimeSec == nil || *p.PrevElapsedTimeSec != manual {
		t.Fatalf("manual prev time should win: %v", p.PrevElapsedTimeSec)
	}
	if p.PrevElapsedTime == nil || *p.PrevElapsedTime != "08:00:00" {
		t.Fatalf("manual prev time formatted mismatch: %v", p.PrevElapsedTime)
	}
	if p.PrevElapsedTimeManualSec == nil || *p.PrevElapsedTimeManualSec != manual {
		t.Fatalf("manual field should be populated: %v", p.PrevElapsedTimeManualSec)
	}
	// Дельта считается от ручного значения: 28800 − 25500 = +3300 (+00:55:00).
	if p.PrevElapsedDeltaSec == nil || *p.PrevElapsedDeltaSec != 3300 {
		t.Fatalf("delta from manual mismatch: %v", p.PrevElapsedDeltaSec)
	}
	if p.PrevElapsedDelta == nil || *p.PrevElapsedDelta != "+00:55:00" {
		t.Fatalf("delta formatted mismatch: %v", p.PrevElapsedDelta)
	}
}

func TestParticipantsHandlerSearchFilter(t *testing.T) {
	got := getParticipantsList(t, newParticipantsListTestHandler(), "/api/events/77/participants?q=carol")
	if got.Total != 1 || len(got.Participants) != 1 {
		t.Fatalf("search q=carol should yield 1, got total=%d len=%d", got.Total, len(got.Participants))
	}
	if got.Participants[0].Username != "carol" {
		t.Fatalf("search returned wrong participant: %s", got.Participants[0].Username)
	}
}

// Список участников должен отдавать метрики заезда и вычисляемые поля для
// участника с результатом и опускать их для участника без результата.
func TestParticipantsHandlerListIncludesRideMetrics(t *testing.T) {
	started := time.Date(2026, 6, 21, 8, 0, 0, 0, time.UTC)
	finished := started.Add(2 * time.Hour)
	elapsed := 7200
	moving := 6600
	distance := 50000
	avgHR := 150
	maxHR := 180
	cadence := 85
	calories := 1200
	peak := 45.5

	withResult := &entity.Participant{
		ID:       1,
		UserID:   111,
		EventID:  77,
		BikeType: valueobject.BikeTypeGravel,
		Gender:   valueobject.GenderMale,
		User:     &entity.User{ID: 111, Username: "rider_with_result"},
		Result: &entity.Result{
			ID:             5,
			ParticipantID:  1,
			IsCurrent:      true,
			ElapsedTimeSec: &elapsed,
			MovingTimeSec:  &moving,
			StartedAt:      &started,
			FinishedAt:     &finished,
			DistanceMeters: &distance,
			AvgHeartRate:   &avgHR,
			MaxHeartRate:   &maxHR,
			PeakSpeedKmh:   &peak,
			AvgCadence:     &cadence,
			Calories:       &calories,
		},
	}
	withoutResult := &entity.Participant{
		ID:       2,
		UserID:   222,
		EventID:  77,
		BikeType: valueobject.BikeTypeMTB,
		Gender:   valueobject.GenderFemale,
		User:     &entity.User{ID: 222, Username: "rider_without_result"},
	}

	participantRepo := &participantListParticipantRepoFake{
		participants: []*entity.Participant{withResult, withoutResult},
	}
	h := &ParticipantsHandler{
		participantRepo:        participantRepo,
		resultRepo:             &participantListResultRepoFake{},
		giftRepo:               &participantListGiftRepoFake{},
		getParticipantsHandler: query.NewGetParticipantsHandler(participantRepo),
	}

	got := getParticipantsList(t, h, "/api/events/77/participants")

	byID := make(map[uint]*dto.ParticipantDTO, len(got.Participants))
	for _, p := range got.Participants {
		byID[p.ID] = p
	}

	wr := byID[1]
	if wr == nil {
		t.Fatal("participant with result missing from response")
	}
	if wr.StartedAt == nil {
		t.Error("started_at should be set for participant with result")
	}
	if wr.RideFinishedAt == nil {
		t.Error("ride_finished_at should be set for participant with result")
	}
	if wr.DistanceMeters == nil || *wr.DistanceMeters != distance {
		t.Errorf("distance_meters mismatch: got %v, want %d", wr.DistanceMeters, distance)
	}
	if wr.AvgHeartRate == nil || *wr.AvgHeartRate != avgHR {
		t.Errorf("avg_heart_rate mismatch: got %v, want %d", wr.AvgHeartRate, avgHR)
	}
	if wr.MaxHeartRate == nil || *wr.MaxHeartRate != maxHR {
		t.Errorf("max_heart_rate mismatch: got %v, want %d", wr.MaxHeartRate, maxHR)
	}
	if wr.PeakSpeedKmh == nil || *wr.PeakSpeedKmh != peak {
		t.Errorf("peak_speed_kmh mismatch: got %v, want %g", wr.PeakSpeedKmh, peak)
	}
	if wr.AvgCadence == nil || *wr.AvgCadence != cadence {
		t.Errorf("avg_cadence mismatch: got %v, want %d", wr.AvgCadence, cadence)
	}
	if wr.Calories == nil || *wr.Calories != calories {
		t.Errorf("calories mismatch: got %v, want %d", wr.Calories, calories)
	}
	if wr.RideDate == nil || *wr.RideDate != "2026-06-21" {
		t.Errorf("ride_date mismatch: got %v, want 2026-06-21", wr.RideDate)
	}
	if wr.AvgSpeedKmh == nil {
		t.Error("avg_speed_kmh should be computed for participant with result")
	}
	if wr.AvgMovingSpeedKmh == nil {
		t.Error("avg_moving_speed_kmh should be computed for participant with result")
	}
	// Пиковая − средняя: 45.5 − (50 км / 2 ч) = 20.5 км/ч.
	if wr.PeakAvgSpeedDeltaKmh == nil || *wr.PeakAvgSpeedDeltaKmh != 20.5 {
		t.Errorf("peak_avg_speed_delta_kmh mismatch: got %v, want 20.5", wr.PeakAvgSpeedDeltaKmh)
	}

	wo := byID[2]
	if wo == nil {
		t.Fatal("participant without result missing from response")
	}
	if wo.StartedAt != nil || wo.RideFinishedAt != nil || wo.DistanceMeters != nil ||
		wo.AvgHeartRate != nil || wo.PeakSpeedKmh != nil || wo.RideDate != nil ||
		wo.AvgSpeedKmh != nil || wo.AvgMovingSpeedKmh != nil || wo.PeakAvgSpeedDeltaKmh != nil {
		t.Error("participant without result should have nil ride metrics and computed fields")
	}
}

// Detail-эндпоинт должен отдавать «время прошлого года» и дельту так же,
// как список (вычисленное значение по предыдущему событию).
func TestParticipantsHandlerGetByIDIncludesPrevElapsedTime(t *testing.T) {
	elapsed := 7200
	participantRepo := &participantListParticipantRepoFake{
		participants: []*entity.Participant{
			{ID: 1, UserID: 111, EventID: 77, BikeType: valueobject.BikeTypeGravel, Gender: valueobject.GenderMale,
				User:   &entity.User{ID: 111, Username: "alice"},
				Result: &entity.Result{ID: 9, ParticipantID: 1, IsCurrent: true, ElapsedTimeSec: &elapsed}},
		},
	}
	resultRepo := &participantListResultRepoFake{prevElapsedByUser: map[int64]int{111: 7199}}
	giftRepo := &participantListGiftRepoFake{}
	h := &ParticipantsHandler{
		participantRepo:             participantRepo,
		resultRepo:                  resultRepo,
		giftRepo:                    giftRepo,
		getParticipantByIDHandler:   query.NewGetParticipantByIDHandler(participantRepo),
		getPrizeDistributionHandler: query.NewGetPrizeDistributionHandler(resultRepo, giftRepo, participantRepo, nil),
	}

	router := chi.NewRouter()
	router.Get("/api/participants/{id}", h.GetByID)
	req := httptest.NewRequest(http.MethodGet, "/api/participants/1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
	}

	var got dto.ParticipantDTO
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.PrevElapsedTimeSec == nil || *got.PrevElapsedTimeSec != 7199 {
		t.Fatalf("prev elapsed sec mismatch: %v", got.PrevElapsedTimeSec)
	}
	if got.PrevElapsedTime == nil || *got.PrevElapsedTime != "01:59:59" {
		t.Fatalf("prev elapsed formatted mismatch: %v", got.PrevElapsedTime)
	}
	if got.PrevElapsedDeltaSec == nil || *got.PrevElapsedDeltaSec != -1 {
		t.Fatalf("prev delta sec mismatch: %v", got.PrevElapsedDeltaSec)
	}
	if got.PrevElapsedDelta == nil || *got.PrevElapsedDelta != "-00:00:01" {
		t.Fatalf("prev delta formatted mismatch: %v", got.PrevElapsedDelta)
	}
	if got.PrevElapsedTimeManualSec != nil {
		t.Fatalf("computed prev time must not be exposed as manual: %v", *got.PrevElapsedTimeManualSec)
	}
}
