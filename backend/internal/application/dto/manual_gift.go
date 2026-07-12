package dto

import (
	"fmt"
	"strings"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

// ManualGiftDTO is a protected read model for dashboard and owner Mini App
// screens. It deliberately has no Telegram user ID for the recipient.
type ManualGiftDTO struct {
	ID                 uint                    `json:"id"`
	EventID            uint                    `json:"event_id"`
	Description        string                  `json:"description"`
	GenderFilter       string                  `json:"gender_filter,omitempty"`
	BikeTypeFilter     string                  `json:"bike_type_filter,omitempty"`
	ReviewStatus       string                  `json:"review_status"`
	ManualDistribution bool                    `json:"manual_distribution"`
	Place              *int                    `json:"place,omitempty"`
	PlaceRule          *GiftPlaceRuleDTO       `json:"place_rule"`
	Attachments        []*GiftAttachmentDTO    `json:"attachments,omitempty"`
	Criteria           []*CriteriaDTO          `json:"criteria,omitempty"`
	Recipient          *ManualGiftRecipientDTO `json:"recipient,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
}

// ManualGiftRecipientDTO is the minimal participant summary safe for
// authenticated gift owner and administrator screens.
type ManualGiftRecipientDTO struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username,omitempty"`
	Status      string `json:"status"`
}

// ManualGiftListResponse is the protected manual-gift management response.
type ManualGiftListResponse struct {
	Gifts []*ManualGiftDTO `json:"gifts"`
}

// MiniappParticipantOptionDTO is the minimal selectable participant model.
// It never exposes Telegram IDs, notes, registration dates, result metrics, or
// award details beyond the HasPrize selection hint.
type MiniappParticipantOptionDTO struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username,omitempty"`
	Status      string `json:"status"`
	HasPrize    bool   `json:"has_prize"`
}

func FromManualGift(gift *entity.Gift) *ManualGiftDTO {
	if gift == nil {
		return nil
	}

	dto := &ManualGiftDTO{
		ID:                 gift.ID,
		EventID:            gift.EventID,
		Description:        gift.Description,
		GenderFilter:       gift.GenderFilter,
		BikeTypeFilter:     gift.BikeTypeFilter,
		ReviewStatus:       gift.ReviewStatus.String(),
		ManualDistribution: gift.ManualDistribution,
		Place:              gift.FirstLegacyPlace(),
		PlaceRule:          FromGiftPlaceRule(gift.PlaceRule),
		CreatedAt:          gift.CreatedAt,
	}
	if len(gift.Attachments) > 0 {
		dto.Attachments = make([]*GiftAttachmentDTO, len(gift.Attachments))
		for index, attachment := range gift.Attachments {
			dto.Attachments[index] = &GiftAttachmentDTO{
				ID:             attachment.ID,
				GiftID:         attachment.GiftID,
				TelegramFileID: attachment.TelegramFileID,
				FileType:       attachment.FileType,
			}
		}
	}
	if len(gift.Criteria) > 0 {
		dto.Criteria = make([]*CriteriaDTO, len(gift.Criteria))
		for index, criteria := range gift.Criteria {
			dto.Criteria[index] = FromCriteria(criteria)
		}
	}
	if gift.ManualRecipient != nil {
		dto.Recipient = FromManualGiftRecipient(gift.ManualRecipient)
	}
	return dto
}

func FromManualGiftRecipient(participant *entity.Participant) *ManualGiftRecipientDTO {
	if participant == nil {
		return nil
	}
	return &ManualGiftRecipientDTO{
		ID:          participant.ID,
		DisplayName: participantDisplayName(participant),
		Username:    participantUsername(participant),
		Status:      participantStatusString(participant),
	}
}

func FromMiniappParticipantOption(participant *entity.Participant) *MiniappParticipantOptionDTO {
	if participant == nil {
		return nil
	}
	return &MiniappParticipantOptionDTO{
		ID:          participant.ID,
		DisplayName: participantDisplayName(participant),
		Username:    participantUsername(participant),
		Status:      participantStatusString(participant),
	}
}

func participantDisplayName(participant *entity.Participant) string {
	if participant.User != nil {
		name := strings.TrimSpace(strings.Join([]string{
			strings.TrimSpace(participant.User.FirstName),
			strings.TrimSpace(participant.User.LastName),
		}, " "))
		if name != "" {
			return name
		}
		if username := strings.TrimSpace(participant.User.Username); username != "" {
			return "@" + username
		}
	}
	return fmt.Sprintf("Участник #%d", participant.ID)
}

func participantUsername(participant *entity.Participant) string {
	if participant.User == nil {
		return ""
	}
	return strings.TrimSpace(participant.User.Username)
}

func participantStatusString(participant *entity.Participant) string {
	status := participant.Status
	if status == "" {
		status = valueobject.ParticipantStatusActive
	}
	return string(status)
}
