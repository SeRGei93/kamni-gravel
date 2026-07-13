package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
)

func TestGiftsHandlerAssignRandomRecipientInvalidatesCacheOnlyAfterAutomaticConversion(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		becameManual bool
		wantCache    bool
	}{
		{name: "automatic gift converted to manual", becameManual: true, wantCache: true},
		{name: "already manual gift", becameManual: false, wantCache: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cacheFake := &miniappGiftsCacheInvalidatorFake{}
			h := &GiftsHandler{
				assignRandomRecipient: &randomGiftRecipientAssignerFake{result: &command.AssignRandomAdminGiftRecipientResult{
					GiftID:                 1,
					EventID:                77,
					RecipientParticipantID: 10,
					BecameManual:           testCase.becameManual,
				}},
				giftsCache: cacheFake,
			}

			rr := assignRandomRecipientRequest(t, h, "1")
			if rr.Code != http.StatusNoContent || rr.Body.Len() != 0 {
				t.Fatalf("status/body = %d/%q, want 204 and empty", rr.Code, rr.Body.String())
			}
			if testCase.wantCache && (len(cacheFake.events) != 1 || cacheFake.events[0] != 77) {
				t.Fatalf("cache invalidations = %v, want [77]", cacheFake.events)
			}
			if !testCase.wantCache && len(cacheFake.events) != 0 {
				t.Fatalf("cache invalidations = %v, want none", cacheFake.events)
			}
		})
	}
}

func TestGiftsHandlerAssignRandomRecipientMapsErrorsAndUnavailableDependency(t *testing.T) {
	testCases := []struct {
		name       string
		assigner   randomGiftRecipientAssigner
		giftID     string
		wantStatus int
	}{
		{name: "unavailable", wantStatus: http.StatusInternalServerError, giftID: "1"},
		{name: "invalid ID", assigner: &randomGiftRecipientAssignerFake{}, giftID: "invalid", wantStatus: http.StatusBadRequest},
		{name: "not found", assigner: &randomGiftRecipientAssignerFake{err: command.ErrGiftNotFound}, giftID: "1", wantStatus: http.StatusNotFound},
		{name: "conflict", assigner: &randomGiftRecipientAssignerFake{err: command.ErrAdminRandomGiftAlreadyAssigned}, giftID: "1", wantStatus: http.StatusConflict},
		{name: "unexpected failure", assigner: &randomGiftRecipientAssignerFake{err: errors.New("database unavailable")}, giftID: "1", wantStatus: http.StatusInternalServerError},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			h := &GiftsHandler{assignRandomRecipient: testCase.assigner}
			rr := assignRandomRecipientRequest(t, h, testCase.giftID)
			if rr.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d body=%s", rr.Code, testCase.wantStatus, rr.Body.String())
			}
		})
	}
}

func assignRandomRecipientRequest(t *testing.T, h *GiftsHandler, giftID string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.Post("/api/gifts/{id}/random-recipient", h.AssignRandomRecipient)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/gifts/"+giftID+"/random-recipient", nil))
	return rr
}

type randomGiftRecipientAssignerFake struct {
	result *command.AssignRandomAdminGiftRecipientResult
	err    error
}

func (f *randomGiftRecipientAssignerFake) Handle(context.Context, command.AssignRandomAdminGiftRecipientCommand) (*command.AssignRandomAdminGiftRecipientResult, error) {
	return f.result, f.err
}
