package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/entity"

	"github.com/go-chi/chi/v5"
)

func TestGiftsHandlerCopyCreatesCopiesAndInvalidatesApprovedGiftCache(t *testing.T) {
	tests := []struct {
		name                 string
		reviewStatus         entity.GiftReviewStatus
		wantCacheInvalidated bool
	}{
		{name: "approved gift", reviewStatus: entity.GiftReviewStatusApproved, wantCacheInvalidated: true},
		{name: "pending gift", reviewStatus: entity.GiftReviewStatusPendingReview, wantCacheInvalidated: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copier := &giftCopyHandlerFake{
				result: &command.CopyGiftResult{
					EventID:      77,
					ReviewStatus: tt.reviewStatus,
					CreatedCount: 2,
				},
			}
			cache := &giftCopyCacheInvalidatorFake{}
			handler := &GiftsHandler{copyGiftHandler: copier, giftsCache: cache}

			rr := copyGiftRequestToHandler(t, handler, "5", `{"copies_count":2}`)

			if rr.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusCreated, rr.Body.String())
			}
			var response copyGiftResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.CreatedCount != 2 {
				t.Fatalf("created_count = %d, want 2", response.CreatedCount)
			}
			if len(copier.commands) != 1 || copier.commands[0].GiftID != 5 || copier.commands[0].CopiesCount != 2 {
				t.Fatalf("copy command = %#v, want gift 5 and 2 copies", copier.commands)
			}
			if got := len(cache.eventIDs); (got == 1) != tt.wantCacheInvalidated {
				t.Fatalf("cache invalidations = %v, want invalidated=%t", cache.eventIDs, tt.wantCacheInvalidated)
			}
		})
	}
}

func TestGiftsHandlerCopyMapsExpectedErrors(t *testing.T) {
	tests := []struct {
		name       string
		handlerErr error
		wantStatus int
		wantText   string
	}{
		{name: "invalid count", handlerErr: command.ErrInvalidGiftCopiesCount, wantStatus: http.StatusBadRequest},
		{name: "gift missing", handlerErr: command.ErrGiftNotFound, wantStatus: http.StatusNotFound},
		{name: "place constraint", handlerErr: command.ErrGiftCopyHasPlaceConstraint, wantStatus: http.StatusConflict, wantText: giftCopyPlaceConstraintMessage},
		{name: "unexpected failure", handlerErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copier := &giftCopyHandlerFake{err: tt.handlerErr}
			handler := &GiftsHandler{copyGiftHandler: copier}

			rr := copyGiftRequestToHandler(t, handler, "5", `{"copies_count":2}`)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if tt.wantText != "" && !strings.Contains(rr.Body.String(), tt.wantText) {
				t.Fatalf("response body = %s, want %q", rr.Body.String(), tt.wantText)
			}
		})
	}
}

func TestGiftsHandlerCopyRejectsUnknownFieldsAndAdditionalJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"copies_count":2,"unexpected":true}`},
		{name: "second JSON object", body: `{"copies_count":2}{"copies_count":3}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copier := &giftCopyHandlerFake{}
			handler := &GiftsHandler{copyGiftHandler: copier}

			rr := copyGiftRequestToHandler(t, handler, "5", tt.body)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
			}
			if len(copier.commands) != 0 {
				t.Fatalf("copy handler calls = %d, want 0", len(copier.commands))
			}
		})
	}
}

func TestGiftsHandlerCopyRejectsInvalidGiftID(t *testing.T) {
	copier := &giftCopyHandlerFake{}
	handler := &GiftsHandler{copyGiftHandler: copier}

	rr := copyGiftRequestToHandler(t, handler, "0", `{"copies_count":2}`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}
	if len(copier.commands) != 0 {
		t.Fatalf("copy handler calls = %d, want 0", len(copier.commands))
	}
}

func copyGiftRequestToHandler(t *testing.T, handler *GiftsHandler, giftID string, body string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	router.Post("/api/gifts/{id}/copies", handler.Copy)
	req := httptest.NewRequest(http.MethodPost, "/api/gifts/"+giftID+"/copies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

type giftCopyHandlerFake struct {
	result   *command.CopyGiftResult
	err      error
	commands []command.CopyGiftCommand
}

func (f *giftCopyHandlerFake) Handle(ctx context.Context, cmd command.CopyGiftCommand) (*command.CopyGiftResult, error) {
	f.commands = append(f.commands, cmd)
	return f.result, f.err
}

type giftCopyCacheInvalidatorFake struct {
	eventIDs []uint
}

func (f *giftCopyCacheInvalidatorFake) InvalidateEvent(eventID uint) {
	f.eventIDs = append(f.eventIDs, eventID)
}
