package repository

import (
	"context"
	"gravel_bot/internal/domain/entity"
)

// GiftRepository определяет интерфейс для работы с подарками
type GiftRepository interface {
	// Create создаёт новый подарок
	Create(ctx context.Context, gift *entity.Gift) error

	// CreateWithAttachments создаёт новый подарок и прикреплённые файлы в одной транзакции
	CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error

	// Update обновляет подарок
	Update(ctx context.Context, gift *entity.Gift) error

	// UpdateWithCriteria обновляет подарок и полностью заменяет критерии в одной транзакции
	UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error

	// FindByID находит подарок по ID
	FindByID(ctx context.Context, id uint) (*entity.Gift, error)

	// FindByEvent находит все подарки события
	FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error)

	// FindByEventAndReviewStatus находит подарки события по статусу проверки
	FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error)

	// ListByEventPaged возвращает страницу подарков события (с опц. фильтром по статусу)
	// и общее количество с учётом фильтра. limit <= 0 — вернуть все строки.
	ListByEventPaged(ctx context.Context, eventID uint, reviewStatus *entity.GiftReviewStatus, limit, offset int) ([]*entity.Gift, int, error)

	// CountsByReviewStatus возвращает количество подарков события по статусам проверки
	// (ключи статусов + ключ "all" с общим количеством).
	CountsByReviewStatus(ctx context.Context, eventID uint) (map[string]int, error)

	// FindByUser находит все подарки пользователя
	FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error)

	// Delete удаляет подарок
	Delete(ctx context.Context, id uint) error

	// AddAttachment добавляет файл к подарку
	AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error

	// GetAttachments возвращает все файлы подарка
	GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error)
}
