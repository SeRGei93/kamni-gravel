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
	Limit        int     // размер страницы
	Offset       int     // смещение страницы
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

// Handle выполняет запрос на получение страницы критериев и общего количества.
func (h *GetCriteriaHandler) Handle(ctx context.Context, query GetCriteriaQuery) ([]*entity.Criteria, int, error) {
	var typeFilter *valueobject.CriteriaType
	if query.CriteriaType != nil {
		criteriaType, err := valueobject.NewCriteriaType(*query.CriteriaType)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid criteria type: %w", err)
		}
		typeFilter = &criteriaType
	}

	criteria, total, err := h.criteriaRepo.ListPaged(ctx, typeFilter, query.Limit, query.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list criteria: %w", err)
	}

	return criteria, total, nil
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
