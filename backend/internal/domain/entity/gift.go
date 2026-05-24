package entity

import "time"

// Gift представляет подарок от участника
type Gift struct {
	ID             uint
	UserID         int64
	EventID        uint
	Description    string
	GenderFilter   string // all, male, female
	BikeTypeFilter string // all, gravel, mtb, road, single_speed, tandem
	Place          *int   // место (позиция), nil если не задано
	CreatedAt      time.Time

	// Связанные сущности
	User        *User
	Attachments []GiftAttachment
	Criteria    []*Criteria // Критерии, привязанные к подарку
}

// HasAttachments проверяет, есть ли у подарка прикреплённые файлы
func (g *Gift) HasAttachments() bool {
	return len(g.Attachments) > 0
}

// HasCriteria проверяет, есть ли у подарка привязанные критерии
func (g *Gift) HasCriteria() bool {
	return len(g.Criteria) > 0
}

// GiftAttachment представляет прикреплённый файл к подарку
type GiftAttachment struct {
	ID             uint
	GiftID         uint
	TelegramFileID string
	FileType       string // photo, document
}
