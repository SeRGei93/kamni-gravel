package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestGiftsHandlerUpdateInvalidatesCacheOnApproval(t *testing.T) {
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	rr := giftUpdateRequest(t, h, 1, `{"review_status":"approved","criteria_ids":[5]}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if len(cacheFake.events) != 1 || cacheFake.events[0] != 77 {
		t.Fatalf("expected one invalidation for event 77, got %v", cacheFake.events)
	}
}

func TestGiftsHandlerUpdateInvalidatesCacheOnApprovedEdit(t *testing.T) {
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved, Description: "old"}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	// Правка уже одобренного подарка (без смены статуса) тоже меняет каталог.
	rr := giftUpdateRequest(t, h, 1, `{"description":"new description"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if len(cacheFake.events) != 1 || cacheFake.events[0] != 77 {
		t.Fatalf("expected invalidation for approved edit, got %v", cacheFake.events)
	}
}

func TestGiftsHandlerUpdateInvalidatesCacheOnUnapproval(t *testing.T) {
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	// Снятие одобрения убирает подарок из публичного каталога — кеш должен сброситься.
	rr := giftUpdateRequest(t, h, 1, `{"review_status":"pending_review"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if len(cacheFake.events) != 1 || cacheFake.events[0] != 77 {
		t.Fatalf("expected invalidation on unapproval, got %v", cacheFake.events)
	}
}

func TestGiftsHandlerUpdateDoesNotInvalidateForPendingGift(t *testing.T) {
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	rr := giftUpdateRequest(t, h, 1, `{"description":"still pending"}`)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(cacheFake.events) != 0 {
		t.Fatalf("pending gift edit must not invalidate, got %v", cacheFake.events)
	}
}

func TestGiftsHandlerUpdateDoesNotInvalidateCacheForRecipientOnlyChange(t *testing.T) {
	recipientID := uint(42)
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved, ManualDistribution: true}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	rr := giftUpdateRequest(t, h, 1, `{"manual_recipient_participant_id":42}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(cacheFake.events) != 0 {
		t.Fatalf("recipient-only update must not invalidate public catalog, got %v", cacheFake.events)
	}
	if giftRepo.gift.ManualRecipientParticipantID == nil || *giftRepo.gift.ManualRecipientParticipantID != recipientID {
		t.Fatalf("recipient was not persisted: %+v", giftRepo.gift)
	}
}

func TestGiftsHandlerUpdateInvalidatesCacheForManualDistributionChange(t *testing.T) {
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	rr := giftUpdateRequest(t, h, 1, `{"manual_distribution":true}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(cacheFake.events) != 1 || cacheFake.events[0] != 77 {
		t.Fatalf("manual distribution update must invalidate public catalog, got %v", cacheFake.events)
	}
}

func TestGiftsHandlerDeleteInvalidatesCacheForApprovedGift(t *testing.T) {
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	rr := giftDeleteRequest(t, h, 1)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(cacheFake.events) != 1 || cacheFake.events[0] != 77 {
		t.Fatalf("expected invalidation on approved delete, got %v", cacheFake.events)
	}
}

func TestGiftsHandlerDeleteDoesNotInvalidateForPendingGift(t *testing.T) {
	giftRepo := &invalidationGiftRepoFake{gift: &entity.Gift{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview}}
	cacheFake := &miniappGiftsCacheInvalidatorFake{}
	h := newInvalidationGiftsHandler(giftRepo, cacheFake)

	rr := giftDeleteRequest(t, h, 1)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
	}
	if len(cacheFake.events) != 0 {
		t.Fatalf("pending gift delete must not invalidate, got %v", cacheFake.events)
	}
}

func newInvalidationGiftsHandler(giftRepo *invalidationGiftRepoFake, cacheFake *miniappGiftsCacheInvalidatorFake) *GiftsHandler {
	criteriaRepo := &miniappHandlerCriteriaRepoFake{}
	return &GiftsHandler{
		giftRepo:           giftRepo,
		getGiftByIDHandler: query.NewGetGiftByIDHandler(giftRepo, criteriaRepo),
		updateGiftHandler:  command.NewUpdateGiftHandler(giftRepo, &invalidationParticipantRepoFake{participant: &entity.Participant{ID: 42, EventID: 77}}),
		giftsCache:         cacheFake,
	}
}

func giftUpdateRequest(t *testing.T, h *GiftsHandler, giftID uint, body string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Put("/api/gifts/{id}", h.Update)
	req := httptest.NewRequest(http.MethodPut, "/api/gifts/"+uintString(giftID), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func giftDeleteRequest(t *testing.T, h *GiftsHandler, giftID uint) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Delete("/api/gifts/{id}", h.Delete)
	req := httptest.NewRequest(http.MethodDelete, "/api/gifts/"+uintString(giftID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

type miniappGiftsCacheInvalidatorFake struct {
	events []uint
}

func (f *miniappGiftsCacheInvalidatorFake) InvalidateEvent(eventID uint) {
	f.events = append(f.events, eventID)
}

type invalidationGiftRepoFake struct {
	gift        *entity.Gift
	deleteCalls int
}

type invalidationParticipantRepoFake struct {
	repository.ParticipantRepository
	participant *entity.Participant
}

func (r *invalidationParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	return r.participant, nil
}

func (r *invalidationGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *invalidationGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	return nil
}
func (r *invalidationGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *invalidationGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *invalidationGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return r.gift, nil
}
func (r *invalidationGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *invalidationGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *invalidationGiftRepoFake) ListByEventPaged(ctx context.Context, eventID uint, reviewStatus *entity.GiftReviewStatus, limit, offset int) ([]*entity.Gift, int, error) {
	return nil, 0, nil
}
func (r *invalidationGiftRepoFake) CountsByReviewStatus(ctx context.Context, eventID uint) (map[string]int, error) {
	return nil, nil
}
func (r *invalidationGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *invalidationGiftRepoFake) Delete(ctx context.Context, id uint) error {
	r.deleteCalls++
	return nil
}
func (r *invalidationGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *invalidationGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}
