package dto

import (
	"time"

	"gravel_bot/internal/application/query"
)

// DailyCountDTO представляет количество за один день для API.
type DailyCountDTO struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int    `json:"count"`
}

// DailyStatsDTO представляет посуточную статистику события для API.
type DailyStatsDTO struct {
	EventID       uint            `json:"event_id"`
	EventName     string          `json:"event_name"`
	StartDate     *string         `json:"start_date"`
	Registrations []DailyCountDTO `json:"registrations"`
	Finishes      []DailyCountDTO `json:"finishes"`
}

// FromEventDailyStats создаёт DTO из query.EventDailyStats.
func FromEventDailyStats(stats *query.EventDailyStats) *DailyStatsDTO {
	var startDate *string
	if stats.StartDate != nil {
		s := stats.StartDate.Format(time.RFC3339)
		startDate = &s
	}

	return &DailyStatsDTO{
		EventID:       stats.EventID,
		EventName:     stats.EventName,
		StartDate:     startDate,
		Registrations: fromDailyCounts(stats.Registrations),
		Finishes:      fromDailyCounts(stats.Finishes),
	}
}

// fromDailyCounts конвертирует срез доменных точек в DTO. Всегда возвращает
// непустой срез (а не nil), чтобы JSON отдавал [] вместо null.
func fromDailyCounts(points []query.DailyCount) []DailyCountDTO {
	out := make([]DailyCountDTO, 0, len(points))
	for _, p := range points {
		out = append(out, DailyCountDTO{Date: p.Date, Count: p.Count})
	}
	return out
}
