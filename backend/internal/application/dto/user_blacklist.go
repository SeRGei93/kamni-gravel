package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
)

// UserBlacklistDTO представляет запись blacklist для API.
type UserBlacklistDTO struct {
	TelegramUserID int64     `json:"telegram_user_id"`
	Reason         string    `json:"reason"`
	Username       string    `json:"username,omitempty"`
	FirstName      string    `json:"first_name,omitempty"`
	LastName       string    `json:"last_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// FromUserBlacklist создаёт DTO из entity.UserBlacklist.
func FromUserBlacklist(entry *entity.UserBlacklist) *UserBlacklistDTO {
	dto := &UserBlacklistDTO{
		TelegramUserID: entry.TelegramUserID,
		Reason:         entry.Reason,
		CreatedAt:      entry.CreatedAt,
		UpdatedAt:      entry.UpdatedAt,
	}

	if entry.User != nil {
		dto.Username = entry.User.Username
		dto.FirstName = entry.User.FirstName
		dto.LastName = entry.User.LastName
	}

	return dto
}

// UserBlacklistListResponse представляет ответ со списком blacklist.
type UserBlacklistListResponse struct {
	Entries []*UserBlacklistDTO `json:"entries"`
	Total   int                 `json:"total"`
}
