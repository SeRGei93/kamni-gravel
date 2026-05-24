package command

import (
	"context"
	"errors"
	"fmt"

	"gravel_bot/internal/domain/repository"
)

var (
	ErrResultCriteriaAlreadyAdded = errors.New("criteria already added to result")
	ErrResultNotFound             = errors.New("result not found")
)

// AddResultCriteriaCommand представляет команду добавления критерия к результату
type AddResultCriteriaCommand struct {
	ResultID   uint
	CriteriaID uint
}

// AddResultCriteriaHandler обрабатывает добавление критерия к результату
type AddResultCriteriaHandler struct {
	resultRepo   repository.ResultRepository
	criteriaRepo repository.CriteriaRepository
}

// NewAddResultCriteriaHandler создаёт новый handler
func NewAddResultCriteriaHandler(
	resultRepo repository.ResultRepository,
	criteriaRepo repository.CriteriaRepository,
) *AddResultCriteriaHandler {
	return &AddResultCriteriaHandler{
		resultRepo:   resultRepo,
		criteriaRepo: criteriaRepo,
	}
}

// Handle выполняет команду добавления критерия к результату
func (h *AddResultCriteriaHandler) Handle(ctx context.Context, cmd AddResultCriteriaCommand) error {
	// Проверяем существование результата
	_, err := h.resultRepo.FindByID(ctx, cmd.ResultID)
	if err != nil {
		return ErrResultNotFound
	}

	// Проверяем существование критерия
	_, err = h.criteriaRepo.FindByID(ctx, cmd.CriteriaID)
	if err != nil {
		return fmt.Errorf("criteria not found: %w", err)
	}

	// Добавляем связь
	if err := h.resultRepo.AddCriteria(ctx, cmd.ResultID, cmd.CriteriaID); err != nil {
		return fmt.Errorf("failed to add criteria to result: %w", err)
	}

	return nil
}
