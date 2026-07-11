package command

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gravel_bot/internal/domain/repository"
)

var (
	// ErrManualGiftOwnerForbidden hides gifts that do not belong to the Mini App user.
	ErrManualGiftOwnerForbidden = errors.New("manual gift does not belong to actor")
)

// ManualGiftRecipientActor describes the verified Mini App actor. Infrastructure
// must populate it only from Telegram init-data middleware context.
type ManualGiftRecipientActor struct {
	TelegramUserID int64
}

// SetManualGiftRecipientCommand replaces or clears a recipient selected by the gift owner.
type SetManualGiftRecipientCommand struct {
	GiftID uint
	// EventID scopes an owner action to the event selected by the server.
	// Zero preserves compatibility for internal callers that do not have an
	// event scope; Mini App requests must always provide it.
	EventID                uint
	Actor                  ManualGiftRecipientActor
	RecipientParticipantID *uint
}

// SetManualGiftRecipientHandler applies owner-scoped recipient changes.
type SetManualGiftRecipientHandler struct {
	giftRepo        repository.ManualGiftRepository
	participantRepo repository.ParticipantRepository
}

func NewSetManualGiftRecipientHandler(
	giftRepo repository.ManualGiftRepository,
	participantRepo repository.ParticipantRepository,
) *SetManualGiftRecipientHandler {
	return &SetManualGiftRecipientHandler{
		giftRepo:        giftRepo,
		participantRepo: participantRepo,
	}
}

func (h *SetManualGiftRecipientHandler) Handle(ctx context.Context, cmd SetManualGiftRecipientCommand) error {
	log.Printf(
		"DEBUG manual gift recipient command started: actor_type=telegram gift_id=%d recipient_participant_id=%s",
		cmd.GiftID,
		manualGiftRecipientIDLogValue(cmd.RecipientParticipantID),
	)

	gift, err := h.giftRepo.FindByID(ctx, cmd.GiftID)
	if err != nil {
		if errors.Is(err, repository.ErrGiftNotFound) {
			log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d reason=gift_not_found", cmd.GiftID)
			return ErrGiftNotFound
		}
		log.Printf("ERROR manual gift recipient command failed: actor_type=telegram gift_id=%d stage=find_gift error=%v", cmd.GiftID, err)
		return fmt.Errorf("find manual gift: %w", err)
	}
	if gift.UserID != cmd.Actor.TelegramUserID {
		log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d reason=owner_mismatch", cmd.GiftID)
		return ErrManualGiftOwnerForbidden
	}
	if cmd.EventID != 0 && gift.EventID != cmd.EventID {
		log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d reason=event_mismatch", cmd.GiftID)
		return ErrManualGiftOwnerForbidden
	}
	if !gift.ManualDistribution {
		log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d reason=manual_distribution_disabled", cmd.GiftID)
		return ErrManualGiftNotManual
	}
	if manualGiftRecipientIDsEqual(gift.ManualRecipientParticipantID, cmd.RecipientParticipantID) {
		log.Printf("DEBUG manual gift recipient command idempotent: actor_type=telegram gift_id=%d recipient_participant_id=%s", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.RecipientParticipantID))
		return nil
	}

	if cmd.RecipientParticipantID != nil {
		recipient, err := h.participantRepo.FindByID(ctx, *cmd.RecipientParticipantID)
		if err != nil {
			if errors.Is(err, repository.ErrParticipantNotFound) {
				log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d recipient_participant_id=%s reason=recipient_not_found", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.RecipientParticipantID))
				return ErrManualGiftRecipientNotFound
			}
			log.Printf("ERROR manual gift recipient command failed: actor_type=telegram gift_id=%d recipient_participant_id=%s stage=find_recipient error=%v", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.RecipientParticipantID), err)
			return fmt.Errorf("find manual gift recipient: %w", err)
		}
		if recipient.EventID != gift.EventID {
			log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d recipient_participant_id=%s reason=recipient_event_mismatch", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.RecipientParticipantID))
			return ErrManualGiftRecipientEvent
		}
	}

	if err := h.giftRepo.SetManualRecipient(ctx, cmd.GiftID, cmd.RecipientParticipantID); err != nil {
		return mapManualGiftRecipientRepositoryError(cmd, err)
	}

	log.Printf(
		"INFO manual gift recipient command completed: actor_type=telegram gift_id=%d recipient_participant_id=%s",
		cmd.GiftID,
		manualGiftRecipientIDLogValue(cmd.RecipientParticipantID),
	)
	return nil
}

func mapManualGiftRecipientRepositoryError(cmd SetManualGiftRecipientCommand, err error) error {
	switch {
	case errors.Is(err, repository.ErrGiftNotFound):
		log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d reason=gift_not_found_at_write", cmd.GiftID)
		return ErrGiftNotFound
	case errors.Is(err, repository.ErrManualDistributionDisabled):
		log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d reason=manual_distribution_disabled_at_write", cmd.GiftID)
		return ErrManualGiftNotManual
	case errors.Is(err, repository.ErrManualRecipientNotFound):
		log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d recipient_participant_id=%s reason=recipient_not_found_at_write", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.RecipientParticipantID))
		return ErrManualGiftRecipientNotFound
	case errors.Is(err, repository.ErrManualRecipientEventMismatch):
		log.Printf("WARN manual gift recipient command rejected: actor_type=telegram gift_id=%d recipient_participant_id=%s reason=recipient_event_mismatch_at_write", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.RecipientParticipantID))
		return ErrManualGiftRecipientEvent
	default:
		log.Printf("ERROR manual gift recipient command failed: actor_type=telegram gift_id=%d recipient_participant_id=%s stage=set_recipient error=%v", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.RecipientParticipantID), err)
		return fmt.Errorf("set manual gift recipient: %w", err)
	}
}

func manualGiftRecipientIDsEqual(left, right *uint) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
