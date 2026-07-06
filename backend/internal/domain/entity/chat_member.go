package entity

import "time"

// ChatMember представляет текущего участника публичного чата.
// Строка существует, только пока участник в чате.
type ChatMember struct {
	TelegramUserID int64
	Username       string
	FirstName      string
	LastName       string
	IsBot          bool
	IsAdmin        bool
	JoinedAt       time.Time
	UpdatedAt      time.Time
}

// FullName возвращает отображаемое имя участника чата.
func (m *ChatMember) FullName() string {
	if m.FirstName == "" && m.LastName == "" {
		return m.Username
	}
	if m.LastName == "" {
		return m.FirstName
	}
	return m.FirstName + " " + m.LastName
}
