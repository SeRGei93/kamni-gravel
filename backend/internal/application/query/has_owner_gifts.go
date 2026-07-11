package query

import (
	"context"
	"fmt"
	"log"

	"gravel_bot/internal/domain/repository"
)

// HasOwnerGiftsQuery requests whether an owner has created gifts for one event.
type HasOwnerGiftsQuery struct {
	OwnerTelegramUserID int64
	EventID             uint
}

// HasOwnerGiftsHandler answers the Mini App navigation visibility check.
type HasOwnerGiftsHandler struct {
	giftRepo repository.ManualGiftRepository
}

// NewHasOwnerGiftsHandler creates a handler for owner gift presence checks.
func NewHasOwnerGiftsHandler(giftRepo repository.ManualGiftRepository) *HasOwnerGiftsHandler {
	return &HasOwnerGiftsHandler{giftRepo: giftRepo}
}

// Handle reports whether the verified owner has at least one gift for the event.
func (h *HasOwnerGiftsHandler) Handle(ctx context.Context, query HasOwnerGiftsQuery) (bool, error) {
	hasGifts, err := h.giftRepo.HasByUserAndEvent(ctx, query.OwnerTelegramUserID, query.EventID)
	if err != nil {
		return false, fmt.Errorf("check owner gifts for user %d event %d: %w", query.OwnerTelegramUserID, query.EventID, err)
	}

	log.Printf("DEBUG owner gifts presence resolved: owner_user_id=%d event_id=%d has_gifts=%t", query.OwnerTelegramUserID, query.EventID, hasGifts)
	return hasGifts, nil
}
