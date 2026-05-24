package command

import (
	"context"
	"errors"
	"fmt"
	
	"gravel_bot/internal/domain/repository"
)

var (
	ErrCriteriaAlreadyAdded = errors.New("criteria already added to gift")
)

// AddCriteriaToGiftCommand представляет команду добавления критерия к подарку
type AddCriteriaToGiftCommand struct {
	GiftID     uint
	CriteriaID uint
}

// AddCriteriaToGiftHandler обрабатывает добавление критерия к подарку
type AddCriteriaToGiftHandler struct {
	giftRepo         repository.GiftRepository
	criteriaRepo     repository.CriteriaRepository
	giftCriteriaRepo repository.GiftCriteriaRepository
}

// NewAddCriteriaToGiftHandler создаёт новый handler
func NewAddCriteriaToGiftHandler(
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
	giftCriteriaRepo repository.GiftCriteriaRepository,
) *AddCriteriaToGiftHandler {
	return &AddCriteriaToGiftHandler{
		giftRepo:         giftRepo,
		criteriaRepo:     criteriaRepo,
		giftCriteriaRepo: giftCriteriaRepo,
	}
}

// Handle выполняет команду добавления критерия к подарку
func (h *AddCriteriaToGiftHandler) Handle(ctx context.Context, cmd AddCriteriaToGiftCommand) error {
	// Проверяем существование подарка
	_, err := h.giftRepo.FindByID(ctx, cmd.GiftID)
	if err != nil {
		return ErrGiftNotFound
	}
	
	// Проверяем существование критерия
	_, err = h.criteriaRepo.FindByID(ctx, cmd.CriteriaID)
	if err != nil {
		return ErrCriteriaNotFound
	}
	
	// Добавляем связь
	if err := h.giftCriteriaRepo.AddCriteriaToGift(ctx, cmd.GiftID, cmd.CriteriaID); err != nil {
		return fmt.Errorf("failed to add criteria to gift: %w", err)
	}
	
	return nil
}
