package query

import (
	"context"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// --- helpers ---------------------------------------------------------------

func dayKeyAt(t time.Time) string {
	return t.In(valueobject.MinskLocation()).Format("2006-01-02")
}

func countFor(series []DailyCount, key string) (int, bool) {
	for _, dc := range series {
		if dc.Date == key {
			return dc.Count, true
		}
	}
	return 0, false
}

func assertSortedAscending(t *testing.T, series []DailyCount) {
	t.Helper()
	for i := 1; i < len(series); i++ {
		if series[i-1].Date >= series[i].Date {
			t.Fatalf("series not sorted ascending at index %d: %q >= %q", i, series[i-1].Date, series[i].Date)
		}
	}
}

func participantWithResult(id uint, registeredAt time.Time, submittedAt *time.Time) *entity.Participant {
	p := &entity.Participant{ID: id, EventID: 1, RegisteredAt: registeredAt}
	if submittedAt != nil {
		p.Result = &entity.Result{ParticipantID: id, IsCurrent: true, SubmittedAt: *submittedAt}
	}
	return p
}

// --- tests -----------------------------------------------------------------

func TestGetDailyStatsBucketsAndGapFill(t *testing.T) {
	loc := valueobject.MinskLocation()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	start := today.AddDate(0, 0, -4) // событие стартовало 4 дня назад

	day := func(offset int) time.Time { return start.AddDate(0, 0, offset) }
	at := func(offset int) *time.Time { d := day(offset); return &d }

	participants := []*entity.Participant{
		participantWithResult(1, day(-2), at(0)),  // рег: start-2, финиш: start
		participantWithResult(2, day(-1), at(1)),  // рег: start-1, финиш: start+1
		participantWithResult(3, day(1), at(1)),   // рег: start+1, финиш: start+1
		participantWithResult(4, day(3), at(3)),   // рег: start+3, финиш: start+3
		participantWithResult(5, day(1), nil),     // рег: start+1, без результата
	}

	h := NewGetDailyStatsHandler(
		&dailyEventRepoFake{event: &entity.Event{ID: 1, Name: "Test", StartDate: &start}},
		&dailyParticipantRepoFake{participants: participants},
	)

	stats, err := h.Handle(context.Background(), GetDailyStatsQuery{EventID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Регистрации: ось от первой регистрации (start-2) до сегодня (start+4) = 7 дней.
	if len(stats.Registrations) != 7 {
		t.Fatalf("registrations length: got %d, want 7", len(stats.Registrations))
	}
	if got := dailySeriesTotal(stats.Registrations); got != 5 {
		t.Fatalf("registrations total: got %d, want 5", got)
	}
	if c, _ := countFor(stats.Registrations, dayKeyAt(day(-2))); c != 1 {
		t.Fatalf("registrations at start-2: got %d, want 1", c)
	}
	if c, _ := countFor(stats.Registrations, dayKeyAt(day(1))); c != 2 {
		t.Fatalf("registrations at start+1: got %d, want 2", c)
	}
	if c, _ := countFor(stats.Registrations, dayKeyAt(day(0))); c != 0 {
		t.Fatalf("registrations gap day at start: got %d, want 0 (gap-filled)", c)
	}
	assertSortedAscending(t, stats.Registrations)

	// Финиши: ось от дня старта события (start) до сегодня (start+4) = 5 дней.
	if len(stats.Finishes) != 5 {
		t.Fatalf("finishes length: got %d, want 5", len(stats.Finishes))
	}
	if stats.Finishes[0].Date != dayKeyAt(start) {
		t.Fatalf("finishes axis must start at event start_date: got %q, want %q", stats.Finishes[0].Date, dayKeyAt(start))
	}
	if got := dailySeriesTotal(stats.Finishes); got != 4 {
		t.Fatalf("finishes total: got %d, want 4 (participant without result excluded)", got)
	}
	if c, _ := countFor(stats.Finishes, dayKeyAt(day(0))); c != 1 {
		t.Fatalf("finishes at start: got %d, want 1", c)
	}
	if c, _ := countFor(stats.Finishes, dayKeyAt(day(1))); c != 2 {
		t.Fatalf("finishes at start+1: got %d, want 2", c)
	}
	if c, _ := countFor(stats.Finishes, dayKeyAt(day(3))); c != 1 {
		t.Fatalf("finishes at start+3: got %d, want 1", c)
	}
	assertSortedAscending(t, stats.Finishes)
}

func TestGetDailyStatsNilStartDateFallsBackToFirstFinish(t *testing.T) {
	loc := valueobject.MinskLocation()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	firstFinish := today.AddDate(0, 0, -2)

	participants := []*entity.Participant{
		participantWithResult(1, today.AddDate(0, 0, -3), &firstFinish),
	}

	h := NewGetDailyStatsHandler(
		&dailyEventRepoFake{event: &entity.Event{ID: 1, Name: "Test", StartDate: nil}},
		&dailyParticipantRepoFake{participants: participants},
	)

	stats, err := h.Handle(context.Background(), GetDailyStatsQuery{EventID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.Finishes) == 0 {
		t.Fatal("finishes must not be empty when start_date is nil but a finish exists")
	}
	if stats.Finishes[0].Date != dayKeyAt(firstFinish) {
		t.Fatalf("finishes axis must fall back to first finish: got %q, want %q", stats.Finishes[0].Date, dayKeyAt(firstFinish))
	}
}

func TestGetDailyStatsFutureStartDateYieldsEmptyFinishes(t *testing.T) {
	loc := valueobject.MinskLocation()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	futureStart := today.AddDate(0, 0, 2)
	regDay := today.AddDate(0, 0, -1)

	participants := []*entity.Participant{
		participantWithResult(1, regDay, nil),
	}

	h := NewGetDailyStatsHandler(
		&dailyEventRepoFake{event: &entity.Event{ID: 1, Name: "Test", StartDate: &futureStart}},
		&dailyParticipantRepoFake{participants: participants},
	)

	stats, err := h.Handle(context.Background(), GetDailyStatsQuery{EventID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.Finishes) != 0 {
		t.Fatalf("future start_date must yield empty finishes: got %d points", len(stats.Finishes))
	}
	if len(stats.Registrations) == 0 {
		t.Fatal("registrations must still be computed for a not-yet-started event")
	}
}

func TestGetDailyStatsNoParticipants(t *testing.T) {
	loc := valueobject.MinskLocation()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, loc)
	start := today.AddDate(0, 0, -2)

	h := NewGetDailyStatsHandler(
		&dailyEventRepoFake{event: &entity.Event{ID: 1, Name: "Test", StartDate: &start}},
		&dailyParticipantRepoFake{participants: nil},
	)

	stats, err := h.Handle(context.Background(), GetDailyStatsQuery{EventID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.Registrations) != 0 {
		t.Fatalf("registrations must be empty with no participants: got %d", len(stats.Registrations))
	}
	if got := dailySeriesTotal(stats.Finishes); got != 0 {
		t.Fatalf("finishes total must be 0 with no participants: got %d", got)
	}
}

func TestGetDailyStatsEventNotFound(t *testing.T) {
	h := NewGetDailyStatsHandler(
		&dailyEventRepoFake{event: nil},
		&dailyParticipantRepoFake{},
	)
	_, err := h.Handle(context.Background(), GetDailyStatsQuery{EventID: 999})
	if err == nil {
		t.Fatal("expected error when event not found")
	}
	if err.Error() != "event not found" {
		t.Fatalf("unexpected error: got %q, want %q", err.Error(), "event not found")
	}
}

// --- fakes (implement the full repository interfaces) ----------------------

type dailyEventRepoFake struct {
	event *entity.Event
}

func (r *dailyEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *dailyEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *dailyEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *dailyEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *dailyEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *dailyEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) { return nil, nil }
func (r *dailyEventRepoFake) Delete(ctx context.Context, id uint) error           { return nil }

type dailyParticipantRepoFake struct {
	participants []*entity.Participant
}

func (r *dailyParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *dailyParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *dailyParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *dailyParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *dailyParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return r.participants, nil
}
func (r *dailyParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *dailyParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *dailyParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *dailyParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}

// compile-time interface checks
var (
	_ repository.EventRepository       = (*dailyEventRepoFake)(nil)
	_ repository.ParticipantRepository = (*dailyParticipantRepoFake)(nil)
)
