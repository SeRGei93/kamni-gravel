package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/http/middleware"
)

func TestMiniappSessionReturnsTelegramUserAndActiveEvent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(&miniappEventRepoFake{
		activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Description: "Race gifts", Active: true},
	}, nil, nil)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got MiniappSessionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.User.ID != 42 || got.User.FirstName != "Alex" {
		t.Fatalf("user mismatch: %#v", got.User)
	}
	if got.Event.ID != 77 || got.Event.Name != "Gravel Race" {
		t.Fatalf("event mismatch: %#v", got.Event)
	}
}

func TestMiniappSessionReturnsNotFoundWhenNoActiveEvent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(&miniappEventRepoFake{}, nil, nil)

	rr := miniappRequest(t, token, now, h.Session, "/api/miniapp/session")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
	}
}

func TestMiniappGiftsUsesActiveEventAndApprovedCatalog(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	giftRepo := &miniappHandlerGiftRepoFake{
		gifts: []*entity.Gift{
			{
				ID:             1,
				UserID:         123,
				EventID:        77,
				Description:    "Bottle cage",
				GenderFilter:   "male",
				BikeTypeFilter: "gravel",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 123, FirstName: "Alex"},
			},
			{
				ID:             2,
				UserID:         124,
				EventID:        77,
				Description:    "Women prize",
				GenderFilter:   "female",
				BikeTypeFilter: "gravel",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 124, FirstName: "Kate"},
			},
		},
		attachments: map[uint][]*entity.GiftAttachment{
			1: {{ID: 10, GiftID: 1, TelegramFileID: "file-1", FileType: "photo"}},
		},
	}
	criteriaRepo := &miniappHandlerCriteriaRepoFake{
		criteriaByGift: map[uint][]*entity.Criteria{
			1: {{ID: 5, Name: "Speed"}},
		},
	}
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		giftRepo,
		criteriaRepo,
	)

	rr := miniappRequest(t, token, now, h.Gifts, "/api/miniapp/gifts?gender=male&bike_type=gravel")

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if !giftRepo.findByStatusCalled {
		t.Fatal("FindByEventAndReviewStatus was not called")
	}
	if giftRepo.eventID != 77 || giftRepo.reviewStatus != entity.GiftReviewStatusApproved {
		t.Fatalf("approved catalog query mismatch: event_id=%d review_status=%s", giftRepo.eventID, giftRepo.reviewStatus)
	}

	var got struct {
		Gifts []struct {
			ID          uint   `json:"id"`
			Description string `json:"description"`
			Attachments []struct {
				TelegramFileID string `json:"telegram_file_id"`
			} `json:"attachments"`
			Criteria []struct {
				ID uint `json:"id"`
			} `json:"criteria"`
		} `json:"gifts"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 1 || len(got.Gifts) != 1 || got.Gifts[0].ID != 1 {
		t.Fatalf("gift list mismatch: %#v", got)
	}
	if len(got.Gifts[0].Attachments) != 1 || got.Gifts[0].Attachments[0].TelegramFileID != "file-1" {
		t.Fatalf("attachments mismatch: %#v", got.Gifts[0].Attachments)
	}
	if len(got.Gifts[0].Criteria) != 1 || got.Gifts[0].Criteria[0].ID != 5 {
		t.Fatalf("criteria mismatch: %#v", got.Gifts[0].Criteria)
	}
}

func TestMiniappGiftsRejectsInvalidFilters(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	giftRepo := &miniappHandlerGiftRepoFake{}
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		giftRepo,
		&miniappHandlerCriteriaRepoFake{},
	)

	rr := miniappRequest(t, token, now, h.Gifts, "/api/miniapp/gifts?gender=everyone&bike_type=gravel")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if giftRepo.findByStatusCalled {
		t.Fatal("repository should not be called for invalid filters")
	}
}

func TestMiniappTelegramFileStreamsContent(t *testing.T) {
	const token = "123456:secret"
	now := time.Unix(1_700_000_000, 0).UTC()
	h := newMiniappTestHandler(
		&miniappEventRepoFake{activeEvent: &entity.Event{ID: 77, Name: "Gravel Race", Active: true}},
		nil,
		nil,
	)
	h.fileFetcher = miniappFileFetcherFunc(func(ctx context.Context, fileID string) (*http.Response, error) {
		if fileID != "photo-1" {
			t.Fatalf("file ID mismatch: got %q", fileID)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type":   {"image/jpeg"},
				"Content-Length": {"5"},
			},
			Body: io.NopCloser(strings.NewReader("image")),
		}, nil
	})

	router := chi.NewRouter()
	router.Use(middleware.TelegramWebAppAuthWithConfig(middleware.TelegramWebAppAuthConfig{
		BotToken: token,
		Now:      func() time.Time { return now },
	}))
	router.Get("/api/miniapp/telegram/files/{fileId}", h.TelegramFile)

	req := httptest.NewRequest(http.MethodGet, "/api/miniapp/telegram/files/photo-1", nil)
	req.Header.Set(middleware.TelegramInitDataHeader, signedMiniappInitData(t, token, now))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("content type mismatch: got %q", got)
	}
	if got := rr.Body.String(); got != "image" {
		t.Fatalf("body mismatch: got %q", got)
	}
}

func newMiniappTestHandler(
	eventRepo *miniappEventRepoFake,
	giftRepo *miniappHandlerGiftRepoFake,
	criteriaRepo *miniappHandlerCriteriaRepoFake,
) *MiniappHandler {
	if giftRepo == nil {
		giftRepo = &miniappHandlerGiftRepoFake{}
	}
	if criteriaRepo == nil {
		criteriaRepo = &miniappHandlerCriteriaRepoFake{}
	}

	return newMiniappHandlerWithFileFetcher(
		eventRepo,
		query.NewGetMiniappGiftsHandler(giftRepo, criteriaRepo),
		miniappFileFetcherFunc(func(ctx context.Context, fileID string) (*http.Response, error) {
			return nil, fmt.Errorf("unexpected file fetch: %s", fileID)
		}),
	)
}

func miniappRequest(t *testing.T, token string, now time.Time, handlerFunc http.HandlerFunc, target string) *httptest.ResponseRecorder {
	t.Helper()

	handler := middleware.TelegramWebAppAuthWithConfig(middleware.TelegramWebAppAuthConfig{
		BotToken: token,
		Now:      func() time.Time { return now },
	})(handlerFunc)

	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set(middleware.TelegramInitDataHeader, signedMiniappInitData(t, token, now))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

type miniappFileFetcherFunc func(ctx context.Context, fileID string) (*http.Response, error)

func (f miniappFileFetcherFunc) Fetch(ctx context.Context, fileID string) (*http.Response, error) {
	return f(ctx, fileID)
}

type miniappEventRepoFake struct {
	activeEvent *entity.Event
}

func (r *miniappEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *miniappEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *miniappEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return nil, nil
}
func (r *miniappEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *miniappEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.activeEvent, nil
}
func (r *miniappEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *miniappEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type miniappHandlerGiftRepoFake struct {
	findByStatusCalled bool
	eventID            uint
	reviewStatus       entity.GiftReviewStatus
	gifts              []*entity.Gift
	attachments        map[uint][]*entity.GiftAttachment
}

func (r *miniappHandlerGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *miniappHandlerGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *miniappHandlerGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	r.findByStatusCalled = true
	r.eventID = eventID
	r.reviewStatus = reviewStatus
	return r.gifts, nil
}
func (r *miniappHandlerGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *miniappHandlerGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *miniappHandlerGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *miniappHandlerGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	if r.attachments == nil {
		return nil, nil
	}
	return r.attachments[giftID], nil
}

type miniappHandlerCriteriaRepoFake struct {
	criteriaByGift map[uint][]*entity.Criteria
}

func (r *miniappHandlerCriteriaRepoFake) Create(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *miniappHandlerCriteriaRepoFake) Update(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *miniappHandlerCriteriaRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByID(ctx context.Context, id uint) (*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappHandlerCriteriaRepoFake) FindAll(ctx context.Context) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByType(ctx context.Context, criteriaType valueobject.CriteriaType) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	if r.criteriaByGift == nil {
		return nil, nil
	}
	return r.criteriaByGift[giftID], nil
}
func (r *miniappHandlerCriteriaRepoFake) FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error) {
	return nil, nil
}

func signedMiniappInitData(t *testing.T, token string, now time.Time) string {
	t.Helper()

	values := url.Values{
		"auth_date": {strconv.FormatInt(now.Unix(), 10)},
		"query_id":  {"query-1"},
		"user":      {`{"id":42,"first_name":"Alex","username":"alex"}`},
	}
	values.Set("hash", miniappHash(token, values))
	return values.Encode()
}

func miniappHash(token string, values url.Values) string {
	pairs := make([]string, 0, len(values))
	for key, value := range values {
		if key == "hash" {
			continue
		}
		pairs = append(pairs, key+"="+value[0])
	}
	sort.Strings(pairs)

	secretHMAC := hmac.New(sha256.New, []byte("WebAppData"))
	secretHMAC.Write([]byte(token))
	secret := secretHMAC.Sum(nil)

	dataHMAC := hmac.New(sha256.New, secret)
	dataHMAC.Write([]byte(strings.Join(pairs, "\n")))
	return fmt.Sprintf("%x", dataHMAC.Sum(nil))
}
