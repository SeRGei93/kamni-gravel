package entity

import (
	"fmt"
	"time"

	"gravel_bot/internal/domain/valueobject"
)

// Result представляет результат участника
type Result struct {
	ID             uint
	ParticipantID  uint
	ResultLink     *valueobject.ResultLink
	ElapsedTimeSec *int // Общее время в секундах
	MovingTimeSec  *int // Время в пути в секундах
	IsCurrent      bool // Актуальный результат (последний)
	SubmittedAt    time.Time

	// Связанные сущности
	Criteria []*Criteria // Критерии, привязанные к результату
}

// ElapsedTimeFormatted возвращает общее время в формате ЧЧ:ММ:СС
func (r *Result) ElapsedTimeFormatted() string {
	if r.ElapsedTimeSec == nil {
		return ""
	}
	return formatSeconds(*r.ElapsedTimeSec)
}

// MovingTimeFormatted возвращает время в пути в формате ЧЧ:ММ:СС
func (r *Result) MovingTimeFormatted() string {
	if r.MovingTimeSec == nil {
		return ""
	}
	return formatSeconds(*r.MovingTimeSec)
}

func formatSeconds(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}
