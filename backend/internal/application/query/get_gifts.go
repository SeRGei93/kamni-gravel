package query

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

var ErrInvalidGiftReviewStatusFilter = errors.New("invalid gift review status filter")

// GetGiftsQuery представляет запрос на получение подарков
type GetGiftsQuery struct {
	EventID      uint
	ReviewStatus *entity.GiftReviewStatus
	OwnerUserID  *int64
	SearchQuery  string
	Limit        int // размер страницы; <= 0 — все подарки
	Offset       int // смещение страницы
}

// GetGiftsHandler обрабатывает запрос на получение подарков
type GetGiftsHandler struct {
	giftRepo     repository.GiftRepository
	criteriaRepo repository.CriteriaRepository
}

// NewGetGiftsHandler создаёт новый handler
func NewGetGiftsHandler(
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
) *GetGiftsHandler {
	return &GetGiftsHandler{
		giftRepo:     giftRepo,
		criteriaRepo: criteriaRepo,
	}
}

// Handle выполняет запрос на получение страницы подарков и общего количества
// (с учётом фильтра по статусу). Total — полное количество, не размер страницы.
func (h *GetGiftsHandler) Handle(ctx context.Context, query GetGiftsQuery) ([]*entity.Gift, int, error) {
	if query.ReviewStatus != nil && !query.ReviewStatus.IsValid() {
		return nil, 0, ErrInvalidGiftReviewStatusFilter
	}

	filter := repository.GiftListFilter{
		ReviewStatus: query.ReviewStatus,
		OwnerUserID:  query.OwnerUserID,
		SearchQuery:  strings.TrimSpace(query.SearchQuery),
	}
	gifts, total, err := h.listGifts(ctx, query.EventID, filter, query.Limit, query.Offset)
	if err != nil {
		reviewStatus := ""
		if query.ReviewStatus != nil {
			reviewStatus = query.ReviewStatus.String()
		}
		return nil, 0, fmt.Errorf("failed to find gifts for event %d review_status=%s owner_user_id=%v q=%q: %w", query.EventID, reviewStatus, query.OwnerUserID, filter.SearchQuery, err)
	}

	// Загружаем критерии для каждого подарка
	for _, gift := range gifts {
		criteria, err := h.criteriaRepo.FindByGift(ctx, gift.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get criteria for gift %d: %w", gift.ID, err)
		}
		gift.Criteria = criteria

		// Загружаем прикреплённые файлы
		attachments, err := h.giftRepo.GetAttachments(ctx, gift.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get attachments for gift %d: %w", gift.ID, err)
		}
		gift.Attachments = make([]entity.GiftAttachment, len(attachments))
		for i, a := range attachments {
			gift.Attachments[i] = *a
		}
	}

	return gifts, total, nil
}

func (h *GetGiftsHandler) listGifts(
	ctx context.Context,
	eventID uint,
	filter repository.GiftListFilter,
	limit int,
	offset int,
) ([]*entity.Gift, int, error) {
	if filter.OwnerUserID == nil && filter.SearchQuery == "" {
		return h.giftRepo.ListByEventPaged(ctx, eventID, filter.ReviewStatus, limit, offset)
	}

	if filteredRepo, ok := h.giftRepo.(repository.FilteredGiftListRepository); ok {
		return filteredRepo.ListByEventFilteredPaged(ctx, eventID, filter, limit, offset)
	}

	// Compatibility fallback for focused test doubles that only implement the
	// legacy repository method. Production PostgreSQL storage implements the
	// database-backed filter above, so real list pages retain server pagination.
	gifts, _, err := h.giftRepo.ListByEventPaged(ctx, eventID, filter.ReviewStatus, 0, 0)
	if err != nil {
		return nil, 0, err
	}

	filtered := make([]*entity.Gift, 0, len(gifts))
	for _, gift := range gifts {
		if giftMatchesListFilter(gift, filter) {
			filtered = append(filtered, gift)
		}
	}
	if limit <= 0 || offset >= len(filtered) {
		if limit <= 0 {
			return filtered, len(filtered), nil
		}
		return []*entity.Gift{}, len(filtered), nil
	}
	end := min(offset+limit, len(filtered))
	return filtered[offset:end], len(filtered), nil
}

func giftMatchesListFilter(gift *entity.Gift, filter repository.GiftListFilter) bool {
	if gift == nil || (filter.OwnerUserID != nil && gift.UserID != *filter.OwnerUserID) {
		return false
	}
	if filter.SearchQuery == "" {
		return true
	}

	query := strings.ToLower(filter.SearchQuery)
	owner := ""
	if gift.User != nil {
		owner = strings.Join([]string{gift.User.Username, gift.User.FirstName, gift.User.LastName}, " ")
	}
	return strings.Contains(strings.ToLower(gift.Description), query) || strings.Contains(strings.ToLower(owner), query)
}

// GetGiftByIDQuery представляет запрос на получение подарка по ID
type GetGiftByIDQuery struct {
	GiftID uint
}

// GetGiftByIDHandler обрабатывает запрос на получение подарка по ID
type GetGiftByIDHandler struct {
	giftRepo     repository.GiftRepository
	criteriaRepo repository.CriteriaRepository
}

// NewGetGiftByIDHandler создаёт новый handler
func NewGetGiftByIDHandler(
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
) *GetGiftByIDHandler {
	return &GetGiftByIDHandler{
		giftRepo:     giftRepo,
		criteriaRepo: criteriaRepo,
	}
}

// Handle выполняет запрос на получение подарка по ID
func (h *GetGiftByIDHandler) Handle(ctx context.Context, query GetGiftByIDQuery) (*entity.Gift, error) {
	gift, err := h.giftRepo.FindByID(ctx, query.GiftID)
	if err != nil {
		return nil, fmt.Errorf("failed to find gift: %w", err)
	}

	// Загружаем прикреплённые файлы и критерии
	if gift != nil {
		attachments, err := h.giftRepo.GetAttachments(ctx, gift.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get attachments: %w", err)
		}
		gift.Attachments = make([]entity.GiftAttachment, len(attachments))
		for i, a := range attachments {
			gift.Attachments[i] = *a
		}

		// Загружаем критерии
		criteria, err := h.criteriaRepo.FindByGift(ctx, gift.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get criteria: %w", err)
		}
		gift.Criteria = criteria
	}

	return gift, nil
}
