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
	ErrInvalidGiftGenderFilter     = errors.New("invalid gift gender filter")
	ErrInvalidGiftBikeTypeFilter   = errors.New("invalid gift bike type filter")
	ErrInvalidGiftReviewStatus     = errors.New("invalid gift review status")
	ErrInvalidGiftPlace            = errors.New("gift place must be greater than zero")
	ErrGiftCriteriaPayloadRequired = errors.New("criteria_ids are required when approving a gift")
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
	CriteriaIDs    []uint
	CriteriaIDsSet bool
}

// UpdateGiftHandler обрабатывает административное обновление подарка.
type UpdateGiftHandler struct {
	giftRepo repository.GiftRepository
}

// NewUpdateGiftHandler создаёт новый handler обновления подарка.
func NewUpdateGiftHandler(giftRepo repository.GiftRepository) *UpdateGiftHandler {
	return &UpdateGiftHandler{giftRepo: giftRepo}
}

// Handle выполняет команду обновления подарка.
func (h *UpdateGiftHandler) Handle(ctx context.Context, cmd UpdateGiftCommand) (*entity.Gift, error) {
	gift, err := h.giftRepo.FindByID(ctx, cmd.GiftID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGiftNotFound, err)
	}

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

	if cmd.PlaceSet {
		if cmd.Place != nil && *cmd.Place <= 0 {
			return nil, ErrInvalidGiftPlace
		}
		gift.Place = cmd.Place
	}

	if cmd.CriteriaIDsSet {
		if err := h.giftRepo.UpdateWithCriteria(ctx, gift, cmd.CriteriaIDs); err != nil {
			log.Printf("Gift update failed: gift_id=%d stage=update_with_criteria error=%v", cmd.GiftID, err)
			return nil, fmt.Errorf("failed to update gift %d with criteria: %w", cmd.GiftID, err)
		}
		return gift, nil
	}

	if err := h.giftRepo.Update(ctx, gift); err != nil {
		log.Printf("Gift update failed: gift_id=%d stage=update_fields error=%v", cmd.GiftID, err)
		return nil, fmt.Errorf("failed to update gift %d fields: %w", cmd.GiftID, err)
	}

	return gift, nil
}
