package command

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/repository"
)

// RemoveResultCriteriaCommand представляет команду удаления критерия из результата
type RemoveResultCriteriaCommand struct {
	ResultID   uint
	CriteriaID uint
}

// RemoveResultCriteriaHandler обрабатывает удаление критерия из результата
type RemoveResultCriteriaHandler struct {
	resultRepo repository.ResultRepository
}

// NewRemoveResultCriteriaHandler создаёт новый handler
func NewRemoveResultCriteriaHandler(
	resultRepo repository.ResultRepository,
) *RemoveResultCriteriaHandler {
	return &RemoveResultCriteriaHandler{
		resultRepo: resultRepo,
	}
}

// Handle выполняет команду удаления критерия из результата
func (h *RemoveResultCriteriaHandler) Handle(ctx context.Context, cmd RemoveResultCriteriaCommand) error {
	// Удаляем связь
	if err := h.resultRepo.RemoveCriteria(ctx, cmd.ResultID, cmd.CriteriaID); err != nil {
		return fmt.Errorf("failed to remove criteria from result: %w", err)
	}

	return nil
}
