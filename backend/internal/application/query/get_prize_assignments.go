package query

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetPrizeAssignmentsQuery представляет запрос на получение назначений призов
type GetPrizeAssignmentsQuery struct {
	EventID      *uint // фильтр по событию (опционально)
	ParticipantID *uint // фильтр по участнику (опционально)
}

// GetPrizeAssignmentsHandler обрабатывает запрос на получение назначений призов
type GetPrizeAssignmentsHandler struct {
	prizeAssignmentRepo repository.PrizeAssignmentRepository
}

// NewGetPrizeAssignmentsHandler создаёт новый handler
func NewGetPrizeAssignmentsHandler(
	prizeAssignmentRepo repository.PrizeAssignmentRepository,
) *GetPrizeAssignmentsHandler {
	return &GetPrizeAssignmentsHandler{
		prizeAssignmentRepo: prizeAssignmentRepo,
	}
}

// Handle выполняет запрос на получение назначений призов
func (h *GetPrizeAssignmentsHandler) Handle(ctx context.Context, query GetPrizeAssignmentsQuery) ([]*entity.PrizeAssignment, error) {
	if query.ParticipantID != nil {
		// Получаем призы конкретного участника
		assignments, err := h.prizeAssignmentRepo.FindByParticipant(ctx, *query.ParticipantID)
		if err != nil {
			return nil, fmt.Errorf("failed to find prize assignments by participant: %w", err)
		}
		return assignments, nil
	}

	if query.EventID != nil {
		// Получаем все призы события
		assignments, err := h.prizeAssignmentRepo.FindByEvent(ctx, *query.EventID)
		if err != nil {
			return nil, fmt.Errorf("failed to find prize assignments by event: %w", err)
		}
		return assignments, nil
	}

	// Если фильтры не указаны, возвращаем ошибку
	return nil, fmt.Errorf("event_id or participant_id must be specified")
}

// GetPrizeAssignmentByIDQuery представляет запрос на получение назначения приза по ID
type GetPrizeAssignmentByIDQuery struct {
	AssignmentID uint
}

// GetPrizeAssignmentByIDHandler обрабатывает запрос на получение назначения приза по ID
type GetPrizeAssignmentByIDHandler struct {
	prizeAssignmentRepo repository.PrizeAssignmentRepository
}

// NewGetPrizeAssignmentByIDHandler создаёт новый handler
func NewGetPrizeAssignmentByIDHandler(
	prizeAssignmentRepo repository.PrizeAssignmentRepository,
) *GetPrizeAssignmentByIDHandler {
	return &GetPrizeAssignmentByIDHandler{
		prizeAssignmentRepo: prizeAssignmentRepo,
	}
}

// Handle выполняет запрос на получение назначения приза по ID
func (h *GetPrizeAssignmentByIDHandler) Handle(ctx context.Context, query GetPrizeAssignmentByIDQuery) (*entity.PrizeAssignment, error) {
	assignment, err := h.prizeAssignmentRepo.FindByID(ctx, query.AssignmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find prize assignment: %w", err)
	}
	return assignment, nil
}
