package repository

import (
	"context"
	"gravel_bot/internal/domain/entity"
)

// AdminRepository определяет интерфейс для работы с администраторами
type AdminRepository interface {
	// Create создаёт нового администратора
	Create(ctx context.Context, admin *entity.Admin) error
	
	// FindByUsername находит администратора по имени пользователя
	FindByUsername(ctx context.Context, username string) (*entity.Admin, error)
	
	// FindByID находит администратора по ID
	FindByID(ctx context.Context, id uint) (*entity.Admin, error)
	
	// UpdateLastLogin обновляет время последнего входа
	UpdateLastLogin(ctx context.Context, id uint) error
	
	// Update обновляет данные администратора
	Update(ctx context.Context, admin *entity.Admin) error
	
	// Delete удаляет администратора
	Delete(ctx context.Context, id uint) error
}
