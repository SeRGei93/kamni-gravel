package repository

import (
	"context"
	"gravel_bot/internal/domain/entity"
)

// UserRepository определяет интерфейс для работы с пользователями
type UserRepository interface {
	// Create создаёт нового пользователя
	Create(ctx context.Context, user *entity.User) error
	
	// Update обновляет данные пользователя
	Update(ctx context.Context, user *entity.User) error
	
	// FindByID находит пользователя по Telegram ID
	FindByID(ctx context.Context, id int64) (*entity.User, error)
	
	// Delete удаляет пользователя
	Delete(ctx context.Context, id int64) error
	
	// GetAll возвращает всех пользователей
	GetAll(ctx context.Context) ([]*entity.User, error)
}
