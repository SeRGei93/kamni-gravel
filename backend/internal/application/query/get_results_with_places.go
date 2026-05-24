package query

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/repository"
)

// GetResultsWithPlacesQuery представляет запрос на получение результатов с местами
type GetResultsWithPlacesQuery struct {
	EventID uint
}

// GetResultsWithPlacesHandler обрабатывает запрос на получение результатов с местами
type GetResultsWithPlacesHandler struct {
	resultRepo repository.ResultRepository
}

// NewGetResultsWithPlacesHandler создаёт новый handler
func NewGetResultsWithPlacesHandler(
	resultRepo repository.ResultRepository,
) *GetResultsWithPlacesHandler {
	return &GetResultsWithPlacesHandler{
		resultRepo: resultRepo,
	}
}

// Handle выполняет запрос на получение результатов с местами
func (h *GetResultsWithPlacesHandler) Handle(ctx context.Context, query GetResultsWithPlacesQuery) ([]*repository.ResultWithPlace, error) {
	results, err := h.resultRepo.FindByEventWithPlaces(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results with places: %w", err)
	}

	return results, nil
}
