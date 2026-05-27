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
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestUserBlacklistHandlerCreateAndList(t *testing.T) {
	repo := newHTTPUserBlacklistRepoFake()
	h := newUserBlacklistHTTPTestHandler(repo)

	createReq := httptest.NewRequest("POST", "/api/user-blacklist", strings.NewReader(`{"telegram_user_id":123,"reason":"spam"}`))
	createRR := httptest.NewRecorder()
	h.Create(createRR, createReq)

	if createRR.Code != http.StatusCreated {
		t.Fatalf("create status mismatch: got %d body=%s", createRR.Code, createRR.Body.String())
	}

	listReq := httptest.NewRequest("GET", "/api/user-blacklist", nil)
	listRR := httptest.NewRecorder()
	h.GetAll(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("list status mismatch: got %d body=%s", listRR.Code, listRR.Body.String())
	}

	var got struct {
		Entries []struct {
			TelegramUserID int64  `json:"telegram_user_id"`
			Reason         string `json:"reason"`
		} `json:"entries"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if got.Total != 1 || got.Entries[0].TelegramUserID != 123 || got.Entries[0].Reason != "spam" {
		t.Fatalf("list response mismatch: %#v", got)
	}
}

func TestUserBlacklistHandlerStatusMapping(t *testing.T) {
	repo := newHTTPUserBlacklistRepoFake()
	h := newUserBlacklistHTTPTestHandler(repo)

	t.Run("invalid create id", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/user-blacklist", strings.NewReader(`{"telegram_user_id":0}`))
		rr := httptest.NewRecorder()
		h.Create(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("update missing", func(t *testing.T) {
		req := userBlacklistPathRequest("PUT", "/api/user-blacklist/999", "999", `{"reason":"new"}`)
		rr := httptest.NewRecorder()
		h.Update(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("delete missing", func(t *testing.T) {
		req := userBlacklistPathRequest("DELETE", "/api/user-blacklist/999", "999", "")
		rr := httptest.NewRecorder()
		h.Delete(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status mismatch: got %d body=%s", rr.Code, rr.Body.String())
		}
	})
}

func newUserBlacklistHTTPTestHandler(repo repository.UserBlacklistRepository) *UserBlacklistHandler {
	return NewUserBlacklistHandler(
		query.NewListUserBlacklistHandler(repo),
		command.NewAddUserBlacklistHandler(repo),
		command.NewUpdateUserBlacklistReasonHandler(repo),
		command.NewRemoveUserBlacklistHandler(repo),
	)
}

func userBlacklistPathRequest(method, target, telegramUserID, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("telegramUserId", telegramUserID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func newHTTPUserBlacklistRepoFake() *httpUserBlacklistRepoFake {
	return &httpUserBlacklistRepoFake{entries: map[int64]*entity.UserBlacklist{}}
}

type httpUserBlacklistRepoFake struct {
	entries map[int64]*entity.UserBlacklist
}

func (r *httpUserBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	entries := make([]*entity.UserBlacklist, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}
	return entries, nil
}

func (r *httpUserBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	entry, ok := r.entries[telegramUserID]
	if !ok {
		return nil, repository.ErrUserBlacklistEntryNotFound
	}
	return entry, nil
}

func (r *httpUserBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	_, ok := r.entries[telegramUserID]
	return ok, nil
}

func (r *httpUserBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	now := time.Now()
	entry.CreatedAt = now
	entry.UpdatedAt = now
	r.entries[entry.TelegramUserID] = entry
	return nil
}

func (r *httpUserBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	entry, ok := r.entries[telegramUserID]
	if !ok {
		return nil, repository.ErrUserBlacklistEntryNotFound
	}
	entry.Reason = reason
	entry.UpdatedAt = time.Now()
	return entry, nil
}

func (r *httpUserBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	if _, ok := r.entries[telegramUserID]; !ok {
		return repository.ErrUserBlacklistEntryNotFound
	}
	delete(r.entries, telegramUserID)
	return nil
}
