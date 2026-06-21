package cache

import (
	"os"
	"sync"
	"testing"
	"time"

	"gravel_bot/internal/application/dto"
)

func testGifts(ids ...uint) []*dto.GiftDTO {
	gifts := make([]*dto.GiftDTO, 0, len(ids))
	for _, id := range ids {
		gifts = append(gifts, &dto.GiftDTO{ID: id, Description: "gift"})
	}
	return gifts
}

func newTestCache(t *testing.T, ttl time.Duration) *MiniappGiftsCache {
	t.Helper()
	c, err := NewMiniappGiftsCache(t.TempDir(), ttl)
	if err != nil {
		t.Fatalf("NewMiniappGiftsCache: %v", err)
	}
	return c
}

func TestNewMiniappGiftsCacheRejectsEmptyDir(t *testing.T) {
	if _, err := NewMiniappGiftsCache("  ", time.Hour); err == nil {
		t.Fatal("expected error for empty cache dir")
	}
}

func TestMiniappGiftsCacheSetGetHit(t *testing.T) {
	c := newTestCache(t, time.Hour)
	c.Set(1, "all", "all", testGifts(10, 20))

	got, ok := c.Get(1, "all", "all")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != 2 || got[0].ID != 10 || got[1].ID != 20 {
		t.Fatalf("unexpected gifts: %+v", got)
	}
}

func TestMiniappGiftsCacheMiss(t *testing.T) {
	c := newTestCache(t, time.Hour)
	if _, ok := c.Get(1, "all", "all"); ok {
		t.Fatal("expected miss on empty cache")
	}
}

func TestMiniappGiftsCacheEmptyFilterEqualsAll(t *testing.T) {
	c := newTestCache(t, time.Hour)
	c.Set(1, "", "", testGifts(5))

	if _, ok := c.Get(1, "all", "all"); !ok {
		t.Fatal("empty filter and explicit \"all\" should map to the same entry")
	}
}

func TestMiniappGiftsCacheKeyIsolation(t *testing.T) {
	c := newTestCache(t, time.Hour)
	c.Set(1, "all", "all", testGifts(1))
	c.Set(1, "male", "all", testGifts(2))
	c.Set(2, "all", "all", testGifts(3))

	cases := []struct {
		eventID  uint
		gender   string
		bikeType string
		wantID   uint
	}{
		{1, "all", "all", 1},
		{1, "male", "all", 2},
		{2, "all", "all", 3},
	}
	for _, tc := range cases {
		got, ok := c.Get(tc.eventID, tc.gender, tc.bikeType)
		if !ok || len(got) != 1 || got[0].ID != tc.wantID {
			t.Fatalf("key (%d,%s,%s): got %+v ok=%v, want id=%d", tc.eventID, tc.gender, tc.bikeType, got, ok, tc.wantID)
		}
	}
}

func TestMiniappGiftsCacheInvalidateEventScoped(t *testing.T) {
	c := newTestCache(t, time.Hour)
	c.Set(1, "all", "all", testGifts(1))
	c.Set(1, "male", "gravel", testGifts(2))
	c.Set(2, "all", "all", testGifts(3))

	c.InvalidateEvent(1)

	if _, ok := c.Get(1, "all", "all"); ok {
		t.Fatal("event 1 (all,all) should be invalidated")
	}
	if _, ok := c.Get(1, "male", "gravel"); ok {
		t.Fatal("event 1 (male,gravel) should be invalidated")
	}
	if _, ok := c.Get(2, "all", "all"); !ok {
		t.Fatal("event 2 should remain cached")
	}
}

func TestMiniappGiftsCacheInvalidateEventNoPrefixCollision(t *testing.T) {
	c := newTestCache(t, time.Hour)
	c.Set(1, "all", "all", testGifts(1))
	c.Set(12, "all", "all", testGifts(12))

	c.InvalidateEvent(1)

	if _, ok := c.Get(12, "all", "all"); !ok {
		t.Fatal("event 12 must not be affected by invalidating event 1")
	}
}

func TestMiniappGiftsCacheTTLExpiry(t *testing.T) {
	c := newTestCache(t, 20*time.Millisecond)
	c.Set(1, "all", "all", testGifts(1))

	if _, ok := c.Get(1, "all", "all"); !ok {
		t.Fatal("expected hit before TTL expiry")
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.Get(1, "all", "all"); ok {
		t.Fatal("expected miss after TTL expiry")
	}
}

func TestMiniappGiftsCacheZeroTTLNeverExpires(t *testing.T) {
	c := newTestCache(t, 0)
	c.Set(1, "all", "all", testGifts(1))
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.Get(1, "all", "all"); !ok {
		t.Fatal("zero TTL should not expire entries")
	}
}

func TestMiniappGiftsCachePersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	c1, err := NewMiniappGiftsCache(dir, time.Hour)
	if err != nil {
		t.Fatalf("first cache: %v", err)
	}
	c1.Set(1, "all", "all", testGifts(7))

	// Новый инстанс над тем же каталогом моделирует перезапуск бэкенда.
	c2, err := NewMiniappGiftsCache(dir, time.Hour)
	if err != nil {
		t.Fatalf("second cache: %v", err)
	}
	got, ok := c2.Get(1, "all", "all")
	if !ok || len(got) != 1 || got[0].ID != 7 {
		t.Fatalf("expected persisted entry, got %+v ok=%v", got, ok)
	}
}

func TestMiniappGiftsCacheCorruptFileIsMiss(t *testing.T) {
	c := newTestCache(t, time.Hour)
	path := c.filePath(1, "all", "all")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	if _, ok := c.Get(1, "all", "all"); ok {
		t.Fatal("corrupt file should be a miss")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt file should have been removed, stat err=%v", err)
	}
}

func TestMiniappGiftsCacheConcurrentAccess(t *testing.T) {
	c := newTestCache(t, time.Hour)

	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			eventID := uint(n%3 + 1)
			c.Set(eventID, "all", "all", testGifts(uint(n)))
			c.Get(eventID, "all", "all")
			c.InvalidateEvent(eventID)
		}(i)
	}
	wg.Wait()
}
