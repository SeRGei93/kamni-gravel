package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

var (
	ErrEventNameRequired = errors.New("event name is required")
	ErrEventNameExists   = errors.New("event with this name already exists")
)

// CreateEventCommand представляет команду создания события
type CreateEventCommand struct {
	Name          string
	Description   string
	Active        bool
	StartDate     *time.Time
	EndDate       *time.Time
	GPXFilePath   string
	TelegramTexts entity.EventTelegramTexts
}

// CreateEventHandler обрабатывает создание события
type CreateEventHandler struct {
	eventRepo repository.EventRepository
}

// NewCreateEventHandler создаёт новый handler
func NewCreateEventHandler(
	eventRepo repository.EventRepository,
) *CreateEventHandler {
	return &CreateEventHandler{
		eventRepo: eventRepo,
	}
}

// Handle выполняет команду создания события
func (h *CreateEventHandler) Handle(ctx context.Context, cmd CreateEventCommand) (*entity.Event, error) {
	// Валидация имени
	if cmd.Name == "" {
		return nil, ErrEventNameRequired
	}

	// Проверяем, что события с таким именем не существует
	existing, err := h.eventRepo.FindByName(ctx, cmd.Name)
	if err == nil && existing != nil {
		return nil, ErrEventNameExists
	}

	// Валидация дат
	if cmd.StartDate != nil && cmd.EndDate != nil {
		if cmd.EndDate.Before(*cmd.StartDate) {
			return nil, errors.New("end date must be after start date")
		}
	}

	// Если создаём активное событие, деактивируем все остальные
	if cmd.Active {
		activeEvent, err := h.eventRepo.FindActive(ctx)
		if err == nil && activeEvent != nil {
			activeEvent.Active = false
			if err := h.eventRepo.Update(ctx, activeEvent); err != nil {
				return nil, fmt.Errorf("failed to deactivate existing active event: %w", err)
			}
		}
	}

	// Создаём событие
	now := time.Now()
	event := &entity.Event{
		Name:          cmd.Name,
		Description:   cmd.Description,
		Active:        cmd.Active,
		StartDate:     cmd.StartDate,
		EndDate:       cmd.EndDate,
		GPXFilePath:   cmd.GPXFilePath,
		TelegramTexts: entity.NormalizeEventTelegramTexts(cmd.TelegramTexts),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Сохраняем событие в БД
	if err := h.eventRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}
