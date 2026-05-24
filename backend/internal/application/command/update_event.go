package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// UpdateEventCommand представляет команду обновления события
type UpdateEventCommand struct {
	EventID     uint
	Name        *string
	Description *string
	Active      *bool
	StartDate   *time.Time
	EndDate     *time.Time
	GPXFilePath *string
}

// UpdateEventHandler обрабатывает обновление события
type UpdateEventHandler struct {
	eventRepo repository.EventRepository
}

// NewUpdateEventHandler создаёт новый handler
func NewUpdateEventHandler(
	eventRepo repository.EventRepository,
) *UpdateEventHandler {
	return &UpdateEventHandler{
		eventRepo: eventRepo,
	}
}

// Handle выполняет команду обновления события
func (h *UpdateEventHandler) Handle(ctx context.Context, cmd UpdateEventCommand) (*entity.Event, error) {
	// Находим событие
	event, err := h.eventRepo.FindByID(ctx, cmd.EventID)
	if err != nil {
		return nil, ErrEventNotFound
	}

	// Обновляем поля, если они указаны
	if cmd.Name != nil {
		// Проверяем, что имя не занято другим событием
		if *cmd.Name != event.Name {
			existing, err := h.eventRepo.FindByName(ctx, *cmd.Name)
			if err == nil && existing != nil && existing.ID != event.ID {
				return nil, ErrEventNameExists
			}
		}
		event.Name = *cmd.Name
	}

	if cmd.Description != nil {
		event.Description = *cmd.Description
	}

	if cmd.Active != nil {
		// Если активируем событие, деактивируем все остальные
		if *cmd.Active && !event.Active {
			activeEvent, err := h.eventRepo.FindActive(ctx)
			if err == nil && activeEvent != nil && activeEvent.ID != event.ID {
				activeEvent.Active = false
				if err := h.eventRepo.Update(ctx, activeEvent); err != nil {
					return nil, fmt.Errorf("failed to deactivate existing active event: %w", err)
				}
			}
		}
		event.Active = *cmd.Active
	}

	if cmd.StartDate != nil {
		event.StartDate = cmd.StartDate
	}

	if cmd.EndDate != nil {
		event.EndDate = cmd.EndDate
	}

	if cmd.GPXFilePath != nil {
		event.GPXFilePath = *cmd.GPXFilePath
	}

	// Валидация дат
	if event.StartDate != nil && event.EndDate != nil {
		if event.EndDate.Before(*event.StartDate) {
			return nil, errors.New("end date must be after start date")
		}
	}

	// Обновляем время изменения
	event.UpdatedAt = time.Now()

	// Сохраняем изменения в БД
	if err := h.eventRepo.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return event, nil
}
