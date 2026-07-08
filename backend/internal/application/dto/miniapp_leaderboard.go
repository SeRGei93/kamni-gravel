package dto

import (
	"strings"
	"time"

	"gravel_bot/internal/domain/entity"
)

// MiniappLeaderboardEntryDTO — публичное представление участника для лидерборда
// Mini App. Содержит только те поля, которые допустимо показывать всем
// участникам события: место, отображаемое имя, категорию и метрики заезда.
// Намеренно НЕ содержит user_id, notes, has_gift, призы и дату регистрации,
// чтобы не раскрывать приватные/административные данные в публичном экране.
type MiniappLeaderboardEntryDTO struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Gender     string `json:"gender"`
	BikeType   string `json:"bike_type"`
	Status     string `json:"status"` // active / dnf / disqualified
	IsFinished bool   `json:"is_finished"`
	Place      int    `json:"place"` // место в абсолютном зачёте (0 если нет)

	ElapsedTime    *string `json:"elapsed_time,omitempty"`     // полное время, ЧЧ:ММ:СС
	ElapsedTimeSec *int    `json:"elapsed_time_sec,omitempty"` // полное время в секундах (для сортировки/ранжирования)
	MovingTime     *string `json:"moving_time,omitempty"`      // чистое время, ЧЧ:ММ:СС
	MovingTimeSec  *int    `json:"moving_time_sec,omitempty"`
	IdleTime       *string `json:"idle_time,omitempty"` // простой, ЧЧ:ММ:СС

	ResultLink  *string    `json:"result_link,omitempty"`  // ссылка на результат (Strava)
	SubmittedAt *time.Time `json:"submitted_at,omitempty"` // дата отправки результата

	RideDate          *string  `json:"ride_date,omitempty"`            // дата проезда, YYYY-MM-DD
	DistanceMeters    *int     `json:"distance_meters,omitempty"`      // дистанция, метры
	AvgSpeedKmh       *float64 `json:"avg_speed_kmh,omitempty"`        // средняя скорость, км/ч
	AvgMovingSpeedKmh *float64 `json:"avg_moving_speed_kmh,omitempty"` // средняя скорость в движении, км/ч
	PeakSpeedKmh      *float64 `json:"peak_speed_kmh,omitempty"`       // пиковая скорость, км/ч
	AvgHeartRate      *int     `json:"avg_heart_rate,omitempty"`       // средний пульс
	MaxHeartRate      *int     `json:"max_heart_rate,omitempty"`       // максимальный пульс
	AvgCadence        *int     `json:"avg_cadence,omitempty"`          // средний каденс
	Calories          *int     `json:"calories,omitempty"`             // калории
}

// MiniappLeaderboardResponse — ответ со списком участников лидерборда.
// Total — полное количество участников активного события.
type MiniappLeaderboardResponse struct {
	Participants []*MiniappLeaderboardEntryDTO `json:"participants"`
	Total        int                           `json:"total"`
}

// NewMiniappLeaderboardEntry строит публичную запись лидерборда из участника и
// заранее рассчитанного места. Форматирование времени и вычисляемые метрики
// заезда переиспользуются из FromParticipant, чтобы оставалась одна точка
// истины (ParticipantDTO/ResultDTO не расходятся с лидербордом).
func NewMiniappLeaderboardEntry(p *entity.Participant, place int) *MiniappLeaderboardEntryDTO {
	full := FromParticipant(p)

	return &MiniappLeaderboardEntryDTO{
		ID:         full.ID,
		Name:       miniappDisplayName(full),
		Gender:     full.Gender,
		BikeType:   full.BikeType,
		Status:     full.Status,
		IsFinished: full.IsFinished,
		Place:      place,

		ElapsedTime:    full.ElapsedTime,
		ElapsedTimeSec: full.ElapsedTimeSec,
		MovingTime:     full.MovingTime,
		MovingTimeSec:  full.MovingTimeSec,
		IdleTime:       full.IdleTime,

		ResultLink:  full.ResultLink,
		SubmittedAt: full.FinishedAt, // FinishedAt в ParticipantDTO — это дата отправки результата

		RideDate:          full.RideDate,
		DistanceMeters:    full.DistanceMeters,
		AvgSpeedKmh:       full.AvgSpeedKmh,
		AvgMovingSpeedKmh: full.AvgMovingSpeedKmh,
		PeakSpeedKmh:      full.PeakSpeedKmh,
		AvgHeartRate:      full.AvgHeartRate,
		MaxHeartRate:      full.MaxHeartRate,
		AvgCadence:        full.AvgCadence,
		Calories:          full.Calories,
	}
}

// miniappDisplayName выбирает отображаемое имя участника, не раскрывая числовой
// Telegram ID: сначала имя+фамилия, затем username, иначе обобщённая подпись.
func miniappDisplayName(p *ParticipantDTO) string {
	name := strings.TrimSpace(strings.TrimSpace(p.FirstName) + " " + strings.TrimSpace(p.LastName))
	if name != "" {
		return name
	}
	if strings.TrimSpace(p.Username) != "" {
		return p.Username
	}
	return "Участник"
}
