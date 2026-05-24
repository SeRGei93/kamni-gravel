package repository

import (
	"context"
	"gravel_bot/internal/domain/entity"
)

// EventRepository определяет интерфейс для работы с событиями
type EventRepository interface {
	// Create создаёт новое событие
	Create(ctx context.Context, event *entity.Event) error
	
	// Update обновляет событие
	Update(ctx context.Context, event *entity.Event) error
	
	// FindByID находит событие по ID
	FindByID(ctx context.Context, id uint) (*entity.Event, error)
	
	// FindByName находит событие по имени
	FindByName(ctx context.Context, name string) (*entity.Event, error)
	
	// FindActive находит активное событие
	FindActive(ctx context.Context) (*entity.Event, error)
	
	// GetAll возвращает все события
	GetAll(ctx context.Context) ([]*entity.Event, error)
	
	// Delete удаляет событие
	Delete(ctx context.Context, id uint) error
}
