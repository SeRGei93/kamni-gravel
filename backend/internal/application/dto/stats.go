package dto

import (
	"gravel_bot/internal/application/query"
)

// StatsDTO представляет DTO статистики для API
type StatsDTO struct {
	EventID             uint           `json:"event_id"`
	EventName           string         `json:"event_name"`
	ParticipantsCount   int            `json:"participants_count"`
	FinishedCount       int            `json:"finished_count"`
	GiftsCount          int            `json:"gifts_count"`
	PrizesAssignedCount int            `json:"prizes_assigned_count"`
	ByGender            map[string]int `json:"by_gender"`
	ByBikeType          map[string]int `json:"by_bike_type"`
}

// FromEventStats создаёт DTO из query.EventStats
func FromEventStats(stats *query.EventStats) *StatsDTO {
	return &StatsDTO{
		EventID:             stats.EventID,
		EventName:           stats.EventName,
		ParticipantsCount:   stats.ParticipantsCount,
		FinishedCount:       stats.FinishedCount,
		GiftsCount:          stats.GiftsCount,
		PrizesAssignedCount: stats.PrizesAssignedCount,
		ByGender:            stats.ByGender,
		ByBikeType:          stats.ByBikeType,
	}
}

// StatsListResponse представляет ответ со статистикой
type StatsListResponse struct {
	Stats []*StatsDTO `json:"stats"`
	Total int         `json:"total"`
}
