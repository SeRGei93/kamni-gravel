package query

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// ErrInvalidTelegramUserID означает невалидный Telegram ID в query.
var ErrInvalidTelegramUserID = errors.New("telegram user id must be positive")

// ListUserBlacklistHandler обрабатывает запрос списка blacklist.
type ListUserBlacklistHandler struct {
	userBlacklistRepo repository.UserBlacklistRepository
}

// NewListUserBlacklistHandler создаёт новый handler.
func NewListUserBlacklistHandler(userBlacklistRepo repository.UserBlacklistRepository) *ListUserBlacklistHandler {
	return &ListUserBlacklistHandler{userBlacklistRepo: userBlacklistRepo}
}

// Handle выполняет запрос списка blacklist.
func (h *ListUserBlacklistHandler) Handle(ctx context.Context) ([]*entity.UserBlacklist, error) {
	entries, err := h.userBlacklistRepo.List(ctx)
	if err != nil {
		log.Printf("ERROR User blacklist list failed: error=%v", err)
		return nil, fmt.Errorf("list user blacklist: %w", err)
	}

	return entries, nil
}

// IsUserBlacklistedQuery представляет запрос проверки blacklist.
type IsUserBlacklistedQuery struct {
	TelegramUserID int64
}

// IsUserBlacklistedHandler обрабатывает запрос проверки blacklist.
type IsUserBlacklistedHandler struct {
	userBlacklistRepo repository.UserBlacklistRepository
}

// NewIsUserBlacklistedHandler создаёт новый handler.
func NewIsUserBlacklistedHandler(userBlacklistRepo repository.UserBlacklistRepository) *IsUserBlacklistedHandler {
	return &IsUserBlacklistedHandler{userBlacklistRepo: userBlacklistRepo}
}

// Handle выполняет запрос проверки blacklist.
func (h *IsUserBlacklistedHandler) Handle(ctx context.Context, q IsUserBlacklistedQuery) (bool, error) {
	if q.TelegramUserID <= 0 {
		log.Printf("WARN User blacklist validation failed: operation=is_blacklisted telegram_user_id=%d reason=invalid_id", q.TelegramUserID)
		return false, ErrInvalidTelegramUserID
	}

	isBlacklisted, err := h.userBlacklistRepo.IsBlacklisted(ctx, q.TelegramUserID)
	if err != nil {
		log.Printf("ERROR User blacklist check failed: telegram_user_id=%d error=%v", q.TelegramUserID, err)
		return false, fmt.Errorf("check user blacklist: %w", err)
	}

	return isBlacklisted, nil
}
