package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

// ParticipantDTO представляет DTO участника для API
type ParticipantDTO struct {
	ID                     uint                      `json:"id"`
	UserID                 int64                     `json:"user_id"`
	Username               string                    `json:"username"`
	FirstName              string                    `json:"first_name"`
	LastName               string                    `json:"last_name"`
	EventID                uint                      `json:"event_id"`
	BikeType               string                    `json:"bike_type"`
	Gender                 string                    `json:"gender"`
	Status                 string                    `json:"status"` // active / dnf / disqualified
	ResultLink             *string                   `json:"result_link,omitempty"`
	IsFinished             bool                      `json:"is_finished"`
	ElapsedTime            *string                   `json:"elapsed_time,omitempty"` // формат ЧЧ:ММ:СС
	MovingTime             *string                   `json:"moving_time,omitempty"`  // формат ЧЧ:ММ:СС
	ElapsedTimeSec         *int                      `json:"elapsed_time_sec,omitempty"`
	MovingTimeSec          *int                      `json:"moving_time_sec,omitempty"`
	Notes                  string                    `json:"notes,omitempty"`
	RegisteredAt           time.Time                 `json:"registered_at"`
	FinishedAt             *time.Time                `json:"finished_at,omitempty"`
	Place                  int                       `json:"place,omitempty"`                    // место в зачёте (0 если нет) - устаревшее, используйте place_absolute
	PlaceAbsolute          *int                      `json:"place_absolute,omitempty"`           // место в абсолютном зачёте
	PlaceByGender          *int                      `json:"place_by_gender,omitempty"`          // место в зачёте по гендеру
	PlaceByGenderBike      *int                      `json:"place_by_gender_bike,omitempty"`     // место в зачёте по гендеру+тип велосипеда
	HasGift                bool                      `json:"has_gift"`                           // добавил ли подарок
	PrizesCount            int                       `json:"prizes_count"`                       // количество полученных призов
	MatchedGifts           []*GiftDTO                `json:"matched_gifts,omitempty"`            // все подобранные подарки
	MatchedGiftAssignments []*PrizeGiftAssignmentDTO `json:"matched_gift_assignments,omitempty"` // назначения слотов подарков

	// Метрики заезда из текущего результата (опциональны; см. dto.ResultDTO).
	// ВНИМАНИЕ: FinishedAt выше — это дата отправки результата (submitted_at).
	// Время финиша самого заезда отдаётся как ride_finished_at, чтобы не
	// конфликтовать с уже существующим json-ключом finished_at.
	StartedAt      *time.Time `json:"started_at,omitempty"`       // Время старта заезда
	RideFinishedAt *time.Time `json:"ride_finished_at,omitempty"` // Время финиша заезда
	DistanceMeters *int       `json:"distance_meters,omitempty"`  // Дистанция в метрах
	AvgHeartRate   *int       `json:"avg_heart_rate,omitempty"`   // Средний пульс
	MaxHeartRate   *int       `json:"max_heart_rate,omitempty"`   // Максимальный пульс
	PeakSpeedKmh   *float64   `json:"peak_speed_kmh,omitempty"`   // Пиковая скорость, км/ч
	AvgCadence     *int       `json:"avg_cadence,omitempty"`      // Средний каденс
	Calories       *int       `json:"calories,omitempty"`         // Калории

	// Вычисляемые поля заезда (только для чтения; считаются на сервере)
	RideDate          *string  `json:"ride_date,omitempty"`            // Дата проезда, YYYY-MM-DD
	IdleTimeSec       *int     `json:"idle_time_sec,omitempty"`        // Простой в секундах
	IdleTime          *string  `json:"idle_time,omitempty"`            // Простой, ЧЧ:ММ:СС
	AvgSpeedKmh       *float64 `json:"avg_speed_kmh,omitempty"`        // Средняя скорость, км/ч
	AvgMovingSpeedKmh *float64 `json:"avg_moving_speed_kmh,omitempty"` // Средняя скорость в движении, км/ч
}

// FromParticipant создаёт DTO из entity.Participant
func FromParticipant(p *entity.Participant) *ParticipantDTO {
	status := p.Status
	if status == "" {
		status = valueobject.ParticipantStatusActive
	}

	dto := &ParticipantDTO{
		ID:             p.ID,
		UserID:         p.UserID,
		EventID:        p.EventID,
		BikeType:       string(p.BikeType),
		Gender:         string(p.Gender),
		Status:         string(status),
		IsFinished:     p.IsFinished(),
		ElapsedTimeSec: p.GetElapsedTimeSec(),
		MovingTimeSec:  p.GetMovingTimeSec(),
		Notes:          p.Notes,
		RegisteredAt:   p.RegisteredAt,
		FinishedAt:     p.GetFinishedAt(),
	}

	// Добавляем данные пользователя, если есть
	if p.User != nil {
		dto.Username = p.User.Username
		dto.FirstName = p.User.FirstName
		dto.LastName = p.User.LastName
	}

	// Добавляем ссылку на результат
	if p.Result != nil && p.Result.ResultLink != nil {
		link := p.Result.ResultLink.URL
		dto.ResultLink = &link
	}

	// Форматируем время
	if p.GetElapsedTimeSec() != nil {
		formatted := p.ElapsedTimeFormatted()
		dto.ElapsedTime = &formatted
	}
	if p.GetMovingTimeSec() != nil {
		formatted := p.MovingTimeFormatted()
		dto.MovingTime = &formatted
	}

	// Метрики и вычисляемые поля заезда переносим из ResultDTO, чтобы
	// ParticipantDTO и ResultDTO не расходились (одна точка истины — FromResult).
	if p.Result != nil {
		if rd := FromResult(p.Result); rd != nil {
			dto.StartedAt = rd.StartedAt
			dto.RideFinishedAt = rd.FinishedAt
			dto.DistanceMeters = rd.DistanceMeters
			dto.AvgHeartRate = rd.AvgHeartRate
			dto.MaxHeartRate = rd.MaxHeartRate
			dto.PeakSpeedKmh = rd.PeakSpeedKmh
			dto.AvgCadence = rd.AvgCadence
			dto.Calories = rd.Calories
			dto.RideDate = rd.RideDate
			dto.IdleTimeSec = rd.IdleTimeSec
			dto.IdleTime = rd.IdleTime
			dto.AvgSpeedKmh = rd.AvgSpeedKmh
			dto.AvgMovingSpeedKmh = rd.AvgMovingSpeedKmh
		}
	}

	return dto
}

// ParticipantDetailDTO представляет детальную информацию об участнике
type ParticipantDetailDTO struct {
	*ParticipantDTO
	Gifts  []*GiftDTO            `json:"gifts"`  // подарки от участника
	Prizes []*PrizeAssignmentDTO `json:"prizes"` // полученные призы
}

// ParticipantListResponse представляет ответ со списком участников.
// Total — полное количество с учётом всех фильтров (не размер страницы).
type ParticipantListResponse struct {
	Participants []*ParticipantDTO `json:"participants"`
	Total        int               `json:"total"`
	Page         int               `json:"page,omitempty"`
	PageSize     int               `json:"page_size,omitempty"`
}
