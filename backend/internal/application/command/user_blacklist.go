package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

var (
	ErrInvalidTelegramUserID = errors.New("telegram user id must be positive")
	ErrUserBlacklistNotFound = errors.New("user blacklist entry not found")
	ErrUserBlacklisted       = errors.New("user is blacklisted")
)

// AddUserBlacklistCommand представляет команду добавления Telegram ID в blacklist.
type AddUserBlacklistCommand struct {
	TelegramUserID int64
	Reason         string
}

// AddUserBlacklistHandler обрабатывает добавление Telegram ID в blacklist.
type AddUserBlacklistHandler struct {
	userBlacklistRepo repository.UserBlacklistRepository
}

// NewAddUserBlacklistHandler создаёт новый handler.
func NewAddUserBlacklistHandler(userBlacklistRepo repository.UserBlacklistRepository) *AddUserBlacklistHandler {
	return &AddUserBlacklistHandler{userBlacklistRepo: userBlacklistRepo}
}

// Handle выполняет команду добавления Telegram ID в blacklist.
func (h *AddUserBlacklistHandler) Handle(ctx context.Context, cmd AddUserBlacklistCommand) (*entity.UserBlacklist, error) {
	if cmd.TelegramUserID <= 0 {
		log.Printf("WARN User blacklist validation failed: operation=add telegram_user_id=%d reason=invalid_id", cmd.TelegramUserID)
		return nil, ErrInvalidTelegramUserID
	}

	entry := &entity.UserBlacklist{
		TelegramUserID: cmd.TelegramUserID,
		Reason:         strings.TrimSpace(cmd.Reason),
	}

	log.Printf("INFO User blacklist add requested: telegram_user_id=%d", entry.TelegramUserID)
	if err := h.userBlacklistRepo.Upsert(ctx, entry); err != nil {
		log.Printf("ERROR User blacklist add failed: telegram_user_id=%d error=%v", entry.TelegramUserID, err)
		return nil, fmt.Errorf("add user blacklist entry: %w", err)
	}

	log.Printf("INFO User blacklist add completed: telegram_user_id=%d", entry.TelegramUserID)
	return entry, nil
}

// UpdateUserBlacklistReasonCommand представляет команду обновления причины блокировки.
type UpdateUserBlacklistReasonCommand struct {
	TelegramUserID int64
	Reason         string
}

// UpdateUserBlacklistReasonHandler обрабатывает обновление причины блокировки.
type UpdateUserBlacklistReasonHandler struct {
	userBlacklistRepo repository.UserBlacklistRepository
}

// NewUpdateUserBlacklistReasonHandler создаёт новый handler.
func NewUpdateUserBlacklistReasonHandler(userBlacklistRepo repository.UserBlacklistRepository) *UpdateUserBlacklistReasonHandler {
	return &UpdateUserBlacklistReasonHandler{userBlacklistRepo: userBlacklistRepo}
}

// Handle выполняет команду обновления причины блокировки.
func (h *UpdateUserBlacklistReasonHandler) Handle(ctx context.Context, cmd UpdateUserBlacklistReasonCommand) (*entity.UserBlacklist, error) {
	if cmd.TelegramUserID <= 0 {
		log.Printf("WARN User blacklist validation failed: operation=update_reason telegram_user_id=%d reason=invalid_id", cmd.TelegramUserID)
		return nil, ErrInvalidTelegramUserID
	}

	reason := strings.TrimSpace(cmd.Reason)
	log.Printf("INFO User blacklist update requested: telegram_user_id=%d", cmd.TelegramUserID)
	entry, err := h.userBlacklistRepo.UpdateReason(ctx, cmd.TelegramUserID, reason)
	if errors.Is(err, repository.ErrUserBlacklistEntryNotFound) {
		log.Printf("WARN User blacklist update skipped: telegram_user_id=%d reason=not_found", cmd.TelegramUserID)
		return nil, ErrUserBlacklistNotFound
	}
	if err != nil {
		log.Printf("ERROR User blacklist update failed: telegram_user_id=%d error=%v", cmd.TelegramUserID, err)
		return nil, fmt.Errorf("update user blacklist reason: %w", err)
	}

	log.Printf("INFO User blacklist update completed: telegram_user_id=%d", cmd.TelegramUserID)
	return entry, nil
}

// RemoveUserBlacklistCommand представляет команду удаления Telegram ID из blacklist.
type RemoveUserBlacklistCommand struct {
	TelegramUserID int64
}

// RemoveUserBlacklistHandler обрабатывает удаление Telegram ID из blacklist.
type RemoveUserBlacklistHandler struct {
	userBlacklistRepo repository.UserBlacklistRepository
}

// NewRemoveUserBlacklistHandler создаёт новый handler.
func NewRemoveUserBlacklistHandler(userBlacklistRepo repository.UserBlacklistRepository) *RemoveUserBlacklistHandler {
	return &RemoveUserBlacklistHandler{userBlacklistRepo: userBlacklistRepo}
}

// Handle выполняет команду удаления Telegram ID из blacklist.
func (h *RemoveUserBlacklistHandler) Handle(ctx context.Context, cmd RemoveUserBlacklistCommand) error {
	if cmd.TelegramUserID <= 0 {
		log.Printf("WARN User blacklist validation failed: operation=remove telegram_user_id=%d reason=invalid_id", cmd.TelegramUserID)
		return ErrInvalidTelegramUserID
	}

	log.Printf("INFO User blacklist remove requested: telegram_user_id=%d", cmd.TelegramUserID)
	err := h.userBlacklistRepo.Delete(ctx, cmd.TelegramUserID)
	if errors.Is(err, repository.ErrUserBlacklistEntryNotFound) {
		log.Printf("WARN User blacklist remove skipped: telegram_user_id=%d reason=not_found", cmd.TelegramUserID)
		return ErrUserBlacklistNotFound
	}
	if err != nil {
		log.Printf("ERROR User blacklist remove failed: telegram_user_id=%d error=%v", cmd.TelegramUserID, err)
		return fmt.Errorf("remove user blacklist entry: %w", err)
	}

	log.Printf("INFO User blacklist remove completed: telegram_user_id=%d", cmd.TelegramUserID)
	return nil
}
