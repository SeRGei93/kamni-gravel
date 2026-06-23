package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/http/response"
	"gravel_bot/internal/infrastructure/lock"
)

// ParticipantLockHandler exposes acquire/release/status for participant edit
// locks. It is a thin transport layer over the in-memory lock manager.
type ParticipantLockHandler struct {
	lockManager *lock.Manager
}

// NewParticipantLockHandler создаёт новый handler блокировок участников
func NewParticipantLockHandler(lockManager *lock.Manager) *ParticipantLockHandler {
	return &ParticipantLockHandler{lockManager: lockManager}
}

func (h *ParticipantLockHandler) parseID(w http.ResponseWriter, r *http.Request) (uint, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return 0, false
	}
	return uint(id), true
}

// Acquire обрабатывает POST /api/participants/:id/lock — захват или продление
// блокировки. Также служит heartbeat: повторный захват владельцем продлевает TTL.
func (h *ParticipantLockHandler) Acquire(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	l, acquired := h.lockManager.Acquire(id, claims.UserID, claims.Username)
	view := middleware.NewLockStatusView(id, claims.UserID, l)
	if !acquired {
		log.Printf("Participant lock acquire conflict: participant_id=%d actor=%s owner=%s",
			id, claims.Username, l.OwnerUsername)
		response.JSON(w, http.StatusConflict, view)
		return
	}

	log.Printf("Participant lock acquired: participant_id=%d actor=%s", id, claims.Username)
	response.Success(w, view)
}

// Release обрабатывает DELETE /api/participants/:id/lock — снятие блокировки.
// Идемпотентно: попытка снять чужую/отсутствующую блокировку всё равно вернёт
// 204, чтобы упростить очистку на стороне клиента (cancel / beforeunload).
func (h *ParticipantLockHandler) Release(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	released := h.lockManager.Release(id, claims.UserID)
	log.Printf("Participant lock release: participant_id=%d actor=%s released=%t", id, claims.Username, released)
	response.NoContent(w)
}

// Status обрабатывает GET /api/participants/:id/lock — текущее состояние блокировки.
func (h *ParticipantLockHandler) Status(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		response.Unauthorized(w, "User not found in context")
		return
	}

	l, _ := h.lockManager.Get(id)
	view := middleware.NewLockStatusView(id, claims.UserID, l)
	log.Printf("DEBUG Participant lock status: participant_id=%d actor=%s locked=%t", id, claims.Username, view.Locked)
	response.Success(w, view)
}
