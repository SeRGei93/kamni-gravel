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
	MovingTimeSec  *int // Чистое время в секундах
	IsCurrent      bool // Актуальный результат (последний)
	SubmittedAt    time.Time

	// Метрики заезда (из Стравы, вводятся вручную; все опциональны)
	StartedAt      *time.Time // Время старта
	FinishedAt     *time.Time // Время финиша
	DistanceMeters *int       // Дистанция в метрах
	AvgHeartRate   *int       // Средний пульс (уд/мин)
	MaxHeartRate   *int       // Максимальный пульс (уд/мин)
	PeakSpeedKmh   *float64   // Пиковая скорость (км/ч)
	AvgCadence     *int       // Средний каденс (об/мин)
	Calories       *int       // Калории (ккал)

	// Связанные сущности
	Criteria []*Criteria // Критерии, привязанные к результату
}

// ElapsedTimeFormatted возвращает общее время в формате ЧЧ:ММ:СС
func (r *Result) ElapsedTimeFormatted() string {
	if r.ElapsedTimeSec == nil {
		return ""
	}
	return FormatSeconds(*r.ElapsedTimeSec)
}

// MovingTimeFormatted возвращает чистое время в формате ЧЧ:ММ:СС
func (r *Result) MovingTimeFormatted() string {
	if r.MovingTimeSec == nil {
		return ""
	}
	return FormatSeconds(*r.MovingTimeSec)
}

// FormatSeconds форматирует длительность в секундах как ЧЧ:ММ:СС.
func FormatSeconds(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, secs)
}

// FormatSignedSeconds форматирует разницу длительностей как ±ЧЧ:ММ:СС
// (ноль — без знака).
func FormatSignedSeconds(seconds int) string {
	if seconds < 0 {
		return "-" + FormatSeconds(-seconds)
	}
	if seconds > 0 {
		return "+" + FormatSeconds(seconds)
	}
	return FormatSeconds(0)
}

// ComputedElapsedSec возвращает общее время как разницу финиша и старта,
// если оба значения заданы; иначе nil. Используется командой как источник
// общего времени, когда старт и финиш введены.
func (r *Result) ComputedElapsedSec() *int {
	if r.StartedAt == nil || r.FinishedAt == nil {
		return nil
	}
	sec := int(r.FinishedAt.Sub(*r.StartedAt).Seconds())
	return &sec
}

// IdleTimeSec возвращает время простоя (Простой) = общее − чистое,
// если оба значения заданы и результат неотрицателен; иначе nil.
func (r *Result) IdleTimeSec() *int {
	if r.ElapsedTimeSec == nil || r.MovingTimeSec == nil {
		return nil
	}
	idle := *r.ElapsedTimeSec - *r.MovingTimeSec
	if idle < 0 {
		return nil
	}
	return &idle
}

// IdleTimeFormatted возвращает простой в формате ЧЧ:ММ:СС или "".
func (r *Result) IdleTimeFormatted() string {
	idle := r.IdleTimeSec()
	if idle == nil {
		return ""
	}
	return FormatSeconds(*idle)
}

// AvgSpeedKmh возвращает среднюю скорость (км/ч) = дистанция / общее время,
// если заданы положительная дистанция и общее время; иначе nil.
func (r *Result) AvgSpeedKmh() *float64 {
	return speedKmh(r.DistanceMeters, r.ElapsedTimeSec)
}

// AvgMovingSpeedKmh возвращает среднюю скорость в движении (км/ч) =
// дистанция / чистое время, если заданы положительная дистанция и чистое
// время; иначе nil.
func (r *Result) AvgMovingSpeedKmh() *float64 {
	return speedKmh(r.DistanceMeters, r.MovingTimeSec)
}

// PeakAvgSpeedDeltaKmh возвращает разницу «пиковая − средняя скорость» (км/ч),
// если заданы пиковая скорость и вычислима средняя; иначе nil.
func (r *Result) PeakAvgSpeedDeltaKmh() *float64 {
	avg := r.AvgSpeedKmh()
	if r.PeakSpeedKmh == nil || avg == nil {
		return nil
	}
	delta := *r.PeakSpeedKmh - *avg
	return &delta
}

// speedKmh считает скорость в км/ч из дистанции (метры) и времени (секунды).
// Возвращает nil, если данных недостаточно (защита от деления на ноль).
func speedKmh(distanceMeters, timeSec *int) *float64 {
	if distanceMeters == nil || *distanceMeters <= 0 || timeSec == nil || *timeSec <= 0 {
		return nil
	}
	km := float64(*distanceMeters) / 1000.0
	hours := float64(*timeSec) / 3600.0
	speed := km / hours
	return &speed
}

// RideDate возвращает дату проезда (день старта) или nil, если старт не задан.
func (r *Result) RideDate() *time.Time {
	if r.StartedAt == nil {
		return nil
	}
	y, m, d := r.StartedAt.Date()
	date := time.Date(y, m, d, 0, 0, 0, 0, r.StartedAt.Location())
	return &date
}
