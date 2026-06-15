package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

func TestDecodeUpdateGiftRequestPlacePresence(t *testing.T) {
	withNull := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place":null}`))
	req, err := decodeUpdateGiftRequest(withNull)
	if err != nil {
		t.Fatalf("decode null place error: %v", err)
	}
	if !req.PlaceSet {
		t.Fatal("place field should be marked as present")
	}
	if req.Place != nil {
		t.Fatalf("null place should decode to nil, got %v", *req.Place)
	}

	omitted := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"description":"Gift"}`))
	req, err = decodeUpdateGiftRequest(omitted)
	if err != nil {
		t.Fatalf("decode omitted place error: %v", err)
	}
	if req.PlaceSet {
		t.Fatal("omitted place should not be marked as present")
	}
}

func TestGiftsHandlerNotifyPublicGiftApprovedUsesRetryNotifier(t *testing.T) {
	notifier := &giftPublicationNotifierFake{}
	h := &GiftsHandler{publicGiftNotifier: notifier}
	gift := &entity.Gift{ID: 10, EventID: 77, UserID: 123}

	h.notifyPublicGiftApproved(context.Background(), gift)

	if notifier.calls != 1 {
		t.Fatalf("notifier calls mismatch: got %d, want 1", notifier.calls)
	}
	if notifier.giftID != gift.ID {
		t.Fatalf("notified gift mismatch: got %d, want %d", notifier.giftID, gift.ID)
	}
}

func TestGiftsHandlerNotifyPublicGiftApprovedDoesNotReturnNotifierError(t *testing.T) {
	notifier := &giftPublicationNotifierFake{err: errors.New("telegram unavailable")}
	h := &GiftsHandler{publicGiftNotifier: notifier}

	h.notifyPublicGiftApproved(context.Background(), &entity.Gift{ID: 10, EventID: 77, UserID: 123})

	if notifier.calls != 1 {
		t.Fatalf("notifier calls mismatch: got %d, want 1", notifier.calls)
	}
}

func TestGiftsHandlerCreateManualGift(t *testing.T) {
	giftRepo := &createGiftRepoFake{}
	h := newCreateGiftTestHandler(
		&createGiftUserRepoFake{user: &entity.User{ID: 123}},
		&createGiftEventRepoFake{event: &entity.Event{ID: 77}},
		giftRepo,
		&createGiftBlacklistRepoFake{},
	)

	rr := giftCreateRequest(t, h, 77, CreateGiftRequest{
		UserID:         123,
		Description:    "  Bottle cage  ",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
	})

	if rr.Code != http.StatusCreated {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if giftRepo.createdGift == nil {
		t.Fatal("gift was not created")
	}
	if giftRepo.createdGift.UserID != 123 || giftRepo.createdGift.EventID != 77 {
		t.Fatalf("gift owner/event mismatch: got user=%d event=%d", giftRepo.createdGift.UserID, giftRepo.createdGift.EventID)
	}
	if giftRepo.createdGift.Description != "Bottle cage" {
		t.Fatalf("description mismatch: got %q", giftRepo.createdGift.Description)
	}
	if giftRepo.createdGift.ReviewStatus != entity.GiftReviewStatusPendingReview {
		t.Fatalf("review status mismatch: got %s", giftRepo.createdGift.ReviewStatus)
	}
}

func TestGiftsHandlerCreateRejectsMissingUserID(t *testing.T) {
	giftRepo := &createGiftRepoFake{}
	h := newCreateGiftTestHandler(
		&createGiftUserRepoFake{user: &entity.User{ID: 123}},
		&createGiftEventRepoFake{event: &entity.Event{ID: 77}},
		giftRepo,
		&createGiftBlacklistRepoFake{},
	)

	rr := giftCreateRequest(t, h, 77, CreateGiftRequest{Description: "Bottle cage"})

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if giftRepo.createdGift != nil {
		t.Fatal("gift should not be created without user_id")
	}
}

func TestGiftsHandlerCreateRejectsBlacklistedUser(t *testing.T) {
	giftRepo := &createGiftRepoFake{}
	h := newCreateGiftTestHandler(
		&createGiftUserRepoFake{user: &entity.User{ID: 123}},
		&createGiftEventRepoFake{event: &entity.Event{ID: 77}},
		giftRepo,
		&createGiftBlacklistRepoFake{blacklisted: true},
	)

	rr := giftCreateRequest(t, h, 77, CreateGiftRequest{UserID: 123, Description: "Bottle cage"})

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusForbidden, rr.Body.String())
	}
	if giftRepo.createdGift != nil {
		t.Fatal("gift should not be created for blacklisted user")
	}
}

func TestGiftsHandlerCreateReturnsNotFoundForUnknownUserOrEvent(t *testing.T) {
	tests := []struct {
		name  string
		user  *entity.User
		event *entity.Event
	}{
		{name: "unknown user", user: nil, event: &entity.Event{ID: 77}},
		{name: "missing event", user: &entity.User{ID: 123}, event: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			giftRepo := &createGiftRepoFake{}
			h := newCreateGiftTestHandler(
				&createGiftUserRepoFake{user: tt.user},
				&createGiftEventRepoFake{event: tt.event},
				giftRepo,
				&createGiftBlacklistRepoFake{},
			)

			rr := giftCreateRequest(t, h, 77, CreateGiftRequest{UserID: 123, Description: "Bottle cage"})

			if rr.Code != http.StatusNotFound {
				t.Fatalf("status mismatch: got %d, want %d body=%s", rr.Code, http.StatusNotFound, rr.Body.String())
			}
			if giftRepo.createdGift != nil {
				t.Fatal("gift should not be created")
			}
		})
	}
}

type giftPublicationNotifierFake struct {
	calls  int
	giftID uint
	err    error
}

func (n *giftPublicationNotifierFake) NotifyWithRetry(ctx context.Context, gift *entity.Gift) error {
	n.calls++
	if gift != nil {
		n.giftID = gift.ID
	}
	return n.err
}

func newCreateGiftTestHandler(
	userRepo *createGiftUserRepoFake,
	eventRepo *createGiftEventRepoFake,
	giftRepo *createGiftRepoFake,
	blacklistRepo *createGiftBlacklistRepoFake,
) *GiftsHandler {
	return &GiftsHandler{
		addGiftHandler: command.NewAddGiftHandler(userRepo, eventRepo, giftRepo, blacklistRepo),
	}
}

func giftCreateRequest(t *testing.T, h *GiftsHandler, eventID uint, bodyData CreateGiftRequest) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(bodyData)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	router := chi.NewRouter()
	router.Post("/api/events/{eventId}/gifts", h.Create)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+uintString(eventID)+"/gifts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

type createGiftUserRepoFake struct {
	user *entity.User
}

func (r *createGiftUserRepoFake) Create(ctx context.Context, user *entity.User) error { return nil }
func (r *createGiftUserRepoFake) Update(ctx context.Context, user *entity.User) error { return nil }
func (r *createGiftUserRepoFake) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	return r.user, nil
}
func (r *createGiftUserRepoFake) Delete(ctx context.Context, id int64) error { return nil }
func (r *createGiftUserRepoFake) GetAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

type createGiftEventRepoFake struct {
	event *entity.Event
}

func (r *createGiftEventRepoFake) Create(ctx context.Context, event *entity.Event) error {
	return nil
}
func (r *createGiftEventRepoFake) Update(ctx context.Context, event *entity.Event) error {
	return nil
}
func (r *createGiftEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *createGiftEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *createGiftEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *createGiftEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *createGiftEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type createGiftBlacklistRepoFake struct {
	blacklisted bool
}

func (r *createGiftBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *createGiftBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *createGiftBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	return r.blacklisted, nil
}
func (r *createGiftBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	return nil
}
func (r *createGiftBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *createGiftBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return nil
}

type createGiftRepoFake struct {
	createdGift *entity.Gift
}

func (r *createGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *createGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	gift.ID = 99
	r.createdGift = gift
	return nil
}
func (r *createGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *createGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *createGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *createGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *createGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *createGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *createGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *createGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *createGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}

func TestDecodeUpdateGiftRequestPlaceRulePresence(t *testing.T) {
	withNull := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place_rule":null}`))
	req, err := decodeUpdateGiftRequest(withNull)
	if err != nil {
		t.Fatalf("decode null place_rule error: %v", err)
	}
	if !req.PlaceRuleSet {
		t.Fatal("place_rule field should be marked as present")
	}
	if !req.PlaceRule.IsNone() {
		t.Fatalf("null place_rule should decode to none, got %s", req.PlaceRule.Type())
	}

	omitted := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"description":"Gift"}`))
	req, err = decodeUpdateGiftRequest(omitted)
	if err != nil {
		t.Fatalf("decode omitted place_rule error: %v", err)
	}
	if req.PlaceRuleSet {
		t.Fatal("omitted place_rule should not be marked as present")
	}
}

func TestDecodeUpdateGiftRequestStructuredPlaceRules(t *testing.T) {
	places := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place_rule":{"type":"places","places":[3,1,3]}}`))
	req, err := decodeUpdateGiftRequest(places)
	if err != nil {
		t.Fatalf("decode places rule error: %v", err)
	}
	assertDecodedGiftRulePlaces(t, req.PlaceRule, []int{1, 3})

	lastN := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place_rule":{"type":"last_n","last_count":5}}`))
	req, err = decodeUpdateGiftRequest(lastN)
	if err != nil {
		t.Fatalf("decode last_n rule error: %v", err)
	}
	if req.PlaceRule.Type() != valueobject.GiftPlaceRuleTypeLastN || req.PlaceRule.LastCount() != 5 {
		t.Fatalf("place_rule = %s/%d, want last_n/5", req.PlaceRule.Type(), req.PlaceRule.LastCount())
	}
}

func TestDecodeUpdateGiftRequestRejectsInvalidPlaceRule(t *testing.T) {
	tests := []string{
		`{"place_rule":{"type":"places","places":[]}}`,
		`{"place_rule":{"type":"places","places":[0]}}`,
		`{"place_rule":{"type":"last_n","last_count":0}}`,
		`{"place_rule":{"type":"last_n"}}`,
		`{"place_rule":{"type":"unknown"}}`,
	}

	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(body))
			if _, err := decodeUpdateGiftRequest(req); err == nil {
				t.Fatal("decodeUpdateGiftRequest() error = nil, want error")
			}
		})
	}
}

func TestDecodeUpdateGiftRequestPlaceRuleWinsOverLegacyPlacePayload(t *testing.T) {
	request := httptest.NewRequest("PUT", "/api/gifts/1", strings.NewReader(`{"place":2,"place_rule":{"type":"places","places":[10]}}`))
	req, err := decodeUpdateGiftRequest(request)
	if err != nil {
		t.Fatalf("decode request error: %v", err)
	}

	if !req.PlaceSet || req.Place == nil || *req.Place != 2 {
		t.Fatalf("legacy place decode mismatch: set=%t place=%v", req.PlaceSet, req.Place)
	}
	assertDecodedGiftRulePlaces(t, req.PlaceRule, []int{10})
}

func assertDecodedGiftRulePlaces(t *testing.T, rule valueobject.GiftPlaceRule, want []int) {
	t.Helper()

	got := rule.Places()
	if len(got) != len(want) {
		t.Fatalf("place_rule places = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("place_rule places = %v, want %v", got, want)
		}
	}
}
