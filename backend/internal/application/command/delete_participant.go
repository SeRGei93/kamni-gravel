package command

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gravel_bot/internal/domain/repository"
)

// DeleteParticipantCommand представляет команду безопасного удаления участника.
type DeleteParticipantCommand struct {
	ParticipantID uint
}

// DeleteParticipantHandler обрабатывает безопасное удаление участника.
type DeleteParticipantHandler struct {
	participantRepo repository.ParticipantRepository
}

// NewDeleteParticipantHandler создаёт новый handler.
func NewDeleteParticipantHandler(participantRepo repository.ParticipantRepository) *DeleteParticipantHandler {
	return &DeleteParticipantHandler{participantRepo: participantRepo}
}

// Handle выполняет команду безопасного удаления участника.
func (h *DeleteParticipantHandler) Handle(ctx context.Context, cmd DeleteParticipantCommand) error {
	log.Printf("INFO Participant deletion requested: participant_id=%d", cmd.ParticipantID)

	participant, err := h.participantRepo.FindByID(ctx, cmd.ParticipantID)
	if errors.Is(err, repository.ErrParticipantNotFound) {
		log.Printf("WARN Participant deletion skipped: participant_id=%d stage=find_participant error=%v", cmd.ParticipantID, err)
		return ErrParticipantNotFound
	}
	if err != nil {
		log.Printf("ERROR Participant deletion failed: participant_id=%d stage=find_participant error=%v", cmd.ParticipantID, err)
		return fmt.Errorf("find participant %d: %w", cmd.ParticipantID, err)
	}

	log.Printf("INFO Participant deletion started: participant_id=%d event_id=%d", participant.ID, participant.EventID)
	if err := h.participantRepo.DeleteWithResultCriteria(ctx, participant.ID); err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf("WARN Participant deletion skipped: participant_id=%d event_id=%d stage=delete_with_result_criteria error=%v", participant.ID, participant.EventID, err)
			return ErrParticipantNotFound
		}
		log.Printf("ERROR Participant deletion failed: participant_id=%d event_id=%d stage=delete_with_result_criteria error=%v", participant.ID, participant.EventID, err)
		return fmt.Errorf("delete participant %d: %w", participant.ID, err)
	}

	log.Printf("INFO Participant deletion completed: participant_id=%d event_id=%d", participant.ID, participant.EventID)
	return nil
}
