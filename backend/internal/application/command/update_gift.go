package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var (
	ErrInvalidGiftGenderFilter       = errors.New("invalid gift gender filter")
	ErrInvalidGiftBikeTypeFilter     = errors.New("invalid gift bike type filter")
	ErrInvalidGiftReviewStatus       = errors.New("invalid gift review status")
	ErrInvalidGiftPlace              = errors.New("gift place must be greater than zero")
	ErrInvalidGiftPlaceRule          = errors.New("invalid gift place rule")
	ErrGiftCriteriaPayloadRequired   = errors.New("criteria_ids are required when approving a gift")
	ErrManualGiftRecipientConflict   = errors.New("manual recipient requires manual distribution")
	ErrManualGiftNotManual           = errors.New("gift is not configured for manual distribution")
	ErrManualGiftRecipientNotFound   = errors.New("manual gift recipient participant not found")
	ErrManualGiftRecipientEvent      = errors.New("manual gift recipient belongs to another event")
	ErrManualGiftRecipientIneligible = errors.New("manual gift recipient must have finished or be marked dnf")
)

// UpdateGiftCommand представляет команду административного обновления подарка.
type UpdateGiftCommand struct {
	GiftID         uint
	Description    *string
	GenderFilter   *string
	BikeTypeFilter *string
	ReviewStatus   *string
	Place          *int
	PlaceSet       bool
	PlaceRule      valueobject.GiftPlaceRule
	PlaceRuleSet   bool
	CriteriaIDs    []uint
	CriteriaIDsSet bool
	// ManualDistribution nil means that the administrator did not change the mode.
	ManualDistribution *bool
	// ManualRecipientParticipantIDSet distinguishes omitted from explicit null.
	ManualRecipientParticipantID    *uint
	ManualRecipientParticipantIDSet bool
}

// UpdateGiftResult представляет результат административного обновления подарка.
type UpdateGiftResult struct {
	Gift                 *entity.Gift
	BecameApproved       bool
	PreviousReviewStatus entity.GiftReviewStatus
}

// UpdateGiftHandler обрабатывает административное обновление подарка.
type UpdateGiftHandler struct {
	giftRepo        repository.GiftRepository
	participantRepo repository.ParticipantRepository
}

// NewUpdateGiftHandler создаёт новый handler обновления подарка.
func NewUpdateGiftHandler(giftRepo repository.GiftRepository, participantRepos ...repository.ParticipantRepository) *UpdateGiftHandler {
	var participantRepo repository.ParticipantRepository
	if len(participantRepos) > 0 {
		participantRepo = participantRepos[0]
	}
	return &UpdateGiftHandler{giftRepo: giftRepo, participantRepo: participantRepo}
}

// Handle выполняет команду обновления подарка.
func (h *UpdateGiftHandler) Handle(ctx context.Context, cmd UpdateGiftCommand) (*UpdateGiftResult, error) {
	if cmd.ManualDistribution != nil || cmd.ManualRecipientParticipantIDSet {
		log.Printf(
			"DEBUG manual gift update requested: gift_id=%d manual_distribution_set=%t recipient_set=%t recipient_participant_id=%s",
			cmd.GiftID,
			cmd.ManualDistribution != nil,
			cmd.ManualRecipientParticipantIDSet,
			manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID),
		)
	}
	gift, err := h.giftRepo.FindByID(ctx, cmd.GiftID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGiftNotFound, err)
	}
	previousReviewStatus := gift.ReviewStatus

	if cmd.Description != nil {
		description := strings.TrimSpace(*cmd.Description)
		if description == "" {
			return nil, ErrEmptyDescription
		}
		gift.Description = description
	}

	if cmd.GenderFilter != nil {
		genderFilter, err := valueobject.NewGenderFilter(*cmd.GenderFilter)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidGiftGenderFilter, *cmd.GenderFilter)
		}
		gift.GenderFilter = string(genderFilter)
	}

	if cmd.BikeTypeFilter != nil {
		bikeTypeFilter, err := valueobject.NewBikeTypeFilter(*cmd.BikeTypeFilter)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidGiftBikeTypeFilter, *cmd.BikeTypeFilter)
		}
		gift.BikeTypeFilter = string(bikeTypeFilter)
	}

	if cmd.ReviewStatus != nil {
		reviewStatus, err := entity.NewGiftReviewStatus(*cmd.ReviewStatus)
		if err != nil {
			log.Printf("level=warn msg=\"Invalid gift review status\" gift_id=%d review_status=%s", cmd.GiftID, *cmd.ReviewStatus)
			return nil, fmt.Errorf("%w: %s", ErrInvalidGiftReviewStatus, *cmd.ReviewStatus)
		}
		if reviewStatus == entity.GiftReviewStatusApproved && !cmd.CriteriaIDsSet {
			return nil, ErrGiftCriteriaPayloadRequired
		}
		gift.ReviewStatus = reviewStatus
	}

	if cmd.PlaceRuleSet {
		gift.PlaceRule = cmd.PlaceRule
		gift.Place = gift.PlaceRule.FirstLegacyPlace()
	} else if cmd.PlaceSet {
		if cmd.Place != nil && *cmd.Place <= 0 {
			log.Printf("level=warn msg=\"Invalid legacy gift place\" gift_id=%d reason=non_positive", cmd.GiftID)
			return nil, ErrInvalidGiftPlace
		}
		gift.Place = cmd.Place
		if cmd.Place == nil {
			gift.PlaceRule = valueobject.NewGiftPlaceRuleNone()
		} else {
			placeRule, err := valueobject.NewGiftPlaceRulePlaces([]int{*cmd.Place})
			if err != nil {
				log.Printf("level=warn msg=\"Invalid gift place rule\" gift_id=%d rule_type=places reason=%q", cmd.GiftID, err.Error())
				return nil, fmt.Errorf("%w: %v", ErrInvalidGiftPlaceRule, err)
			}
			gift.PlaceRule = placeRule
		}
	}

	if err := h.applyManualDistribution(ctx, gift, cmd); err != nil {
		return nil, err
	}

	if cmd.CriteriaIDsSet {
		if err := h.giftRepo.UpdateWithCriteria(ctx, gift, cmd.CriteriaIDs); err != nil {
			log.Printf("ERROR gift update failed: gift_id=%d stage=update_with_criteria error=%v", cmd.GiftID, err)
			return nil, fmt.Errorf("failed to update gift %d with criteria: %w", cmd.GiftID, err)
		}
		h.logManualGiftUpdate(gift, cmd)
		return &UpdateGiftResult{
			Gift:                 gift,
			BecameApproved:       previousReviewStatus != entity.GiftReviewStatusApproved && gift.ReviewStatus == entity.GiftReviewStatusApproved,
			PreviousReviewStatus: previousReviewStatus,
		}, nil
	}

	if err := h.giftRepo.Update(ctx, gift); err != nil {
		log.Printf("ERROR gift update failed: gift_id=%d stage=update_fields error=%v", cmd.GiftID, err)
		return nil, fmt.Errorf("failed to update gift %d fields: %w", cmd.GiftID, err)
	}
	h.logManualGiftUpdate(gift, cmd)

	return &UpdateGiftResult{
		Gift:                 gift,
		BecameApproved:       previousReviewStatus != entity.GiftReviewStatusApproved && gift.ReviewStatus == entity.GiftReviewStatusApproved,
		PreviousReviewStatus: previousReviewStatus,
	}, nil
}

func (h *UpdateGiftHandler) applyManualDistribution(ctx context.Context, gift *entity.Gift, cmd UpdateGiftCommand) error {
	if cmd.ManualDistribution != nil {
		if !*cmd.ManualDistribution && cmd.ManualRecipientParticipantIDSet && cmd.ManualRecipientParticipantID != nil {
			log.Printf("WARN manual gift update rejected: gift_id=%d recipient_participant_id=%s reason=manual_distribution_disabled", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID))
			return ErrManualGiftRecipientConflict
		}
		gift.ManualDistribution = *cmd.ManualDistribution
		if !gift.ManualDistribution {
			gift.ManualRecipientParticipantID = nil
			gift.ManualRecipient = nil
		}
	}

	if !cmd.ManualRecipientParticipantIDSet {
		return nil
	}
	if cmd.ManualRecipientParticipantID == nil {
		gift.ManualRecipientParticipantID = nil
		gift.ManualRecipient = nil
		return nil
	}
	if !gift.ManualDistribution {
		log.Printf("WARN manual gift update rejected: gift_id=%d recipient_participant_id=%s reason=manual_distribution_disabled", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID))
		return ErrManualGiftNotManual
	}
	if h.participantRepo == nil {
		log.Printf("ERROR manual gift update failed: gift_id=%d recipient_participant_id=%s stage=participant_repository_not_configured", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID))
		return errors.New("participant repository is required for manual recipient updates")
	}

	recipient, err := h.participantRepo.FindByID(ctx, *cmd.ManualRecipientParticipantID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf("WARN manual gift update rejected: gift_id=%d recipient_participant_id=%s reason=recipient_not_found", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID))
			return ErrManualGiftRecipientNotFound
		}
		log.Printf("ERROR manual gift update failed: gift_id=%d recipient_participant_id=%s stage=find_recipient error=%v", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID), err)
		return fmt.Errorf("find manual gift recipient: %w", err)
	}
	if recipient.EventID != gift.EventID {
		log.Printf("WARN manual gift update rejected: gift_id=%d recipient_participant_id=%s reason=recipient_event_mismatch", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID))
		return ErrManualGiftRecipientEvent
	}
	if !recipient.IsEligibleForManualGift() {
		log.Printf("WARN [FIX:manual-recipient-eligibility] manual gift update rejected: gift_id=%d recipient_participant_id=%s status=%s has_result=%t reason=recipient_ineligible", cmd.GiftID, manualGiftRecipientIDLogValue(cmd.ManualRecipientParticipantID), recipient.Status, recipient.IsFinished())
		return ErrManualGiftRecipientIneligible
	}

	gift.ManualRecipientParticipantID = cmd.ManualRecipientParticipantID
	gift.ManualRecipient = recipient
	return nil
}

func (h *UpdateGiftHandler) logManualGiftUpdate(gift *entity.Gift, cmd UpdateGiftCommand) {
	if cmd.ManualDistribution == nil && !cmd.ManualRecipientParticipantIDSet {
		return
	}
	log.Printf(
		"INFO manual gift configuration updated: actor_type=admin gift_id=%d manual_distribution=%t recipient_participant_id=%s",
		gift.ID,
		gift.ManualDistribution,
		manualGiftRecipientIDLogValue(gift.ManualRecipientParticipantID),
	)
}

func manualGiftRecipientIDLogValue(recipientParticipantID *uint) string {
	if recipientParticipantID == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *recipientParticipantID)
}
