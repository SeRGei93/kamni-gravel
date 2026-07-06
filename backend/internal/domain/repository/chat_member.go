package repository

import (
	"context"

	"gravel_bot/internal/domain/entity"
)

// ChatMemberRepository определяет интерфейс для работы с ростером публичного чата.
type ChatMemberRepository interface {
	// Upsert добавляет или обновляет участника чата (joined_at не перезаписывается).
	Upsert(ctx context.Context, member *entity.ChatMember) error

	// BulkUpsert добавляет или обновляет набор участников в одной транзакции.
	BulkUpsert(ctx context.Context, members []*entity.ChatMember) error

	// Delete удаляет участника из ростера (без ошибки, если его нет).
	Delete(ctx context.Context, telegramUserID int64) error

	// GetAll возвращает всех текущих участников чата.
	GetAll(ctx context.Context) ([]*entity.ChatMember, error)

	// Count возвращает количество участников в ростере.
	Count(ctx context.Context) (int, error)
}
