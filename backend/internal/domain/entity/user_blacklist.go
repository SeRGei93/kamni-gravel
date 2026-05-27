package entity

import "time"

// UserBlacklist представляет заблокированный Telegram ID.
type UserBlacklist struct {
	TelegramUserID int64
	Reason         string
	CreatedAt      time.Time
	UpdatedAt      time.Time

	// User заполняется через LEFT JOIN, если пользователь уже существует.
	User *User
}
