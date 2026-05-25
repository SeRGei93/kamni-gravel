package query

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var (
	ErrInvalidMiniappGiftGenderFilter   = errors.New("invalid miniapp gift gender filter")
	ErrInvalidMiniappGiftBikeTypeFilter = errors.New("invalid miniapp gift bike type filter")
)

// GetMiniappGiftsQuery представляет запрос каталога подарков для Telegram Mini App.
type GetMiniappGiftsQuery struct {
	EventID  uint
	Gender   string
	BikeType string
}

// GetMiniappGiftsHandler обрабатывает запрос каталога подарков для Telegram Mini App.
type GetMiniappGiftsHandler struct {
	getGiftsHandler *GetGiftsHandler
}

func NewGetMiniappGiftsHandler(
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
) *GetMiniappGiftsHandler {
	return &GetMiniappGiftsHandler{
		getGiftsHandler: NewGetGiftsHandler(giftRepo, criteriaRepo),
	}
}

func (h *GetMiniappGiftsHandler) Handle(ctx context.Context, query GetMiniappGiftsQuery) ([]*entity.Gift, error) {
	genderFilter, err := normalizeMiniappGenderFilter(query.Gender)
	if err != nil {
		return nil, err
	}
	bikeTypeFilter, err := normalizeMiniappBikeTypeFilter(query.BikeType)
	if err != nil {
		return nil, err
	}

	approvedStatus := entity.GiftReviewStatusApproved
	gifts, err := h.getGiftsHandler.Handle(ctx, GetGiftsQuery{
		EventID:      query.EventID,
		ReviewStatus: &approvedStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find miniapp gifts for event %d gender=%s bike_type=%s: %w", query.EventID, genderFilter, bikeTypeFilter, err)
	}

	return filterMiniappGifts(gifts, genderFilter, bikeTypeFilter), nil
}

func filterMiniappGifts(gifts []*entity.Gift, genderFilter, bikeTypeFilter string) []*entity.Gift {
	filtered := make([]*entity.Gift, 0, len(gifts))
	for _, gift := range gifts {
		if !matchesMiniappGenderFilter(gift.GenderFilter, genderFilter) {
			continue
		}
		if !matchesMiniappBikeTypeFilter(gift.BikeTypeFilter, bikeTypeFilter) {
			continue
		}
		filtered = append(filtered, gift)
	}

	return filtered
}

func matchesMiniappGenderFilter(giftFilter, selectedFilter string) bool {
	giftFilter = strings.ToLower(strings.TrimSpace(giftFilter))
	if giftFilter == "" {
		giftFilter = "all"
	}

	if selectedFilter == "all" {
		return giftFilter == "all"
	}

	return giftFilter == "all" || giftFilter == selectedFilter
}

func matchesMiniappBikeTypeFilter(giftFilter, selectedFilter string) bool {
	giftFilter = strings.ToLower(strings.TrimSpace(giftFilter))
	if giftFilter == "" {
		giftFilter = "all"
	}

	return selectedFilter == "all" || giftFilter == "all" || giftFilter == selectedFilter
}

func normalizeMiniappGenderFilter(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = string(valueobject.GenderFilterAll)
	}

	filter, err := valueobject.NewGenderFilter(normalized)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidMiniappGiftGenderFilter, normalized)
	}

	return string(filter), nil
}

func normalizeMiniappBikeTypeFilter(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = string(valueobject.BikeTypeFilterAll)
	}

	filter, err := valueobject.NewBikeTypeFilter(normalized)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrInvalidMiniappGiftBikeTypeFilter, normalized)
	}

	return string(filter), nil
}
