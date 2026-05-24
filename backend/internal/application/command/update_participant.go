package command

import (
	"context"
	"errors"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// UpdateParticipantCommand представляет команду обновления участника
type UpdateParticipantCommand struct {
	ParticipantID uint
	BikeType      *string
	Gender        *string
	Notes         *string
}

// UpdateParticipantHandler обрабатывает обновление участника
type UpdateParticipantHandler struct {
	participantRepo repository.ParticipantRepository
}

// NewUpdateParticipantHandler создаёт новый handler
func NewUpdateParticipantHandler(
	participantRepo repository.ParticipantRepository,
) *UpdateParticipantHandler {
	return &UpdateParticipantHandler{
		participantRepo: participantRepo,
	}
}

// Handle выполняет команду обновления участника
func (h *UpdateParticipantHandler) Handle(ctx context.Context, cmd UpdateParticipantCommand) (*entity.Participant, error) {
	// Находим участника
	participant, err := h.participantRepo.FindByID(ctx, cmd.ParticipantID)
	if err != nil {
		return nil, errors.New("participant not found")
	}

	// Обновляем поля, если они указаны
	if cmd.BikeType != nil {
		bikeType, err := valueobject.NewBikeType(*cmd.BikeType)
		if err != nil {
			return nil, ErrInvalidBikeType
		}
		participant.BikeType = bikeType
	}

	if cmd.Gender != nil {
		gender, err := valueobject.NewGender(*cmd.Gender)
		if err != nil {
			return nil, ErrInvalidGender
		}
		participant.Gender = gender
	}

	if cmd.Notes != nil {
		participant.Notes = *cmd.Notes
	}

	// Сохраняем изменения в БД
	if err := h.participantRepo.Update(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to update participant: %w", err)
	}

	return participant, nil
}
