package query

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetGiftsQuery представляет запрос на получение подарков
type GetGiftsQuery struct {
	EventID uint
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

// Handle выполняет запрос на получение подарков
func (h *GetGiftsHandler) Handle(ctx context.Context, query GetGiftsQuery) ([]*entity.Gift, error) {
	// Получаем все подарки события
	gifts, err := h.giftRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to find gifts: %w", err)
	}

	// Загружаем критерии для каждого подарка
	for _, gift := range gifts {
		criteria, err := h.criteriaRepo.FindByGift(ctx, gift.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get criteria for gift %d: %w", gift.ID, err)
		}
		gift.Criteria = criteria

		// Загружаем прикреплённые файлы
		attachments, err := h.giftRepo.GetAttachments(ctx, gift.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get attachments for gift %d: %w", gift.ID, err)
		}
		gift.Attachments = make([]entity.GiftAttachment, len(attachments))
		for i, a := range attachments {
			gift.Attachments[i] = *a
		}
	}

	return gifts, nil
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
