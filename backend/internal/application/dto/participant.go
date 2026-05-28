package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
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
}

// FromParticipant создаёт DTO из entity.Participant
func FromParticipant(p *entity.Participant) *ParticipantDTO {
	dto := &ParticipantDTO{
		ID:             p.ID,
		UserID:         p.UserID,
		EventID:        p.EventID,
		BikeType:       string(p.BikeType),
		Gender:         string(p.Gender),
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

	return dto
}

// ParticipantDetailDTO представляет детальную информацию об участнике
type ParticipantDetailDTO struct {
	*ParticipantDTO
	Gifts  []*GiftDTO            `json:"gifts"`  // подарки от участника
	Prizes []*PrizeAssignmentDTO `json:"prizes"` // полученные призы
}

// ParticipantListResponse представляет ответ со списком участников
type ParticipantListResponse struct {
	Participants []*ParticipantDTO `json:"participants"`
	Total        int               `json:"total"`
}
