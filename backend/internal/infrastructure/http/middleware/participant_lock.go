package middleware

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
	"gravel_bot/internal/infrastructure/lock"
)

// LockStatusView is the JSON shape returned for participant edit-lock state by
// both the lock handler (acquire/release/status) and this enforcement
// middleware, so the frontend parses a single contract. It lives here (and not
// in the handler package) because the handler may import middleware but not the
// other way around — keeping the shared type cycle-free.
type LockStatusView struct {
	ParticipantID    uint   `json:"participant_id"`
	Locked           bool   `json:"locked"`
	LockedByUserID   uint   `json:"locked_by_user_id,omitempty"`
	LockedByUsername string `json:"locked_by_username,omitempty"`
	AcquiredAt       string `json:"acquired_at,omitempty"`
	ExpiresAt        string `json:"expires_at,omitempty"`
	IsMine           bool   `json:"is_mine"`
}

// NewLockStatusView builds a view for participantID relative to currentUserID.
// A nil lock means "not locked".
func NewLockStatusView(participantID, currentUserID uint, l *lock.Lock) LockStatusView {
	if l == nil {
		return LockStatusView{ParticipantID: participantID, Locked: false}
	}
	return LockStatusView{
		ParticipantID:    participantID,
		Locked:           true,
		LockedByUserID:   l.OwnerUserID,
		LockedByUsername: l.OwnerUsername,
		AcquiredAt:       l.AcquiredAt.UTC().Format(time.RFC3339),
		ExpiresAt:        l.ExpiresAt.UTC().Format(time.RFC3339),
		IsMine:           l.OwnerUserID == currentUserID,
	}
}

// ParticipantIDResolver maps a request to the participant id whose lock guards
// the write. Different routes carry the id differently (participant id in the
// path, or a result id that must be resolved to its participant).
type ParticipantIDResolver func(r *http.Request) (uint, error)

// ParticipantIDFromParam reads the participant id directly from a chi URL param
// (e.g. "id" for /participants/{id}, "participantId" for the nested results route).
func ParticipantIDFromParam(name string) ParticipantIDResolver {
	return func(r *http.Request) (uint, error) {
		id, err := strconv.ParseUint(chi.URLParam(r, name), 10, 32)
		if err != nil {
			return 0, err
		}
		return uint(id), nil
	}
}

// ParticipantIDFromResult resolves the owning participant id from a result id in
// the "id" URL param, via the result repository.
func ParticipantIDFromResult(resultRepo repository.ResultRepository) ParticipantIDResolver {
	return func(r *http.Request) (uint, error) {
		id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
		if err != nil {
			return 0, err
		}
		result, err := resultRepo.FindByID(r.Context(), uint(id))
		if err != nil {
			return 0, err
		}
		return result.ParticipantID, nil
	}
}

// RequireParticipantUnlocked rejects a write with 409 Conflict when the target
// participant is locked by another admin. It must run after Auth so the JWT
// claims are present in the request context.
//
// When the resolver cannot map the request to a participant (malformed id,
// missing result, transient repo error) the guard fails open: it logs a WARN
// and lets the request through so the downstream handler returns its own proper
// status. The lock is an advisory coordination mechanism between trusted admins,
// not a security boundary, so there is nothing meaningful to guard for a target
// that does not resolve.
func RequireParticipantUnlocked(mgr *lock.Manager, resolve ParticipantIDResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				response.Unauthorized(w, "User not found in context")
				return
			}

			pid, err := resolve(r)
			if err != nil {
				log.Printf("WARN Participant lock guard could not resolve participant: path=%s actor=%s error=%v",
					r.URL.Path, claims.Username, err)
				next.ServeHTTP(w, r)
				return
			}

			if l, locked := mgr.LockedByOther(pid, claims.UserID); locked {
				log.Printf("DEBUG Participant lock guard denied: participant_id=%d actor=%s owner=%s path=%s",
					pid, claims.Username, l.OwnerUsername, r.URL.Path)
				response.JSON(w, http.StatusConflict, NewLockStatusView(pid, claims.UserID, l))
				return
			}

			log.Printf("DEBUG Participant lock guard allowed: participant_id=%d actor=%s path=%s",
				pid, claims.Username, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
