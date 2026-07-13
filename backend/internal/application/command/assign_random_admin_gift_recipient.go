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

var (
	// ErrAdminRandomGiftNotApproved means only approved gifts can be distributed.
	ErrAdminRandomGiftNotApproved = errors.New("gift must be approved before random distribution")
	// ErrAdminRandomGiftAlreadyAssigned means the gift already has an automatic
	// or manual recipient, or another request won the compare-and-set race.
	ErrAdminRandomGiftAlreadyAssigned = errors.New("gift is already distributed")
)

// AssignRandomAdminGiftRecipientCommand requests an administrator-driven
// random recipient assignment for one gift.
type AssignRandomAdminGiftRecipientCommand struct {
	GiftID uint
}

// AssignRandomAdminGiftRecipientResult describes the completed assignment.
type AssignRandomAdminGiftRecipientResult struct {
	GiftID                 uint
	EventID                uint
	RecipientParticipantID uint
	BecameManual           bool
}

type eligibleUnawardedParticipantIDsReader interface {
	Handle(ctx context.Context, query query.GetEligibleUnawardedParticipantIDsQuery) ([]uint, error)
}

type adminPrizeDistributionReader interface {
	Handle(ctx context.Context, query query.GetPrizeDistributionQuery) ([]*query.PrizeDistributionResult, error)
}

// AssignRandomAdminGiftRecipientHandler protects the administrator workflow
// while deliberately avoiding Mini App ownership and DTO dependencies.
type AssignRandomAdminGiftRecipientHandler struct {
	giftRepo                repository.GiftRepository
	recipientWriter         repository.RandomManualGiftRecipientRepository
	participantRepo         repository.ParticipantRepository
	candidateIDsReader      eligibleUnawardedParticipantIDsReader
	prizeDistributionReader adminPrizeDistributionReader
	randomIndex             func(max int) (int, error)
}

// NewAssignRandomAdminGiftRecipientHandler creates an administrator random
// recipient handler backed by crypto/rand.
func NewAssignRandomAdminGiftRecipientHandler(
	giftRepo repository.GiftRepository,
	recipientWriter repository.RandomManualGiftRecipientRepository,
	participantRepo repository.ParticipantRepository,
	candidateIDsReader eligibleUnawardedParticipantIDsReader,
	prizeDistributionReader adminPrizeDistributionReader,
) *AssignRandomAdminGiftRecipientHandler {
	return newAssignRandomAdminGiftRecipientHandler(
		giftRepo,
		recipientWriter,
		participantRepo,
		candidateIDsReader,
		prizeDistributionReader,
		cryptoRandomIndex,
	)
}

func newAssignRandomAdminGiftRecipientHandler(
	giftRepo repository.GiftRepository,
	recipientWriter repository.RandomManualGiftRecipientRepository,
	participantRepo repository.ParticipantRepository,
	candidateIDsReader eligibleUnawardedParticipantIDsReader,
	prizeDistributionReader adminPrizeDistributionReader,
	randomIndex func(max int) (int, error),
) *AssignRandomAdminGiftRecipientHandler {
	return &AssignRandomAdminGiftRecipientHandler{
		giftRepo:                giftRepo,
		recipientWriter:         recipientWriter,
		participantRepo:         participantRepo,
		candidateIDsReader:      candidateIDsReader,
		prizeDistributionReader: prizeDistributionReader,
		randomIndex:             randomIndex,
	}
}

// Handle selects an unawarded participant and atomically makes the gift a
// manually distributed gift with that recipient.
func (h *AssignRandomAdminGiftRecipientHandler) Handle(
	ctx context.Context,
	cmd AssignRandomAdminGiftRecipientCommand,
) (*AssignRandomAdminGiftRecipientResult, error) {
	if h.recipientWriter == nil || h.candidateIDsReader == nil || h.prizeDistributionReader == nil || h.participantRepo == nil {
		log.Printf("ERROR admin random gift recipient command unavailable: gift_id=%d", cmd.GiftID)
		return nil, errors.New("admin random gift recipient dependencies are not configured")
	}

	log.Printf("DEBUG admin random gift recipient command started: gift_id=%d", cmd.GiftID)
	gift, err := h.giftRepo.FindByID(ctx, cmd.GiftID)
	if err != nil {
		if errors.Is(err, repository.ErrGiftNotFound) {
			log.Printf("WARN admin random gift recipient command rejected: gift_id=%d reason=gift_not_found", cmd.GiftID)
			return nil, ErrGiftNotFound
		}
		log.Printf("ERROR admin random gift recipient command failed: gift_id=%d stage=find_gift error=%v", cmd.GiftID, err)
		return nil, fmt.Errorf("find gift for random recipient: %w", err)
	}
	if gift.ReviewStatus != entity.GiftReviewStatusApproved {
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d review_status=%s reason=gift_not_approved", gift.ID, gift.EventID, gift.ReviewStatus)
		return nil, ErrAdminRandomGiftNotApproved
	}
	if gift.ManualRecipientParticipantID != nil {
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d reason=manual_recipient_already_assigned", gift.ID, gift.EventID)
		return nil, ErrAdminRandomGiftAlreadyAssigned
	}

	becameManual := !gift.ManualDistribution
	if becameManual {
		distribution, err := h.prizeDistributionReader.Handle(ctx, query.GetPrizeDistributionQuery{EventID: gift.EventID})
		if err != nil {
			log.Printf("ERROR admin random gift recipient command failed: gift_id=%d event_id=%d stage=get_prize_distribution error=%v", gift.ID, gift.EventID, err)
			return nil, fmt.Errorf("get automatic prize distribution: %w", err)
		}
		if automaticGiftIsAssigned(gift.ID, distribution) {
			log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d reason=automatic_recipient_already_assigned", gift.ID, gift.EventID)
			return nil, ErrAdminRandomGiftAlreadyAssigned
		}
	}

	candidateIDs, err := h.candidateIDsReader.Handle(ctx, query.GetEligibleUnawardedParticipantIDsQuery{EventID: gift.EventID})
	if err != nil {
		log.Printf("ERROR admin random gift recipient command failed: gift_id=%d event_id=%d stage=get_candidates error=%v", gift.ID, gift.EventID, err)
		return nil, fmt.Errorf("find random gift recipient candidates: %w", err)
	}
	if len(candidateIDs) == 0 {
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d reason=no_unawarded_participants", gift.ID, gift.EventID)
		return nil, ErrManualGiftNoUnawardedParticipants
	}

	index, err := h.randomIndex(len(candidateIDs))
	if err != nil {
		log.Printf("ERROR admin random gift recipient command failed: gift_id=%d event_id=%d stage=choose_random_candidate error=%v", gift.ID, gift.EventID, err)
		return nil, fmt.Errorf("choose random gift recipient: %w", err)
	}
	if index < 0 || index >= len(candidateIDs) {
		log.Printf("ERROR admin random gift recipient command failed: gift_id=%d event_id=%d stage=choose_random_candidate invalid_index=%d candidate_count=%d", gift.ID, gift.EventID, index, len(candidateIDs))
		return nil, fmt.Errorf("choose random gift recipient: invalid random index %d", index)
	}

	recipientID := candidateIDs[index]
	if err := h.validateRecipient(ctx, gift.EventID, recipientID); err != nil {
		return nil, err
	}
	if err := h.recipientWriter.AssignRandomManualRecipient(ctx, gift.ID, recipientID); err != nil {
		return nil, mapAdminRandomGiftRecipientRepositoryError(gift, recipientID, err)
	}

	log.Printf("INFO admin random gift recipient command completed: gift_id=%d event_id=%d candidate_count=%d recipient_participant_id=%d converted_to_manual=%t", gift.ID, gift.EventID, len(candidateIDs), recipientID, becameManual)
	return &AssignRandomAdminGiftRecipientResult{
		GiftID:                 gift.ID,
		EventID:                gift.EventID,
		RecipientParticipantID: recipientID,
		BecameManual:           becameManual,
	}, nil
}

func (h *AssignRandomAdminGiftRecipientHandler) validateRecipient(ctx context.Context, eventID, recipientID uint) error {
	recipient, err := h.participantRepo.FindByID(ctx, recipientID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf("WARN admin random gift recipient command rejected: event_id=%d recipient_participant_id=%d reason=recipient_not_found", eventID, recipientID)
			return ErrManualGiftRecipientNotFound
		}
		log.Printf("ERROR admin random gift recipient command failed: event_id=%d recipient_participant_id=%d stage=find_recipient error=%v", eventID, recipientID, err)
		return fmt.Errorf("find random gift recipient: %w", err)
	}
	if recipient.EventID != eventID {
		log.Printf("WARN admin random gift recipient command rejected: event_id=%d recipient_participant_id=%d reason=recipient_event_mismatch", eventID, recipientID)
		return ErrManualGiftRecipientEvent
	}
	if !recipient.IsEligibleForManualGift() {
		log.Printf("WARN [FIX:manual-recipient-eligibility] admin random gift recipient command rejected: event_id=%d recipient_participant_id=%d status=%s has_result=%t reason=recipient_ineligible", eventID, recipientID, recipient.Status, recipient.IsFinished())
		return ErrManualGiftRecipientIneligible
	}
	return nil
}

func automaticGiftIsAssigned(giftID uint, distribution []*query.PrizeDistributionResult) bool {
	for _, participant := range distribution {
		if participant == nil {
			continue
		}
		for _, assignment := range participant.MatchedGiftAssignments {
			if assignment != nil && assignment.Gift != nil && assignment.Gift.ID == giftID {
				return true
			}
		}
		for _, gift := range participant.MatchedGifts {
			if gift != nil && gift.ID == giftID {
				return true
			}
		}
	}
	return false
}

func mapAdminRandomGiftRecipientRepositoryError(gift *entity.Gift, recipientID uint, err error) error {
	switch {
	case errors.Is(err, repository.ErrGiftNotFound):
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d reason=gift_not_found_at_write", gift.ID, gift.EventID)
		return ErrGiftNotFound
	case errors.Is(err, repository.ErrRandomGiftRecipientGiftNotApproved):
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d reason=gift_not_approved_at_write", gift.ID, gift.EventID)
		return ErrAdminRandomGiftNotApproved
	case errors.Is(err, repository.ErrRandomGiftRecipientAlreadyAssigned):
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d reason=recipient_already_assigned_at_write", gift.ID, gift.EventID)
		return ErrAdminRandomGiftAlreadyAssigned
	case errors.Is(err, repository.ErrManualRecipientNotFound):
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_not_found_at_write", gift.ID, gift.EventID, recipientID)
		return ErrManualGiftRecipientNotFound
	case errors.Is(err, repository.ErrManualRecipientEventMismatch):
		log.Printf("WARN admin random gift recipient command rejected: gift_id=%d event_id=%d recipient_participant_id=%d reason=recipient_event_mismatch_at_write", gift.ID, gift.EventID, recipientID)
		return ErrManualGiftRecipientEvent
	default:
		log.Printf("ERROR admin random gift recipient command failed: gift_id=%d event_id=%d recipient_participant_id=%d stage=compare_and_set error=%v", gift.ID, gift.EventID, recipientID, err)
		return fmt.Errorf("assign random gift recipient: %w", err)
	}
}
