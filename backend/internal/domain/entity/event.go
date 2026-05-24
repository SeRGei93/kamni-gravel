package entity

import "time"

// Event представляет велогонку/мероприятие
type Event struct {
	ID          uint
	Name        string
	Description string
	Active      bool
	StartDate   *time.Time
	EndDate     *time.Time
	GPXFilePath string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsActive проверяет, активно ли событие
func (e *Event) IsActive() bool {
	return e.Active
}

// IsOngoing проверяет, идёт ли событие сейчас
func (e *Event) IsOngoing() bool {
	now := time.Now()
	if e.StartDate != nil && now.Before(*e.StartDate) {
		return false
	}
	if e.EndDate != nil && now.After(*e.EndDate) {
		return false
	}
	return e.Active
}
