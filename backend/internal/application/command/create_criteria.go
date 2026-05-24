package command

import (
	"context"
	"errors"
	"fmt"
	"time"
	
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var (
	ErrCriteriaNameRequired = errors.New("criteria name is required")
	ErrInvalidCriteriaType  = errors.New("invalid criteria type")
)

// CreateCriteriaCommand представляет команду создания критерия
type CreateCriteriaCommand struct {
	Name         string
	Description  string
	CriteriaType string
}

// CreateCriteriaHandler обрабатывает создание критерия
type CreateCriteriaHandler struct {
	criteriaRepo repository.CriteriaRepository
}

// NewCreateCriteriaHandler создаёт новый handler
func NewCreateCriteriaHandler(
	criteriaRepo repository.CriteriaRepository,
) *CreateCriteriaHandler {
	return &CreateCriteriaHandler{
		criteriaRepo: criteriaRepo,
	}
}

// Handle выполняет команду создания критерия
func (h *CreateCriteriaHandler) Handle(ctx context.Context, cmd CreateCriteriaCommand) (*entity.Criteria, error) {
	// Валидация
	if cmd.Name == "" {
		return nil, ErrCriteriaNameRequired
	}
	
	// Валидация типа критерия
	criteriaType, err := valueobject.NewCriteriaType(cmd.CriteriaType)
	if err != nil {
		return nil, ErrInvalidCriteriaType
	}
	
	// Создаём критерий
	criteria := &entity.Criteria{
		Name:         cmd.Name,
		Description:  cmd.Description,
		CriteriaType: criteriaType,
		CreatedAt:    time.Now(),
	}
	
	// Сохраняем в БД
	if err := h.criteriaRepo.Create(ctx, criteria); err != nil {
		return nil, fmt.Errorf("failed to create criteria: %w", err)
	}
	
	return criteria, nil
}
