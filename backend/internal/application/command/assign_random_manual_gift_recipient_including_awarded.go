package command

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/repository"
)

var (
	// ErrManualGiftNoEligibleParticipants means no finished or DNF participant
	// can receive the manual gift, even when existing prizes are allowed.
	ErrManualGiftNoEligibleParticipants = errors.New("no eligible participants")
	// ErrManualGiftRecipientAlreadyAssigned means this one-time random action
	// cannot replace an existing manually selected recipient.
	ErrManualGiftRecipientAlreadyAssigned = errors.New("manual gift recipient is already assigned")
)

// eligibleManualGiftParticipantIDsReader returns eligible participant IDs
// without filtering out automatic or manual prize recipients.
type eligibleManualGiftParticipantIDsReader interface {
	Handle(ctx context.Context, query query.GetEligibleManualGiftParticipantIDsQuery) ([]uint, error)
}

// AssignRandomManualGiftRecipientIncludingAwardedCommand selects a random
// eligible participant for an owner-managed manual gift, including users who
// already hold another prize.
type AssignRandomManualGiftRecipientIncludingAwardedCommand struct {
	GiftID  uint
	EventID uint
	Actor   ManualGiftRecipientActor
}

// AssignRandomManualGiftRecipientIncludingAwardedHandler assigns an unassigned
// manual gift to a cryptographically selected eligible participant.
type AssignRandomManualGiftRecipientIncludingAwardedHandler struct {
	participantIDsReader eligibleManualGiftParticipantIDsReader
	setRecipientHandler  *SetManualGiftRecipientHandler
	recipientWriter      repository.InitialManualGiftRecipientRepository
	randomIndex          func(max int) (int, error)
}

// NewAssignRandomManualGiftRecipientIncludingAwardedHandler creates the
// owner-scoped random-recipient handler backed by crypto/rand.
func NewAssignRandomManualGiftRecipientIncludingAwardedHandler(
	participantIDsReader eligibleManualGiftParticipantIDsReader,
	setRecipientHandler *SetManualGiftRecipientHandler,
	recipientWriter repository.InitialManualGiftRecipientRepository,
) *AssignRandomManualGiftRecipientIncludingAwardedHandler {
	return newAssignRandomManualGiftRecipientIncludingAwardedHandler(
		participantIDsReader,
		setRecipientHandler,
		recipientWriter,
		cryptoRandomIndex,
	)
}

func newAssignRandomManualGiftRecipientIncludingAwardedHandler(
	participantIDsReader eligibleManualGiftParticipantIDsReader,
	setRecipientHandler *SetManualGiftRecipientHandler,
	recipientWriter repository.InitialManualGiftRecipientRepository,
	randomIndex func(max int) (int, error),
) *AssignRandomManualGiftRecipientIncludingAwardedHandler {
	return &AssignRandomManualGiftRecipientIncludingAwardedHandler{
		participantIDsReader: participantIDsReader,
		setRecipientHandler:  setRecipientHandler,
		recipientWriter:      recipientWriter,
		randomIndex:          randomIndex,
	}
}

// Handle validates the owner/event/manual invariants before selecting an
// eligible participant and atomically claims the previously unassigned gift.
func (h *AssignRandomManualGiftRecipientIncludingAwardedHandler) Handle(
	ctx context.Context,
	cmd AssignRandomManualGiftRecipientIncludingAwardedCommand,
) (uint, error) {
	log.Printf("DEBUG random manual gift recipient including awarded command started: gift_id=%d event_id=%d", cmd.GiftID, cmd.EventID)
	if h.participantIDsReader == nil || h.setRecipientHandler == nil || h.recipientWriter == nil || h.randomIndex == nil {
		log.Printf("ERROR random manual gift recipient including awarded command unavailable: gift_id=%d event_id=%d", cmd.GiftID, cmd.EventID)
		return 0, errors.New("random manual gift recipient including awarded dependencies are not configured")
	}

	gift, err := h.setRecipientHandler.manualGiftForCommand(ctx, SetManualGiftRecipientCommand{
		GiftID:  cmd.GiftID,
		EventID: cmd.EventID,
		Actor:   cmd.Actor,
	})
	if err != nil {
		return 0, err
	}
	if gift.ManualRecipientParticipantID != nil {
		log.Printf("WARN random manual gift recipient including awarded command rejected: gift_id=%d event_id=%d reason=recipient_already_assigned", gift.ID, gift.EventID)
		return 0, ErrManualGiftRecipientAlreadyAssigned
	}

	candidateIDs, err := h.participantIDsReader.Handle(ctx, query.GetEligibleManualGiftParticipantIDsQuery{EventID: cmd.EventID})
	if err != nil {
		log.Printf("ERROR random manual gift recipient including awarded command failed: gift_id=%d event_id=%d stage=find_candidates error=%v", cmd.GiftID, cmd.EventID, err)
		return 0, fmt.Errorf("find random manual gift recipient candidates including awarded participants: %w", err)
	}
	if len(candidateIDs) == 0 {
		log.Printf("WARN random manual gift recipient including awarded command rejected: gift_id=%d event_id=%d reason=no_eligible_participants", cmd.GiftID, cmd.EventID)
		return 0, ErrManualGiftNoEligibleParticipants
	}

	index, err := h.randomIndex(len(candidateIDs))
	if err != nil {
		log.Printf("ERROR random manual gift recipient including awarded command failed: gift_id=%d event_id=%d stage=choose_random_candidate error=%v", cmd.GiftID, cmd.EventID, err)
		return 0, fmt.Errorf("choose random manual gift recipient including awarded participants: %w", err)
	}
	if index < 0 || index >= len(candidateIDs) {
		log.Printf("ERROR random manual gift recipient including awarded command failed: gift_id=%d event_id=%d stage=choose_random_candidate invalid_index=%d candidate_count=%d", cmd.GiftID, cmd.EventID, index, len(candidateIDs))
		return 0, fmt.Errorf("choose random manual gift recipient including awarded participants: invalid random index %d", index)
	}

	recipientID := candidateIDs[index]
	if err := h.recipientWriter.AssignInitialManualRecipient(ctx, cmd.GiftID, recipientID); err != nil {
		return 0, mapInitialManualGiftRecipientRepositoryError(cmd, recipientID, err)
	}

	log.Printf("INFO random manual gift recipient including awarded command completed: gift_id=%d event_id=%d candidate_count=%d recipient_participant_id=%d", cmd.GiftID, cmd.EventID, len(candidateIDs), recipientID)
	return recipientID, nil
}

func mapInitialManualGiftRecipientRepositoryError(
	cmd AssignRandomManualGiftRecipientIncludingAwardedCommand,
	recipientID uint,
	err error,
) error {
	switch {
	case errors.Is(err, repository.ErrGiftNotFound):
		log.Printf("WARN [FIX:owner-random-recipient-cas] command rejected: gift_id=%d event_id=%d reason=gift_not_found_at_write", cmd.GiftID, cmd.EventID)
		return ErrGiftNotFound
	case errors.Is(err, repository.ErrManualDistributionDisabled):
		log.Printf("WARN [FIX:owner-random-recipient-cas] command rejected: gift_id=%d event_id=%d reason=manual_distribution_disabled_at_write", cmd.GiftID, cmd.EventID)
		return ErrManualGiftNotManual
	case errors.Is(err, repository.ErrRandomGiftRecipientAlreadyAssigned):
		log.Printf("WARN [FIX:owner-random-recipient-cas] command rejected: gift_id=%d event_id=%d reason=recipient_already_assigned_at_write", cmd.GiftID, cmd.EventID)
		return ErrManualGiftRecipientAlreadyAssigned
	case errors.Is(err, repository.ErrManualRecipientNotFound):
		log.Printf("WARN [FIX:owner-random-recipient-cas] command rejected: gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_not_found_at_write", cmd.GiftID, cmd.EventID, recipientID)
		return ErrManualGiftRecipientNotFound
	case errors.Is(err, repository.ErrManualRecipientEventMismatch):
		log.Printf("WARN [FIX:owner-random-recipient-cas] command rejected: gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_event_mismatch_at_write", cmd.GiftID, cmd.EventID, recipientID)
		return ErrManualGiftRecipientEvent
	case errors.Is(err, repository.ErrManualRecipientIneligible):
		log.Printf("WARN [FIX:owner-random-recipient-cas] command rejected: gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_ineligible_at_write", cmd.GiftID, cmd.EventID, recipientID)
		return ErrManualGiftRecipientIneligible
	default:
		log.Printf("ERROR [FIX:owner-random-recipient-cas] command failed: gift_id=%d event_id=%d recipient_participant_id=%d stage=claim_initial_recipient error=%v", cmd.GiftID, cmd.EventID, recipientID, err)
		return fmt.Errorf("claim random manual gift recipient including awarded participants: %w", err)
	}
}
