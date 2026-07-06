package entity

import (
	"strings"
	"testing"
	"time"

	"gravel_bot/internal/domain/valueobject"
)

func TestEventHasStartedAtUsesMinskTime(t *testing.T) {
	start := time.Date(2026, 5, 27, 10, 0, 0, 0, valueobject.MinskLocation())
	event := &Event{StartDate: &start}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "before start",
			now:  time.Date(2026, 5, 27, 9, 59, 59, 0, valueobject.MinskLocation()),
			want: false,
		},
		{
			name: "at start",
			now:  time.Date(2026, 5, 27, 10, 0, 0, 0, valueobject.MinskLocation()),
			want: true,
		},
		{
			name: "after start in another timezone",
			now:  time.Date(2026, 5, 27, 7, 1, 0, 0, time.UTC),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := event.HasStartedAt(tt.now); got != tt.want {
				t.Fatalf("HasStartedAt() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestNormalizeEventTelegramTextsFillsGiftDraftTexts(t *testing.T) {
	texts := NormalizeEventTelegramTexts(EventTelegramTexts{})

	if texts.GiftDraft == "" {
		t.Fatal("GiftDraft must be filled")
	}
	if texts.GiftDraftActionDescription == "" {
		t.Fatal("GiftDraftActionDescription must be filled")
	}
	if texts.GiftConfirmationPrompt == "" {
		t.Fatal("GiftConfirmationPrompt must be filled")
	}
	if texts.GiftCallbackContinue == "" {
		t.Fatal("GiftCallbackContinue must be filled")
	}
}

func TestNormalizeEventParticipationConditionsFillsDefault(t *testing.T) {
	conditions := NormalizeEventParticipationConditions("")
	if !strings.Contains(conditions, "УСЛОВИЯ УЧАСТИЯ") {
		t.Fatalf("default conditions mismatch: got %q", conditions)
	}
}

func TestEventHasStartedAtWithoutStartDate(t *testing.T) {
	event := &Event{}
	if event.HasStartedAt(time.Now()) {
		t.Fatal("HasStartedAt() = true, want false")
	}
}

func TestEventHasEndedAt(t *testing.T) {
	end := time.Date(2026, 7, 7, 0, 0, 0, 0, valueobject.MinskLocation())
	event := &Event{EndDate: &end}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "before end",
			now:  time.Date(2026, 7, 6, 23, 59, 59, 0, valueobject.MinskLocation()),
			want: false,
		},
		{
			name: "at end",
			now:  time.Date(2026, 7, 7, 0, 0, 0, 0, valueobject.MinskLocation()),
			want: false,
		},
		{
			name: "after end in another timezone",
			now:  time.Date(2026, 7, 6, 21, 0, 1, 0, time.UTC),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := event.HasEndedAt(tt.now); got != tt.want {
				t.Fatalf("HasEndedAt() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEventHasEndedAtWithoutEndDate(t *testing.T) {
	event := &Event{}
	if event.HasEndedAt(time.Now()) {
		t.Fatal("HasEndedAt() = true, want false")
	}
}

func TestEventIntakeClosedAt(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, valueobject.MinskLocation())
	pastEnd := now.Add(-time.Hour)

	tests := []struct {
		name              string
		event             *Event
		wantGiftsClosed   bool
		wantResultsClosed bool
	}{
		{
			name:  "open event",
			event: &Event{},
		},
		{
			name:            "gifts stopped manually",
			event:           &Event{StopGifts: true},
			wantGiftsClosed: true,
		},
		{
			name:              "results stopped manually",
			event:             &Event{StopResults: true},
			wantResultsClosed: true,
		},
		{
			name:              "event ended closes both",
			event:             &Event{EndDate: &pastEnd},
			wantGiftsClosed:   true,
			wantResultsClosed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.GiftIntakeClosedAt(now); got != tt.wantGiftsClosed {
				t.Fatalf("GiftIntakeClosedAt() = %t, want %t", got, tt.wantGiftsClosed)
			}
			if got := tt.event.ResultIntakeClosedAt(now); got != tt.wantResultsClosed {
				t.Fatalf("ResultIntakeClosedAt() = %t, want %t", got, tt.wantResultsClosed)
			}
		})
	}
}

func TestEventSubmissionStartTimeInMinsk(t *testing.T) {
	start := time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC)
	event := &Event{StartDate: &start}

	got, ok := event.SubmissionStartTimeInMinsk()
	if !ok {
		t.Fatal("SubmissionStartTimeInMinsk() ok = false, want true")
	}

	want := time.Date(2026, 5, 27, 10, 0, 0, 0, valueobject.MinskLocation())
	if !got.Equal(want) {
		t.Fatalf("SubmissionStartTimeInMinsk() = %s, want %s", got, want)
	}
	if got.Location().String() != valueobject.MinskLocation().String() {
		t.Fatalf("SubmissionStartTimeInMinsk() location = %s, want %s", got.Location(), valueobject.MinskLocation())
	}
}
