package handler

import (
	"testing"
	"time"

	"gravel_bot/internal/domain/valueobject"
)

func TestParseDateUsesMinskForDateOnly(t *testing.T) {
	got, err := parseDate("2026-05-27")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}

	want := time.Date(2026, 5, 27, 0, 0, 0, 0, valueobject.MinskLocation())
	assertSameMinskInstant(t, got, want)
}

func TestParseDateUsesMinskForTimezoneLessDateTime(t *testing.T) {
	got, err := parseDate("2026-05-27T09:30:00")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}

	want := time.Date(2026, 5, 27, 9, 30, 0, 0, valueobject.MinskLocation())
	assertSameMinskInstant(t, got, want)
}

func TestParseDateNormalizesRFC3339ToMinsk(t *testing.T) {
	got, err := parseDate("2026-05-27T08:30:00+02:00")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}

	want := time.Date(2026, 5, 27, 9, 30, 0, 0, valueobject.MinskLocation())
	assertSameMinskInstant(t, got, want)
}

func assertSameMinskInstant(t *testing.T, got time.Time, want time.Time) {
	t.Helper()

	if !got.Equal(want) {
		t.Fatalf("time = %s, want %s", got, want)
	}
	if got.Location().String() != valueobject.MinskLocation().String() {
		t.Fatalf("location = %s, want %s", got.Location(), valueobject.MinskLocation())
	}
}
