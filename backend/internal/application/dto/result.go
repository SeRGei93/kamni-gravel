package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
)

// ResultDTO представляет DTO результата для API
type ResultDTO struct {
	ID             uint           `json:"id"`
	ParticipantID  uint           `json:"participant_id"`
	ResultLink     *string        `json:"result_link,omitempty"`
	ElapsedTimeSec *int           `json:"elapsed_time_sec,omitempty"`
	MovingTimeSec  *int           `json:"moving_time_sec,omitempty"`
	ElapsedTime    *string        `json:"elapsed_time,omitempty"` // формат ЧЧ:ММ:СС
	MovingTime     *string        `json:"moving_time,omitempty"`  // формат ЧЧ:ММ:СС
	IsCurrent      bool           `json:"is_current"`
	SubmittedAt    time.Time      `json:"submitted_at"`
	Criteria       []*CriteriaDTO `json:"criteria,omitempty"` // критерии результата

	// Метрики заезда (вводятся вручную; опциональны)
	StartedAt      *time.Time `json:"started_at,omitempty"`      // Время старта
	FinishedAt     *time.Time `json:"finished_at,omitempty"`     // Время финиша
	DistanceMeters *int       `json:"distance_meters,omitempty"` // Дистанция в метрах
	AvgHeartRate   *int       `json:"avg_heart_rate,omitempty"`  // Средний пульс
	MaxHeartRate   *int       `json:"max_heart_rate,omitempty"`  // Максимальный пульс
	PeakSpeedKmh   *float64   `json:"peak_speed_kmh,omitempty"`  // Пиковая скорость, км/ч
	AvgCadence     *int       `json:"avg_cadence,omitempty"`     // Средний каденс
	Calories       *int       `json:"calories,omitempty"`        // Калории

	// Вычисляемые поля (только для чтения; считаются на сервере)
	RideDate          *string  `json:"ride_date,omitempty"`            // Дата проезда, YYYY-MM-DD
	IdleTimeSec       *int     `json:"idle_time_sec,omitempty"`        // Простой в секундах
	IdleTime          *string  `json:"idle_time,omitempty"`            // Простой, ЧЧ:ММ:СС
	AvgSpeedKmh       *float64 `json:"avg_speed_kmh,omitempty"`        // Средняя скорость, км/ч
	AvgMovingSpeedKmh *float64 `json:"avg_moving_speed_kmh,omitempty"` // Средняя скорость в движении, км/ч
}

// FromResult создаёт DTO из entity.Result
func FromResult(r *entity.Result) *ResultDTO {
	if r == nil {
		return nil
	}

	dto := &ResultDTO{
		ID:             r.ID,
		ParticipantID:  r.ParticipantID,
		ElapsedTimeSec: r.ElapsedTimeSec,
		MovingTimeSec:  r.MovingTimeSec,
		IsCurrent:      r.IsCurrent,
		SubmittedAt:    r.SubmittedAt,
	}

	if r.ResultLink != nil {
		link := r.ResultLink.URL
		dto.ResultLink = &link
	}

	if r.ElapsedTimeSec != nil {
		formatted := r.ElapsedTimeFormatted()
		dto.ElapsedTime = &formatted
	}

	if r.MovingTimeSec != nil {
		formatted := r.MovingTimeFormatted()
		dto.MovingTime = &formatted
	}

	// Сырые метрики заезда
	dto.StartedAt = r.StartedAt
	dto.FinishedAt = r.FinishedAt
	dto.DistanceMeters = r.DistanceMeters
	dto.AvgHeartRate = r.AvgHeartRate
	dto.MaxHeartRate = r.MaxHeartRate
	dto.PeakSpeedKmh = r.PeakSpeedKmh
	dto.AvgCadence = r.AvgCadence
	dto.Calories = r.Calories

	// Вычисляемые поля — только когда есть исходные данные
	if rideDate := r.RideDate(); rideDate != nil {
		formatted := rideDate.Format("2006-01-02")
		dto.RideDate = &formatted
	}
	if idle := r.IdleTimeSec(); idle != nil {
		dto.IdleTimeSec = idle
		formatted := r.IdleTimeFormatted()
		dto.IdleTime = &formatted
	}
	dto.AvgSpeedKmh = r.AvgSpeedKmh()
	dto.AvgMovingSpeedKmh = r.AvgMovingSpeedKmh()

	// Добавляем критерии
	if len(r.Criteria) > 0 {
		dto.Criteria = make([]*CriteriaDTO, len(r.Criteria))
		for i, c := range r.Criteria {
			dto.Criteria[i] = FromCriteria(c)
		}
	}

	return dto
}

// ResultListResponse представляет ответ со списком результатов
type ResultListResponse struct {
	Results []*ResultDTO `json:"results"`
	Total   int          `json:"total"`
}
