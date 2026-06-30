package entity

import (
	"time"

	"gravel_bot/internal/domain/valueobject"
)

// Participant представляет участника события
type Participant struct {
	ID           uint
	UserID       int64
	EventID      uint
	BikeType     valueobject.BikeType
	Gender       valueobject.Gender
	Status       valueobject.ParticipantStatus // active / dnf / disqualified
	Notes        string
	RegisteredAt time.Time

	// Связанные сущности (для удобства, заполняются через JOIN)
	User   *User
	Result *Result // Текущий актуальный результат (is_current = true)
}

// IsDNF сообщает, что участник сошёл с дистанции.
func (p *Participant) IsDNF() bool {
	return p.Status == valueobject.ParticipantStatusDNF
}

// IsDisqualified сообщает, что участник дисквалифицирован.
func (p *Participant) IsDisqualified() bool {
	return p.Status == valueobject.ParticipantStatusDisqualified
}

// IsRanked сообщает, участвует ли участник в зачёте (по местам). Сошедшие и
// дисквалифицированные из зачёта исключаются; пустой статус считается active.
func (p *Participant) IsRanked() bool {
	return !p.IsDNF() && !p.IsDisqualified()
}

// HasResult проверяет, есть ли у участника результат
func (p *Participant) HasResult() bool {
	return p.Result != nil && p.Result.ResultLink != nil && p.Result.ResultLink.URL != ""
}

// IsFinished проверяет, финишировал ли участник (есть актуальный результат)
func (p *Participant) IsFinished() bool {
	return p.Result != nil
}

// ElapsedTimeFormatted возвращает общее время в формате ЧЧ:ММ:СС
func (p *Participant) ElapsedTimeFormatted() string {
	if p.Result == nil {
		return ""
	}
	return p.Result.ElapsedTimeFormatted()
}

// MovingTimeFormatted возвращает чистое время в формате ЧЧ:ММ:СС
func (p *Participant) MovingTimeFormatted() string {
	if p.Result == nil {
		return ""
	}
	return p.Result.MovingTimeFormatted()
}

// GetElapsedTimeSec возвращает время в секундах или nil
func (p *Participant) GetElapsedTimeSec() *int {
	if p.Result == nil {
		return nil
	}
	return p.Result.ElapsedTimeSec
}

// GetMovingTimeSec возвращает чистое время в секундах или nil
func (p *Participant) GetMovingTimeSec() *int {
	if p.Result == nil {
		return nil
	}
	return p.Result.MovingTimeSec
}

// GetResultLink возвращает ссылку на результат или nil
func (p *Participant) GetResultLink() string {
	if p.Result == nil || p.Result.ResultLink == nil {
		return ""
	}
	return p.Result.ResultLink.URL
}

// GetFinishedAt возвращает время финиша (submitted_at результата)
func (p *Participant) GetFinishedAt() *time.Time {
	if p.Result == nil {
		return nil
	}
	return &p.Result.SubmittedAt
}
