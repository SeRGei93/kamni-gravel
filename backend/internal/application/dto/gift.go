package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

// GiftDTO представляет DTO подарка для API
type GiftDTO struct {
	ID             uint                 `json:"id"`
	UserID         int64                `json:"user_id"`
	Username       string               `json:"username,omitempty"`
	FirstName      string               `json:"first_name,omitempty"`
	LastName       string               `json:"last_name,omitempty"`
	EventID        uint                 `json:"event_id"`
	Description    string               `json:"description"`
	GenderFilter   string               `json:"gender_filter,omitempty"`    // all, male, female
	BikeTypeFilter string               `json:"bike_type_filter,omitempty"` // all, gravel, mtb, road, single_speed, tandem
	ReviewStatus   string               `json:"review_status"`
	Place          *int                 `json:"place,omitempty"` // место (позиция)
	PlaceRule      *GiftPlaceRuleDTO    `json:"place_rule"`
	Attachments    []*GiftAttachmentDTO `json:"attachments,omitempty"`
	Criteria       []*CriteriaDTO       `json:"criteria,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
}

// GiftPlaceRuleDTO представляет правило привязки подарка к местам.
type GiftPlaceRuleDTO struct {
	Type      string `json:"type"`
	Places    []int  `json:"places,omitempty"`
	LastCount *int   `json:"last_count,omitempty"`
}

// GiftAttachmentDTO представляет DTO прикреплённого файла
type GiftAttachmentDTO struct {
	ID             uint   `json:"id"`
	GiftID         uint   `json:"gift_id"`
	TelegramFileID string `json:"telegram_file_id"`
	FileType       string `json:"file_type"` // photo, document
}

// FromGift создаёт DTO из entity.Gift
func FromGift(g *entity.Gift) *GiftDTO {
	dto := &GiftDTO{
		ID:             g.ID,
		UserID:         g.UserID,
		EventID:        g.EventID,
		Description:    g.Description,
		GenderFilter:   g.GenderFilter,
		BikeTypeFilter: g.BikeTypeFilter,
		ReviewStatus:   g.ReviewStatus.String(),
		Place:          g.FirstLegacyPlace(),
		PlaceRule:      FromGiftPlaceRule(g.PlaceRule),
		CreatedAt:      g.CreatedAt,
	}

	// Добавляем данные пользователя
	if g.User != nil {
		dto.Username = g.User.Username
		dto.FirstName = g.User.FirstName
		dto.LastName = g.User.LastName
	}

	// Добавляем прикреплённые файлы
	if len(g.Attachments) > 0 {
		dto.Attachments = make([]*GiftAttachmentDTO, len(g.Attachments))
		for i, a := range g.Attachments {
			dto.Attachments[i] = &GiftAttachmentDTO{
				ID:             a.ID,
				GiftID:         a.GiftID,
				TelegramFileID: a.TelegramFileID,
				FileType:       a.FileType,
			}
		}
	}

	// Добавляем критерии
	if len(g.Criteria) > 0 {
		dto.Criteria = make([]*CriteriaDTO, len(g.Criteria))
		for i, c := range g.Criteria {
			dto.Criteria[i] = FromCriteria(c)
		}
	}

	return dto
}

// FromGiftPlaceRule создаёт DTO правила мест.
func FromGiftPlaceRule(rule valueobject.GiftPlaceRule) *GiftPlaceRuleDTO {
	switch rule.Type() {
	case valueobject.GiftPlaceRuleTypePlaces:
		return &GiftPlaceRuleDTO{
			Type:   string(valueobject.GiftPlaceRuleTypePlaces),
			Places: rule.Places(),
		}
	case valueobject.GiftPlaceRuleTypeLastN:
		lastCount := rule.LastCount()
		return &GiftPlaceRuleDTO{
			Type:      string(valueobject.GiftPlaceRuleTypeLastN),
			LastCount: &lastCount,
		}
	default:
		return nil
	}
}

// GiftListResponse представляет ответ со списком подарков
type GiftListResponse struct {
	Gifts []*GiftDTO `json:"gifts"`
	Total int        `json:"total"`
}
