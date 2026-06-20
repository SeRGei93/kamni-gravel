package query

import (
	"context"
	"errors"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

var ErrInvalidGiftReviewStatusFilter = errors.New("invalid gift review status filter")

// GetGiftsQuery представляет запрос на получение подарков
type GetGiftsQuery struct {
	EventID      uint
	ReviewStatus *entity.GiftReviewStatus
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

	gifts, total, err := h.giftRepo.ListByEventPaged(ctx, query.EventID, query.ReviewStatus, query.Limit, query.Offset)
	if err != nil {
		reviewStatus := ""
		if query.ReviewStatus != nil {
			reviewStatus = query.ReviewStatus.String()
		}
		return nil, 0, fmt.Errorf("failed to find gifts for event %d review_status=%s: %w", query.EventID, reviewStatus, err)
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
