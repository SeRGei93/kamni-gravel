package command

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// WithdrawParticipantCommand представляет команду выхода участника из события.
type WithdrawParticipantCommand struct {
	UserID  int64
	EventID uint
}

// WithdrawParticipantHandler обрабатывает выход участника из события.
type WithdrawParticipantHandler struct {
	participantRepo repository.ParticipantRepository
}

// NewWithdrawParticipantHandler создаёт новый handler.
func NewWithdrawParticipantHandler(participantRepo repository.ParticipantRepository) *WithdrawParticipantHandler {
	return &WithdrawParticipantHandler{participantRepo: participantRepo}
}

// Handle выполняет команду выхода участника из события.
func (h *WithdrawParticipantHandler) Handle(ctx context.Context, cmd WithdrawParticipantCommand) (*entity.Participant, error) {
	participant, err := h.participantRepo.FindByUserAndEvent(ctx, cmd.UserID, cmd.EventID)
	if errors.Is(err, repository.ErrParticipantNotFound) {
		log.Printf(
			"WARN Participant withdrawal skipped: telegram_user_id=%d event_id=%d reason=not_found",
			cmd.UserID,
			cmd.EventID,
		)
		return nil, ErrParticipantNotFound
	}
	if err != nil {
		log.Printf(
			"ERROR Participant withdrawal failed: telegram_user_id=%d event_id=%d stage=find_participant error=%v",
			cmd.UserID,
			cmd.EventID,
			err,
		)
		return nil, fmt.Errorf("find participant by user and event: %w", err)
	}

	log.Printf(
		"INFO Participant withdrawal requested: telegram_user_id=%d participant_id=%d event_id=%d",
		cmd.UserID,
		participant.ID,
		participant.EventID,
	)

	if err := h.participantRepo.DeleteWithResultCriteria(ctx, participant.ID); err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf(
				"WARN Participant withdrawal skipped: telegram_user_id=%d participant_id=%d event_id=%d stage=delete_with_result_criteria reason=not_found",
				cmd.UserID,
				participant.ID,
				participant.EventID,
			)
			return nil, ErrParticipantNotFound
		}

		log.Printf(
			"ERROR Participant withdrawal failed: telegram_user_id=%d participant_id=%d event_id=%d stage=delete_with_result_criteria error=%v",
			cmd.UserID,
			participant.ID,
			participant.EventID,
			err,
		)
		return nil, fmt.Errorf("delete participant %d: %w", participant.ID, err)
	}

	log.Printf(
		"INFO Participant withdrawal completed: telegram_user_id=%d participant_id=%d event_id=%d",
		cmd.UserID,
		participant.ID,
		participant.EventID,
	)

	return participant, nil
}
