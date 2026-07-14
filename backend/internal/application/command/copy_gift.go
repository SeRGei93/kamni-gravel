package command

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

const maxGiftCopiesPerRequest = 100

var (
	ErrInvalidGiftCopySourceID    = errors.New("gift ID must be greater than zero")
	ErrInvalidGiftCopiesCount     = errors.New("gift copies count must be between 1 and 100")
	ErrGiftCopyHasPlaceConstraint = errors.New("gift copy is not allowed for a place-constrained gift")
)

// CopyGiftCommand requests creation of independent copies of an existing gift.
// Copies retain the gift definition, but never retain a manually selected
// recipient: every copied manual gift must be awarded independently.
type CopyGiftCommand struct {
	GiftID      uint
	CopiesCount int
}

// CopyGiftResult describes the completed copy operation.
type CopyGiftResult struct {
	EventID      uint
	ReviewStatus entity.GiftReviewStatus
	CreatedCount int
}

// CopyGiftHandler creates copies through one atomic repository operation.
type CopyGiftHandler struct {
	copyRepo repository.GiftCopyRepository
}

func NewCopyGiftHandler(copyRepo repository.GiftCopyRepository) *CopyGiftHandler {
	return &CopyGiftHandler{copyRepo: copyRepo}
}

func (h *CopyGiftHandler) Handle(ctx context.Context, cmd CopyGiftCommand) (*CopyGiftResult, error) {
	log.Printf("INFO gift copy requested: source_gift_id=%d copies_count=%d", cmd.GiftID, cmd.CopiesCount)

	if cmd.GiftID == 0 {
		log.Printf("WARN gift copy rejected: source_gift_id=%d copies_count=%d reason=invalid_gift_id", cmd.GiftID, cmd.CopiesCount)
		return nil, ErrInvalidGiftCopySourceID
	}
	if cmd.CopiesCount < 1 || cmd.CopiesCount > maxGiftCopiesPerRequest {
		log.Printf("WARN gift copy rejected: source_gift_id=%d copies_count=%d reason=invalid_copies_count", cmd.GiftID, cmd.CopiesCount)
		return nil, ErrInvalidGiftCopiesCount
	}
	if h.copyRepo == nil {
		log.Printf("ERROR gift copy failed: source_gift_id=%d copies_count=%d stage=repository_configuration", cmd.GiftID, cmd.CopiesCount)
		return nil, errors.New("gift copy repository is not configured")
	}

	result, err := h.copyRepo.Copy(ctx, cmd.GiftID, cmd.CopiesCount)
	if err != nil {
		if errors.Is(err, repository.ErrGiftNotFound) {
			log.Printf("WARN gift copy rejected: source_gift_id=%d copies_count=%d reason=gift_not_found", cmd.GiftID, cmd.CopiesCount)
			return nil, ErrGiftNotFound
		}
		if errors.Is(err, repository.ErrGiftCopyHasPlaceConstraint) {
			log.Printf("WARN gift copy rejected: source_gift_id=%d copies_count=%d reason=place_constraint", cmd.GiftID, cmd.CopiesCount)
			return nil, ErrGiftCopyHasPlaceConstraint
		}
		log.Printf("ERROR gift copy failed: source_gift_id=%d copies_count=%d stage=repository_copy error=%v", cmd.GiftID, cmd.CopiesCount, err)
		return nil, fmt.Errorf("copy gift %d: %w", cmd.GiftID, err)
	}

	log.Printf("INFO gift copy completed: source_gift_id=%d event_id=%d review_status=%s created_count=%d", cmd.GiftID, result.EventID, result.ReviewStatus, cmd.CopiesCount)
	return &CopyGiftResult{
		EventID:      result.EventID,
		ReviewStatus: result.ReviewStatus,
		CreatedCount: cmd.CopiesCount,
	}, nil
}
