package query

import (
	"context"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// DailyCount представляет количество за один календарный день.
type DailyCount struct {
	Date  string // формат YYYY-MM-DD
	Count int
}

// EventDailyStats содержит посуточную статистику события для дашборда.
type EventDailyStats struct {
	EventID   uint
	EventName string
	StartDate *time.Time
	// Registrations — новые участники по дате регистрации (registered_at).
	Registrations []DailyCount
	// Finishes — проехавшие по дате отправки результата (results.submitted_at),
	// ось дней начинается со дня старта события (start_date).
	Finishes []DailyCount
}

// GetDailyStatsQuery представляет запрос посуточной статистики события.
type GetDailyStatsQuery struct {
	EventID uint
}

// GetDailyStatsHandler обрабатывает запрос посуточной статистики.
// Считает обе серии в памяти из одного FindByEvent (результат уже подгружен
// через LEFT JOIN), повторяя подход GetStatsHandler.
type GetDailyStatsHandler struct {
	eventRepo       repository.EventRepository
	participantRepo repository.ParticipantRepository
}

// NewGetDailyStatsHandler создаёт новый handler.
func NewGetDailyStatsHandler(
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
) *GetDailyStatsHandler {
	return &GetDailyStatsHandler{
		eventRepo:       eventRepo,
		participantRepo: participantRepo,
	}
}

// Handle выполняет запрос. Событие резолвится только по ID (без FindActive —
// маршрут всегда передаёт eventId).
func (h *GetDailyStatsHandler) Handle(ctx context.Context, query GetDailyStatsQuery) (*EventDailyStats, error) {
	event, err := h.eventRepo.FindByID(ctx, query.EventID)
	if err != nil {
		log.Printf("ERROR Daily stats failed: event_id=%d stage=find_event error=%v", query.EventID, err)
		return nil, fmt.Errorf("failed to find event: %w", err)
	}
	if event == nil {
		log.Printf("WARN Daily stats rejected: event_id=%d reason=event_not_found", query.EventID)
		return nil, fmt.Errorf("event not found")
	}

	participants, err := h.participantRepo.FindByEvent(ctx, event.ID)
	if err != nil {
		log.Printf("ERROR Daily stats failed: event_id=%d stage=find_participants error=%v", event.ID, err)
		return nil, fmt.Errorf("failed to find participants: %w", err)
	}

	// Дни считаем в зоне Минска (UTC+3) — так же, как доменная логика старта
	// события (Event.HasStartedAt / SubmissionStartTimeInMinsk). Это фиксированная
	// зона, а не Local(), поэтому бакеты не зависят от TZ контейнера.
	loc := valueobject.MinskLocation()
	today := dailyStartOfDay(time.Now(), loc)

	// Регистрации: бакеты по registered_at, ось от первой регистрации до сегодня.
	regCounts := make(map[string]int)
	var regFirst *time.Time
	for _, p := range participants {
		day := dailyStartOfDay(p.RegisteredAt, loc)
		regCounts[dailyKey(day)]++
		regFirst = earliestDay(regFirst, day)
	}

	// Финиши: бакеты по submitted_at текущего результата, ось от старта события.
	finCounts := make(map[string]int)
	var finFirst *time.Time
	for _, p := range participants {
		if p.Result == nil {
			continue
		}
		day := dailyStartOfDay(p.Result.SubmittedAt, loc)
		finCounts[dailyKey(day)]++
		finFirst = earliestDay(finFirst, day)
	}

	registrations := buildDailySeries(regFirst, today, regCounts, loc)

	// Ось финишей начинается со дня старта события; если старт не задан —
	// откатываемся на день первого финиша (или пустой ряд, если финишей нет).
	finStart := finFirst
	if event.StartDate != nil {
		d := dailyStartOfDay(*event.StartDate, loc)
		finStart = &d
	}
	finishes := buildDailySeries(finStart, today, finCounts, loc)

	stats := &EventDailyStats{
		EventID:       event.ID,
		EventName:     event.Name,
		StartDate:     event.StartDate,
		Registrations: registrations,
		Finishes:      finishes,
	}

	log.Printf("INFO Daily stats computed: event_id=%d event_name=%q start_date=%s participants=%d registrations_range=%s..%s registrations_total=%d finishes_range=%s..%s finishes_total=%d",
		event.ID, event.Name, formatDailyStart(event.StartDate), len(participants),
		seriesStart(registrations), seriesEnd(registrations), dailySeriesTotal(registrations),
		seriesStart(finishes), seriesEnd(finishes), dailySeriesTotal(finishes))

	return stats, nil
}

// dailyStartOfDay приводит момент времени к началу календарного дня в зоне loc.
func dailyStartOfDay(t time.Time, loc *time.Location) time.Time {
	lt := t.In(loc)
	return time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, loc)
}

// dailyKey форматирует день как YYYY-MM-DD.
func dailyKey(day time.Time) string {
	return day.Format("2006-01-02")
}

// earliestDay возвращает более раннюю из двух дат (current может быть nil).
func earliestDay(current *time.Time, candidate time.Time) *time.Time {
	if current == nil || candidate.Before(*current) {
		c := candidate
		return &c
	}
	return current
}

// buildDailySeries строит непрерывный ряд по дням [start..end] включительно,
// заполняя отсутствующие дни нулями. Возвращает пустой ряд, если start == nil
// или диапазон вырожден (start после end, например старт события в будущем).
func buildDailySeries(start *time.Time, end time.Time, counts map[string]int, loc *time.Location) []DailyCount {
	series := make([]DailyCount, 0)
	if start == nil {
		return series
	}
	s := dailyStartOfDay(*start, loc)
	e := dailyStartOfDay(end, loc)
	if s.After(e) {
		return series
	}
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		key := dailyKey(d)
		series = append(series, DailyCount{Date: key, Count: counts[key]})
	}
	return series
}

// dailySeriesTotal суммирует количество по всему ряду.
func dailySeriesTotal(series []DailyCount) int {
	sum := 0
	for _, dc := range series {
		sum += dc.Count
	}
	return sum
}

// seriesStart возвращает первую дату ряда (или "-" для пустого ряда) — для логов.
func seriesStart(series []DailyCount) string {
	if len(series) == 0 {
		return "-"
	}
	return series[0].Date
}

// seriesEnd возвращает последнюю дату ряда (или "-" для пустого ряда) — для логов.
func seriesEnd(series []DailyCount) string {
	if len(series) == 0 {
		return "-"
	}
	return series[len(series)-1].Date
}

// formatDailyStart форматирует дату старта для логов (или "nil").
func formatDailyStart(t *time.Time) string {
	if t == nil {
		return "nil"
	}
	return t.Format(time.RFC3339)
}
