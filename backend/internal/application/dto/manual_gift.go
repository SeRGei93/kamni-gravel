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
	ReviewStatus       string                  `json:"review_status"`
	ManualDistribution bool                    `json:"manual_distribution"`
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

// MiniappParticipantOptionDTO is the minimal selectable participant model.
// It never exposes Telegram IDs, notes, registration dates, or result metrics.
type MiniappParticipantOptionDTO struct {
	ID          uint   `json:"id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username,omitempty"`
	Status      string `json:"status"`
}

func FromManualGift(gift *entity.Gift) *ManualGiftDTO {
	if gift == nil {
		return nil
	}

	dto := &ManualGiftDTO{
		ID:                 gift.ID,
		EventID:            gift.EventID,
		Description:        gift.Description,
		ReviewStatus:       gift.ReviewStatus.String(),
		ManualDistribution: gift.ManualDistribution,
		CreatedAt:          gift.CreatedAt,
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
