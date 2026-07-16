package command

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// AssignRandomAdminGiftRecipientIncludingAwardedCommand requests an
// administrator-driven assignment of an already-manual gift to any eligible
// participant, including participants who already have other prizes.
type AssignRandomAdminGiftRecipientIncludingAwardedCommand struct {
	GiftID  uint
	AdminID uint
}

// AssignRandomAdminGiftRecipientIncludingAwardedResult describes the completed
// assignment without exposing the recipient outside the protected admin flow.
type AssignRandomAdminGiftRecipientIncludingAwardedResult struct {
	GiftID                 uint
	EventID                uint
	RecipientParticipantID uint
}

type eligibleManualParticipantIDsReader interface {
	Handle(ctx context.Context, query query.GetEligibleManualGiftParticipantIDsQuery) ([]uint, error)
}

// AssignRandomAdminGiftRecipientIncludingAwardedHandler protects the new admin
// workflow. Unlike AssignRandomAdminGiftRecipientHandler, it requires an
// already-manual gift and deliberately does not filter candidates by prizes.
type AssignRandomAdminGiftRecipientIncludingAwardedHandler struct {
	giftRepo           repository.GiftRepository
	recipientWriter    repository.RandomManualGiftRecipientIncludingAwardedRepository
	participantRepo    repository.ParticipantRepository
	candidateIDsReader eligibleManualParticipantIDsReader
	randomIndex        func(max int) (int, error)
}

// NewAssignRandomAdminGiftRecipientIncludingAwardedHandler creates the admin
// random-recipient handler backed by crypto/rand.
func NewAssignRandomAdminGiftRecipientIncludingAwardedHandler(
	gitRepo repository.GiftRepository,
	recipientWriter repository.RandomManualGiftRecipientIncludingAwardedRepository,
	participantRepo repository.ParticipantRepository,
	candidateIDsReader eligibleManualParticipantIDsReader,
) *AssignRandomAdminGiftRecipientIncludingAwardedHandler {
	return newAssignRandomAdminGiftRecipientIncludingAwardedHandler(
		gitRepo,
		recipientWriter,
		participantRepo,
		candidateIDsReader,
		cryptoRandomIndex,
	)
}

func newAssignRandomAdminGiftRecipientIncludingAwardedHandler(
	gitRepo repository.GiftRepository,
	recipientWriter repository.RandomManualGiftRecipientIncludingAwardedRepository,
	participantRepo repository.ParticipantRepository,
	candidateIDsReader eligibleManualParticipantIDsReader,
	randomIndex func(max int) (int, error),
) *AssignRandomAdminGiftRecipientIncludingAwardedHandler {
	return &AssignRandomAdminGiftRecipientIncludingAwardedHandler{
		giftRepo:           gitRepo,
		recipientWriter:    recipientWriter,
		participantRepo:    participantRepo,
		candidateIDsReader: candidateIDsReader,
		randomIndex:        randomIndex,
	}
}

// Handle selects an eligible recipient and atomically writes the assignment.
// The repository repeats the state and eligibility checks in its transaction.
func (h *AssignRandomAdminGiftRecipientIncludingAwardedHandler) Handle(
	ctx context.Context,
	cmd AssignRandomAdminGiftRecipientIncludingAwardedCommand,
) (*AssignRandomAdminGiftRecipientIncludingAwardedResult, error) {
	if h.giftRepo == nil || h.recipientWriter == nil || h.participantRepo == nil || h.candidateIDsReader == nil || h.randomIndex == nil {
		log.Printf("ERROR admin random manual gift recipient including awarded command unavailable: admin_id=%d gift_id=%d", cmd.AdminID, cmd.GiftID)
		return nil, errors.New("admin random manual gift recipient including awarded dependencies are not configured")
	}

	log.Printf("DEBUG admin random manual gift recipient including awarded command started: admin_id=%d gift_id=%d", cmd.AdminID, cmd.GiftID)
	gift, err := h.giftRepo.FindByID(ctx, cmd.GiftID)
	if err != nil {
		if errors.Is(err, repository.ErrGiftNotFound) {
			log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d reason=gift_not_found", cmd.AdminID, cmd.GiftID)
			return nil, ErrGiftNotFound
		}
		log.Printf("ERROR admin random manual gift recipient including awarded command failed: admin_id=%d gift_id=%d stage=find_gift error=%v", cmd.AdminID, cmd.GiftID, err)
		return nil, fmt.Errorf("find manual gift for random recipient including awarded participants: %w", err)
	}
	if gift.ReviewStatus != entity.GiftReviewStatusApproved {
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=gift_not_approved", cmd.AdminID, gift.ID, gift.EventID)
		return nil, ErrAdminRandomGiftNotApproved
	}
	if !gift.ManualDistribution {
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=manual_distribution_disabled", cmd.AdminID, gift.ID, gift.EventID)
		return nil, ErrManualGiftNotManual
	}
	if gift.ManualRecipientParticipantID != nil {
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=recipient_already_assigned", cmd.AdminID, gift.ID, gift.EventID)
		return nil, ErrAdminRandomGiftAlreadyAssigned
	}

	candidateIDs, err := h.candidateIDsReader.Handle(ctx, query.GetEligibleManualGiftParticipantIDsQuery{EventID: gift.EventID})
	if err != nil {
		log.Printf("ERROR admin random manual gift recipient including awarded command failed: admin_id=%d gift_id=%d event_id=%d stage=find_candidates error=%v", cmd.AdminID, gift.ID, gift.EventID, err)
		return nil, fmt.Errorf("find random manual gift recipient candidates including awarded participants: %w", err)
	}
	if len(candidateIDs) == 0 {
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=no_eligible_participants", cmd.AdminID, gift.ID, gift.EventID)
		return nil, ErrManualGiftNoEligibleParticipants
	}

	index, err := h.randomIndex(len(candidateIDs))
	if err != nil {
		log.Printf("ERROR admin random manual gift recipient including awarded command failed: admin_id=%d gift_id=%d event_id=%d stage=choose_random_candidate error=%v", cmd.AdminID, gift.ID, gift.EventID, err)
		return nil, fmt.Errorf("choose random manual gift recipient including awarded participants: %w", err)
	}
	if index < 0 || index >= len(candidateIDs) {
		log.Printf("ERROR admin random manual gift recipient including awarded command failed: admin_id=%d gift_id=%d event_id=%d stage=choose_random_candidate invalid_index=%d candidate_count=%d", cmd.AdminID, gift.ID, gift.EventID, index, len(candidateIDs))
		return nil, fmt.Errorf("choose random manual gift recipient including awarded participants: invalid random index %d", index)
	}

	recipientID := candidateIDs[index]
	if err := h.validateRecipient(ctx, gift.EventID, recipientID, cmd.AdminID, gift.ID); err != nil {
		return nil, err
	}
	if err := h.recipientWriter.AssignRandomManualRecipientIncludingAwarded(ctx, gift.ID, recipientID); err != nil {
		return nil, mapAdminRandomManualGiftRecipientIncludingAwardedRepositoryError(gift, recipientID, cmd.AdminID, err)
	}

	log.Printf("INFO admin random manual gift recipient including awarded command completed: admin_id=%d gift_id=%d event_id=%d candidate_count=%d recipient_participant_id=%d", cmd.AdminID, gift.ID, gift.EventID, len(candidateIDs), recipientID)
	return &AssignRandomAdminGiftRecipientIncludingAwardedResult{
		GiftID:                 gift.ID,
		EventID:                gift.EventID,
		RecipientParticipantID: recipientID,
	}, nil
}

func (h *AssignRandomAdminGiftRecipientIncludingAwardedHandler) validateRecipient(
	ctx context.Context,
	eventID uint,
	recipientID uint,
	adminID uint,
	giftID uint,
) error {
	recipient, err := h.participantRepo.FindByID(ctx, recipientID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_not_found", adminID, giftID, eventID, recipientID)
			return ErrManualGiftRecipientNotFound
		}
		log.Printf("ERROR admin random manual gift recipient including awarded command failed: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d stage=find_recipient error=%v", adminID, giftID, eventID, recipientID, err)
		return fmt.Errorf("find random manual gift recipient including awarded participants: %w", err)
	}
	if recipient.EventID != eventID {
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_event_mismatch", adminID, giftID, eventID, recipientID)
		return ErrManualGiftRecipientEvent
	}
	if !recipient.IsEligibleForManualGift() {
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_ineligible", adminID, giftID, eventID, recipientID)
		return ErrManualGiftRecipientIneligible
	}
	return nil
}

func mapAdminRandomManualGiftRecipientIncludingAwardedRepositoryError(
	gift *entity.Gift,
	recipientID uint,
	adminID uint,
	err error,
) error {
	switch {
	case errors.Is(err, repository.ErrGiftNotFound):
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=gift_not_found_at_write", adminID, gift.ID, gift.EventID)
		return ErrGiftNotFound
	case errors.Is(err, repository.ErrRandomGiftRecipientGiftNotApproved):
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=gift_not_approved_at_write", adminID, gift.ID, gift.EventID)
		return ErrAdminRandomGiftNotApproved
	case errors.Is(err, repository.ErrManualDistributionDisabled):
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=manual_distribution_disabled_at_write", adminID, gift.ID, gift.EventID)
		return ErrManualGiftNotManual
	case errors.Is(err, repository.ErrRandomGiftRecipientAlreadyAssigned):
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d reason=recipient_already_assigned_at_write", adminID, gift.ID, gift.EventID)
		return ErrAdminRandomGiftAlreadyAssigned
	case errors.Is(err, repository.ErrManualRecipientNotFound):
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_not_found_at_write", adminID, gift.ID, gift.EventID, recipientID)
		return ErrManualGiftRecipientNotFound
	case errors.Is(err, repository.ErrManualRecipientEventMismatch):
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_event_mismatch_at_write", adminID, gift.ID, gift.EventID, recipientID)
		return ErrManualGiftRecipientEvent
	case errors.Is(err, repository.ErrManualRecipientIneligible):
		log.Printf("WARN admin random manual gift recipient including awarded command rejected: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_ineligible_at_write", adminID, gift.ID, gift.EventID, recipientID)
		return ErrManualGiftRecipientIneligible
	default:
		log.Printf("ERROR admin random manual gift recipient including awarded command failed: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d stage=compare_and_set error=%v", adminID, gift.ID, gift.EventID, recipientID, err)
		return fmt.Errorf("assign random manual gift recipient including awarded participants: %w", err)
	}
}
