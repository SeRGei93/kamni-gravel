package lock

import (
	"testing"
	"time"
)

// expire forces the lock for participantID to look expired, so expiry paths can
// be tested deterministically without sleeping. Same-package access only.
func expire(m *Manager, participantID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.locks[participantID]; ok {
		l.ExpiresAt = time.Now().Add(-time.Minute)
	}
}

func TestAcquire_WhenFree(t *testing.T) {
	m := NewManager(DefaultTTL)

	l, ok := m.Acquire(1, 10, "alice")
	if !ok {
		t.Fatalf("expected acquire to succeed on a free participant")
	}
	if l.OwnerUserID != 10 || l.OwnerUsername != "alice" {
		t.Fatalf("unexpected owner: got user_id=%d username=%q", l.OwnerUserID, l.OwnerUsername)
	}
	if !l.ExpiresAt.After(time.Now()) {
		t.Fatalf("expected a future expiry, got %s", l.ExpiresAt)
	}
}

func TestAcquire_IdempotentForOwner(t *testing.T) {
	m := NewManager(DefaultTTL)

	first, _ := m.Acquire(1, 10, "alice")
	second, ok := m.Acquire(1, 10, "alice")
	if !ok {
		t.Fatalf("expected owner re-acquire (heartbeat) to succeed")
	}
	if !second.AcquiredAt.Equal(first.AcquiredAt) {
		t.Fatalf("expected AcquiredAt preserved on refresh: first=%s second=%s", first.AcquiredAt, second.AcquiredAt)
	}
	if second.ExpiresAt.Before(first.ExpiresAt) {
		t.Fatalf("expected ExpiresAt to be extended on refresh")
	}
}

func TestAcquire_BlockedByOther(t *testing.T) {
	m := NewManager(DefaultTTL)

	m.Acquire(1, 10, "alice")
	l, ok := m.Acquire(1, 20, "bob")
	if ok {
		t.Fatalf("expected acquire to be blocked while held by another admin")
	}
	if l == nil || l.OwnerUsername != "alice" {
		t.Fatalf("expected the existing owner (alice) to be reported, got %+v", l)
	}
}

func TestAcquire_StealsExpiredLock(t *testing.T) {
	m := NewManager(DefaultTTL)

	m.Acquire(1, 10, "alice")
	expire(m, 1)

	l, ok := m.Acquire(1, 20, "bob")
	if !ok {
		t.Fatalf("expected acquire to succeed after the previous lock expired")
	}
	if l.OwnerUserID != 20 || l.OwnerUsername != "bob" {
		t.Fatalf("expected bob to own the stolen lock, got %+v", l)
	}
}

func TestRefresh_OwnerAndNonOwner(t *testing.T) {
	m := NewManager(DefaultTTL)

	if _, ok := m.Refresh(1, 10); ok {
		t.Fatalf("expected refresh to fail when no lock exists")
	}

	first, _ := m.Acquire(1, 10, "alice")
	refreshed, ok := m.Refresh(1, 10)
	if !ok {
		t.Fatalf("expected owner refresh to succeed")
	}
	if refreshed.ExpiresAt.Before(first.ExpiresAt) {
		t.Fatalf("expected refresh to extend expiry")
	}

	if _, ok := m.Refresh(1, 20); ok {
		t.Fatalf("expected refresh by a non-owner to fail")
	}
}

func TestRelease_OwnerAndNonOwner(t *testing.T) {
	m := NewManager(DefaultTTL)

	m.Acquire(1, 10, "alice")

	if m.Release(1, 20) {
		t.Fatalf("expected release by a non-owner to fail")
	}
	if _, ok := m.Get(1); !ok {
		t.Fatalf("lock should still be held after a non-owner release attempt")
	}

	if !m.Release(1, 10) {
		t.Fatalf("expected owner release to succeed")
	}
	if _, ok := m.Get(1); ok {
		t.Fatalf("lock should be gone after owner release")
	}
}

func TestGetAndLockedByOther_ExpiredAndOwnership(t *testing.T) {
	m := NewManager(DefaultTTL)

	m.Acquire(1, 10, "alice")

	// Owner is not "locked by other".
	if _, ok := m.LockedByOther(1, 10); ok {
		t.Fatalf("owner must not be blocked by their own lock")
	}
	// A different admin is blocked.
	if _, ok := m.LockedByOther(1, 20); !ok {
		t.Fatalf("a different admin must be blocked while the lock is valid")
	}

	// Once expired, both Get and LockedByOther report no active lock.
	expire(m, 1)
	if _, ok := m.Get(1); ok {
		t.Fatalf("Get must return no active lock once expired")
	}
	if _, ok := m.LockedByOther(1, 20); ok {
		t.Fatalf("LockedByOther must return false once the lock has expired")
	}
}
