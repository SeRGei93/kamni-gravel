package command

import (
	"context"
	"errors"
	"fmt"
	
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var (
	ErrCriteriaNotFound = errors.New("criteria not found")
)

// UpdateCriteriaCommand представляет команду обновления критерия
type UpdateCriteriaCommand struct {
	CriteriaID   uint
	Name         *string
	Description  *string
	CriteriaType *string
}

// UpdateCriteriaHandler обрабатывает обновление критерия
type UpdateCriteriaHandler struct {
	criteriaRepo repository.CriteriaRepository
}

// NewUpdateCriteriaHandler создаёт новый handler
func NewUpdateCriteriaHandler(
	criteriaRepo repository.CriteriaRepository,
) *UpdateCriteriaHandler {
	return &UpdateCriteriaHandler{
		criteriaRepo: criteriaRepo,
	}
}

// Handle выполняет команду обновления критерия
func (h *UpdateCriteriaHandler) Handle(ctx context.Context, cmd UpdateCriteriaCommand) (*entity.Criteria, error) {
	// Находим критерий
	criteria, err := h.criteriaRepo.FindByID(ctx, cmd.CriteriaID)
	if err != nil {
		return nil, ErrCriteriaNotFound
	}
	
	// Обновляем поля
	if cmd.Name != nil {
		if *cmd.Name == "" {
			return nil, ErrCriteriaNameRequired
		}
		criteria.Name = *cmd.Name
	}
	
	if cmd.Description != nil {
		criteria.Description = *cmd.Description
	}
	
	if cmd.CriteriaType != nil {
		criteriaType, err := valueobject.NewCriteriaType(*cmd.CriteriaType)
		if err != nil {
			return nil, ErrInvalidCriteriaType
		}
		criteria.CriteriaType = criteriaType
	}
	
	// Сохраняем изменения
	if err := h.criteriaRepo.Update(ctx, criteria); err != nil {
		return nil, fmt.Errorf("failed to update criteria: %w", err)
	}
	
	return criteria, nil
}
