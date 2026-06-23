package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/lock"
	"gravel_bot/internal/pkg/jwt"
)

// lockResultRepoFake — частичный фейк ResultRepository: переопределяет только
// FindByID, чтобы резолвер result->participant вернул заданного участника.
type lockResultRepoFake struct {
	repository.ResultRepository
	participantID uint
}

func (f *lockResultRepoFake) FindByID(ctx context.Context, id uint) (*entity.Result, error) {
	return &entity.Result{ID: id, ParticipantID: f.participantID}, nil
}

// authReq строит запрос с claims администратора в контексте (как это делает
// middleware.Auth в проде), без реального JWT.
func authReq(method, target string, userID uint, username string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	claims := &jwt.Claims{UserID: userID, Username: username, Role: "admin"}
	return req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, claims))
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func decodeLockView(t *testing.T, rr *httptest.ResponseRecorder) middleware.LockStatusView {
	t.Helper()
	var view middleware.LockStatusView
	if err := json.Unmarshal(rr.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode lock view: %v body=%s", err, rr.Body.String())
	}
	return view
}

func TestParticipantLockHandler_AcquireReleaseStatus(t *testing.T) {
	mgr := lock.NewManager(lock.DefaultTTL)
	h := NewParticipantLockHandler(mgr)

	r := chi.NewRouter()
	r.Post("/api/participants/{id}/lock", h.Acquire)
	r.Delete("/api/participants/{id}/lock", h.Release)
	r.Get("/api/participants/{id}/lock", h.Status)

	// Alice захватывает лок.
	rr := serve(r, authReq(http.MethodPost, "/api/participants/7/lock", 10, "alice"))
	if rr.Code != http.StatusOK {
		t.Fatalf("alice acquire: got %d, want 200 body=%s", rr.Code, rr.Body.String())
	}
	view := decodeLockView(t, rr)
	if !view.Locked || !view.IsMine || view.LockedByUsername != "alice" {
		t.Fatalf("alice acquire view mismatch: %+v", view)
	}

	// Bob получает 409 с именем владельца.
	rr = serve(r, authReq(http.MethodPost, "/api/participants/7/lock", 20, "bob"))
	if rr.Code != http.StatusConflict {
		t.Fatalf("bob acquire: got %d, want 409 body=%s", rr.Code, rr.Body.String())
	}
	view = decodeLockView(t, rr)
	if !view.Locked || view.IsMine || view.LockedByUsername != "alice" {
		t.Fatalf("bob conflict view mismatch: %+v", view)
	}

	// Alice повторно захватывает (heartbeat) — снова 200.
	rr = serve(r, authReq(http.MethodPost, "/api/participants/7/lock", 10, "alice"))
	if rr.Code != http.StatusOK {
		t.Fatalf("alice heartbeat: got %d, want 200", rr.Code)
	}

	// Bob видит статус: занято не им.
	rr = serve(r, authReq(http.MethodGet, "/api/participants/7/lock", 20, "bob"))
	if rr.Code != http.StatusOK {
		t.Fatalf("bob status: got %d, want 200", rr.Code)
	}
	view = decodeLockView(t, rr)
	if !view.Locked || view.IsMine || view.LockedByUsername != "alice" {
		t.Fatalf("bob status view mismatch: %+v", view)
	}

	// Bob не может снять чужой лок (идемпотентный 204), лок остаётся.
	rr = serve(r, authReq(http.MethodDelete, "/api/participants/7/lock", 20, "bob"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("bob release: got %d, want 204", rr.Code)
	}
	if _, ok := mgr.Get(7); !ok {
		t.Fatal("lock must remain after a non-owner release")
	}

	// Alice снимает лок.
	rr = serve(r, authReq(http.MethodDelete, "/api/participants/7/lock", 10, "alice"))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("alice release: got %d, want 204", rr.Code)
	}

	// Теперь свободно — статус locked=false, Bob может захватить.
	rr = serve(r, authReq(http.MethodGet, "/api/participants/7/lock", 20, "bob"))
	view = decodeLockView(t, rr)
	if view.Locked {
		t.Fatalf("expected free lock after release, got %+v", view)
	}
	rr = serve(r, authReq(http.MethodPost, "/api/participants/7/lock", 20, "bob"))
	if rr.Code != http.StatusOK {
		t.Fatalf("bob acquire after release: got %d, want 200", rr.Code)
	}
}

func TestRequireParticipantUnlocked_Enforcement(t *testing.T) {
	mgr := lock.NewManager(lock.DefaultTTL)
	resultRepo := &lockResultRepoFake{participantID: 5}
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	r := chi.NewRouter()
	r.With(middleware.RequireParticipantUnlocked(mgr, middleware.ParticipantIDFromParam("id"))).
		Put("/api/participants/{id}", ok)
	r.With(middleware.RequireParticipantUnlocked(mgr, middleware.ParticipantIDFromResult(resultRepo))).
		Put("/api/results/{id}", ok)

	// Alice (user 10) держит лок участника 5.
	mgr.Acquire(5, 10, "alice")

	// Bob заблокирован на прямом участнике и на маршруте результата.
	if rr := serve(r, authReq(http.MethodPut, "/api/participants/5", 20, "bob")); rr.Code != http.StatusConflict {
		t.Fatalf("bob PUT participant: got %d, want 409", rr.Code)
	}
	if rr := serve(r, authReq(http.MethodPut, "/api/results/99", 20, "bob")); rr.Code != http.StatusConflict {
		t.Fatalf("bob PUT result: got %d, want 409", rr.Code)
	}

	// Alice (владелец) проходит на обоих маршрутах.
	if rr := serve(r, authReq(http.MethodPut, "/api/participants/5", 10, "alice")); rr.Code != http.StatusOK {
		t.Fatalf("alice PUT participant: got %d, want 200", rr.Code)
	}
	if rr := serve(r, authReq(http.MethodPut, "/api/results/99", 10, "alice")); rr.Code != http.StatusOK {
		t.Fatalf("alice PUT result: got %d, want 200", rr.Code)
	}

	// После снятия лока Bob проходит.
	mgr.Release(5, 10)
	if rr := serve(r, authReq(http.MethodPut, "/api/participants/5", 20, "bob")); rr.Code != http.StatusOK {
		t.Fatalf("bob PUT after release: got %d, want 200", rr.Code)
	}
}
