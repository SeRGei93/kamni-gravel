package query

import (
	"context"
	"fmt"
	
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// GetCriteriaQuery представляет запрос на получение критериев
type GetCriteriaQuery struct {
	CriteriaType *string // фильтр по типу (опционально)
}

// GetCriteriaHandler обрабатывает запрос на получение критериев
type GetCriteriaHandler struct {
	criteriaRepo repository.CriteriaRepository
}

// NewGetCriteriaHandler создаёт новый handler
func NewGetCriteriaHandler(
	criteriaRepo repository.CriteriaRepository,
) *GetCriteriaHandler {
	return &GetCriteriaHandler{
		criteriaRepo: criteriaRepo,
	}
}

// Handle выполняет запрос на получение критериев
func (h *GetCriteriaHandler) Handle(ctx context.Context, query GetCriteriaQuery) ([]*entity.Criteria, error) {
	// Если указан фильтр по типу
	if query.CriteriaType != nil {
		criteriaType, err := valueobject.NewCriteriaType(*query.CriteriaType)
		if err != nil {
			return nil, fmt.Errorf("invalid criteria type: %w", err)
		}
		
		criteria, err := h.criteriaRepo.FindByType(ctx, criteriaType)
		if err != nil {
			return nil, fmt.Errorf("failed to find criteria by type: %w", err)
		}
		return criteria, nil
	}
	
	// Иначе возвращаем все критерии
	criteria, err := h.criteriaRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find all criteria: %w", err)
	}
	
	return criteria, nil
}

// GetCriteriaByIDQuery представляет запрос на получение критерия по ID
type GetCriteriaByIDQuery struct {
	CriteriaID uint
}

// GetCriteriaByIDHandler обрабатывает запрос на получение критерия по ID
type GetCriteriaByIDHandler struct {
	criteriaRepo repository.CriteriaRepository
}

// NewGetCriteriaByIDHandler создаёт новый handler
func NewGetCriteriaByIDHandler(
	criteriaRepo repository.CriteriaRepository,
) *GetCriteriaByIDHandler {
	return &GetCriteriaByIDHandler{
		criteriaRepo: criteriaRepo,
	}
}

// Handle выполняет запрос на получение критерия по ID
func (h *GetCriteriaByIDHandler) Handle(ctx context.Context, query GetCriteriaByIDQuery) (*entity.Criteria, error) {
	criteria, err := h.criteriaRepo.FindByID(ctx, query.CriteriaID)
	if err != nil {
		return nil, fmt.Errorf("failed to find criteria: %w", err)
	}
	
	return criteria, nil
}
