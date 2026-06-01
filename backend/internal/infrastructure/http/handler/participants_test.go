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
