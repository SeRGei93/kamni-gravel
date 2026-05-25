package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
)

// EventDTO представляет DTO события для API
type EventDTO struct {
	ID            uint                      `json:"id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description"`
	Active        bool                      `json:"active"`
	StartDate     *time.Time                `json:"start_date,omitempty"`
	EndDate       *time.Time                `json:"end_date,omitempty"`
	GPXFilePath   string                    `json:"gpx_file_path,omitempty"`
	TelegramTexts entity.EventTelegramTexts `json:"telegram_texts"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

// FromEvent создаёт DTO из entity.Event
func FromEvent(e *entity.Event) *EventDTO {
	return &EventDTO{
		ID:            e.ID,
		Name:          e.Name,
		Description:   e.Description,
		Active:        e.Active,
		StartDate:     e.StartDate,
		EndDate:       e.EndDate,
		GPXFilePath:   e.GPXFilePath,
		TelegramTexts: entity.NormalizeEventTelegramTexts(e.TelegramTexts),
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

// EventListResponse представляет ответ со списком событий
type EventListResponse struct {
	Events []*EventDTO `json:"events"`
	Total  int         `json:"total"`
}
