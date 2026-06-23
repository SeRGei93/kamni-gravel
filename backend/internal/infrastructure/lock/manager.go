// Package lock provides an in-memory, TTL-based pessimistic edit lock for
// participant records. It lets the admin panel guarantee that only one admin
// edits a given participant at a time.
//
// The implementation mirrors the Telegram session manager pattern
// (internal/infrastructure/telegram/session): a mutex-guarded map plus a
// background goroutine that periodically purges expired entries. State is
// process-local, so it is lost on restart and not shared across instances —
// acceptable for the single-instance admin panel. The short TTL means a "stuck"
// lock (crashed tab, closed browser) self-heals quickly.
package lock

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	// DefaultTTL is how long a participant edit lock stays valid without a
	// refresh. The frontend heartbeats well within this window.
	DefaultTTL = 90 * time.Second

	// cleanupInterval is how often expired locks are purged from memory.
	// Expiry is also checked lazily on every read, so this is just hygiene.
	cleanupInterval = 60 * time.Second
)

// Lock represents an exclusive edit lock held by an admin over a participant.
type Lock struct {
	ParticipantID uint
	OwnerUserID   uint
	OwnerUsername string
	AcquiredAt    time.Time
	ExpiresAt     time.Time
}

// clone returns a copy so callers cannot mutate the manager's internal state.
func (l *Lock) clone() *Lock {
	c := *l
	return &c
}

// Manager keeps in-memory edit locks keyed by participant id.
type Manager struct {
	locks map[uint]*Lock
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewManager creates a Manager with the given TTL and starts its background
// cleanup loop. A non-positive ttl falls back to DefaultTTL.
func NewManager(ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	m := &Manager{
		locks: make(map[uint]*Lock),
		ttl:   ttl,
	}

	// Запускаем очистку устаревших блокировок
	go m.cleanupLoop(context.Background())

	log.Printf("Participant edit lock manager started: ttl=%s", ttl)
	return m
}

// Acquire grants or refreshes a lock for participantID on behalf of the given
// admin. It succeeds when the participant is free, the existing lock has
// expired (the lock is then stolen), or the requester already owns it (a
// refresh — this is also the heartbeat path). When the participant is locked by
// another admin and the lock is still valid, it returns the current lock and
// false.
func (m *Manager) Acquire(participantID, ownerUserID uint, ownerUsername string) (*Lock, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	existing, ok := m.locks[participantID]

	// Held by another admin and still valid — deny.
	if ok && now.Before(existing.ExpiresAt) && existing.OwnerUserID != ownerUserID {
		log.Printf("DEBUG Participant lock blocked: participant_id=%d requested_by=%s owner=%s",
			participantID, ownerUsername, existing.OwnerUsername)
		return existing.clone(), false
	}

	// Preserve the original acquisition time when the same owner refreshes.
	acquiredAt := now
	if ok && existing.OwnerUserID == ownerUserID && now.Before(existing.ExpiresAt) {
		acquiredAt = existing.AcquiredAt
	} else if ok && existing.OwnerUserID != ownerUserID {
		log.Printf("DEBUG Participant lock stolen (expired): participant_id=%d new_owner=%s previous_owner=%s",
			participantID, ownerUsername, existing.OwnerUsername)
	}

	l := &Lock{
		ParticipantID: participantID,
		OwnerUserID:   ownerUserID,
		OwnerUsername: ownerUsername,
		AcquiredAt:    acquiredAt,
		ExpiresAt:     now.Add(m.ttl),
	}
	m.locks[participantID] = l
	log.Printf("DEBUG Participant lock acquired/refreshed: participant_id=%d owner=%s expires_at=%s",
		participantID, ownerUsername, l.ExpiresAt.Format(time.RFC3339))
	return l.clone(), true
}

// Refresh extends an existing lock owned by ownerUserID (heartbeat). It returns
// false when the lock is absent, expired, or owned by someone else. Acquire by
// the owner has the same effect; Refresh exists for callers that must not steal
// an expired lock.
func (m *Manager) Refresh(participantID, ownerUserID uint) (*Lock, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	existing, ok := m.locks[participantID]
	if !ok || !now.Before(existing.ExpiresAt) || existing.OwnerUserID != ownerUserID {
		return nil, false
	}
	existing.ExpiresAt = now.Add(m.ttl)
	log.Printf("DEBUG Participant lock refreshed: participant_id=%d owner=%s", participantID, existing.OwnerUsername)
	return existing.clone(), true
}

// Release removes the lock if it is owned by ownerUserID. It returns false when
// the lock is absent or owned by someone else.
func (m *Manager) Release(participantID, ownerUserID uint) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.locks[participantID]
	if !ok {
		return false
	}
	if existing.OwnerUserID != ownerUserID {
		log.Printf("WARN Participant lock release denied (not owner): participant_id=%d requested_by_user_id=%d owner=%s",
			participantID, ownerUserID, existing.OwnerUsername)
		return false
	}
	delete(m.locks, participantID)
	log.Printf("DEBUG Participant lock released: participant_id=%d owner=%s", participantID, existing.OwnerUsername)
	return true
}

// Get returns the active (non-expired) lock for participantID, if any.
func (m *Manager) Get(participantID uint) (*Lock, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	existing, ok := m.locks[participantID]
	if !ok || !time.Now().Before(existing.ExpiresAt) {
		return nil, false
	}
	return existing.clone(), true
}

// LockedByOther returns the active lock and true when participantID is locked by
// an admin other than userID. It is the enforcement primitive used by the
// lock-guard middleware.
func (m *Manager) LockedByOther(participantID, userID uint) (*Lock, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	existing, ok := m.locks[participantID]
	if !ok || !time.Now().Before(existing.ExpiresAt) {
		return nil, false
	}
	if existing.OwnerUserID == userID {
		return nil, false
	}
	return existing.clone(), true
}

// cleanupLoop periodically removes expired locks.
func (m *Manager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// cleanup removes expired locks from the map.
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0
	for pid, l := range m.locks {
		if !now.Before(l.ExpiresAt) {
			delete(m.locks, pid)
			removed++
		}
	}
	if removed > 0 {
		log.Printf("DEBUG Participant lock cleanup: removed=%d", removed)
	}
}
