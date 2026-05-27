package repository

import (
	"context"
	"errors"

	"gravel_bot/internal/domain/entity"
)

// ErrUserBlacklistEntryNotFound означает, что запись blacklist не найдена.
var ErrUserBlacklistEntryNotFound = errors.New("user blacklist entry not found")

// UserBlacklistRepository определяет интерфейс для работы с blacklist пользователей.
type UserBlacklistRepository interface {
	// List возвращает все записи blacklist с данными Telegram-профиля, если пользователь есть.
	List(ctx context.Context) ([]*entity.UserBlacklist, error)

	// FindByTelegramUserID находит запись blacklist по Telegram ID.
	FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error)

	// IsBlacklisted проверяет, находится ли Telegram ID в blacklist.
	IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error)

	// Upsert добавляет или обновляет запись blacklist.
	Upsert(ctx context.Context, entry *entity.UserBlacklist) error

	// UpdateReason обновляет причину блокировки.
	UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error)

	// Delete удаляет запись blacklist.
	Delete(ctx context.Context, telegramUserID int64) error
}
