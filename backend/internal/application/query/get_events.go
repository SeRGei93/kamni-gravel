package query

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetEventsQuery представляет запрос на получение всех событий
type GetEventsQuery struct {
	ActiveOnly bool // если true, возвращает только активные события
}

// GetEventsHandler обрабатывает запрос на получение событий
type GetEventsHandler struct {
	eventRepo repository.EventRepository
}

// NewGetEventsHandler создаёт новый handler
func NewGetEventsHandler(
	eventRepo repository.EventRepository,
) *GetEventsHandler {
	return &GetEventsHandler{
		eventRepo: eventRepo,
	}
}

// Handle выполняет запрос на получение событий
func (h *GetEventsHandler) Handle(ctx context.Context, query GetEventsQuery) ([]*entity.Event, error) {
	if query.ActiveOnly {
		// Возвращаем только активное событие
		event, err := h.eventRepo.FindActive(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to find active event: %w", err)
		}
		if event == nil {
			return []*entity.Event{}, nil
		}
		return []*entity.Event{event}, nil
	}

	// Возвращаем все события
	events, err := h.eventRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	return events, nil
}

// GetEventByIDQuery представляет запрос на получение события по ID
type GetEventByIDQuery struct {
	EventID uint
}

// GetEventByIDHandler обрабатывает запрос на получение события по ID
type GetEventByIDHandler struct {
	eventRepo repository.EventRepository
}

// NewGetEventByIDHandler создаёт новый handler
func NewGetEventByIDHandler(
	eventRepo repository.EventRepository,
) *GetEventByIDHandler {
	return &GetEventByIDHandler{
		eventRepo: eventRepo,
	}
}

// Handle выполняет запрос на получение события по ID
func (h *GetEventByIDHandler) Handle(ctx context.Context, query GetEventByIDQuery) (*entity.Event, error) {
	event, err := h.eventRepo.FindByID(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to find event: %w", err)
	}
	return event, nil
}
