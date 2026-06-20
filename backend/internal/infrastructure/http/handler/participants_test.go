package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

type participantListResultRepoFake struct {
	repository.ResultRepository
}

func (r *participantListResultRepoFake) FindByEventWithPlaces(ctx context.Context, eventID uint) ([]*repository.ResultWithPlace, error) {
	return nil, nil
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

func newParticipantsListTestHandler() *ParticipantsHandler {
	participantRepo := &participantListParticipantRepoFake{
		participants: []*entity.Participant{
			{ID: 1, UserID: 111, EventID: 77, BikeType: valueobject.BikeTypeGravel, Gender: valueobject.GenderMale, User: &entity.User{ID: 111, Username: "alice"}},
			{ID: 2, UserID: 222, EventID: 77, BikeType: valueobject.BikeTypeMTB, Gender: valueobject.GenderFemale, User: &entity.User{ID: 222, Username: "bob_with_gift"}},
			{ID: 3, UserID: 333, EventID: 77, BikeType: valueobject.BikeTypeGravel, Gender: valueobject.GenderMale, User: &entity.User{ID: 333, Username: "carol"}},
		},
	}
	giftRepo := &participantListGiftRepoFake{
		gifts: []*entity.Gift{{ID: 10, UserID: 222, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview}},
	}
	return &ParticipantsHandler{
		participantRepo:        participantRepo,
		resultRepo:             &participantListResultRepoFake{},
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

func TestParticipantsHandlerHasGiftFilter(t *testing.T) {
	got := getParticipantsList(t, newParticipantsListTestHandler(), "/api/events/77/participants?page=1&page_size=50&has_gift=true")
	if got.Total != 1 || len(got.Participants) != 1 {
		t.Fatalf("has_gift=true should yield 1, got total=%d len=%d", got.Total, len(got.Participants))
	}
	if got.Participants[0].UserID != 222 {
		t.Fatalf("has_gift filter returned wrong participant: user_id=%d", got.Participants[0].UserID)
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
