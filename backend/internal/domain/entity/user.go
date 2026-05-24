package entity

import "time"

// User представляет пользователя Telegram
type User struct {
	ID        int64     // Telegram ID
	Username  string
	FirstName string
	LastName  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FullName возвращает полное имя пользователя
func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Username
	}
	return u.FirstName + " " + u.LastName
}
