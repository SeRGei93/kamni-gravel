package repository

import (
	"context"
	"gravel_bot/internal/domain/entity"
)

// GiftRepository определяет интерфейс для работы с подарками
type GiftRepository interface {
	// Create создаёт новый подарок
	Create(ctx context.Context, gift *entity.Gift) error

	// Update обновляет подарок
	Update(ctx context.Context, gift *entity.Gift) error

	// FindByID находит подарок по ID
	FindByID(ctx context.Context, id uint) (*entity.Gift, error)

	// FindByEvent находит все подарки события
	FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error)

	// FindByUser находит все подарки пользователя
	FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error)

	// Delete удаляет подарок
	Delete(ctx context.Context, id uint) error

	// AddAttachment добавляет файл к подарку
	AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error

	// GetAttachments возвращает все файлы подарка
	GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error)
}
