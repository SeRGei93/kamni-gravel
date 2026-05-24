package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
)

// PrizeAssignmentDTO представляет DTO назначения приза для API
type PrizeAssignmentDTO struct {
	ID            uint      `json:"id"`
	ParticipantID uint      `json:"participant_id"`
	GiftID        uint      `json:"gift_id"`
	Comment       string    `json:"comment,omitempty"`
	AssignedAt    time.Time `json:"assigned_at"`
	Gift          *GiftDTO  `json:"gift,omitempty"`
}

// FromPrizeAssignment создаёт DTO из entity.PrizeAssignment
func FromPrizeAssignment(pa *entity.PrizeAssignment) *PrizeAssignmentDTO {
	dto := &PrizeAssignmentDTO{
		ID:            pa.ID,
		ParticipantID: pa.ParticipantID,
		GiftID:        pa.GiftID,
		Comment:       pa.Comment,
		AssignedAt:    pa.AssignedAt,
	}

	// Добавляем подарок
	if pa.Gift != nil {
		dto.Gift = FromGift(pa.Gift)
	}

	return dto
}

// PrizeAssignmentListResponse представляет ответ со списком назначений призов
type PrizeAssignmentListResponse struct {
	PrizeAssignments []*PrizeAssignmentDTO `json:"prize_assignments"`
	Total            int                   `json:"total"`
}
