package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"

	telegrambot "github.com/go-telegram/bot"

	"gravel_bot/internal/application/command"
)

// kickerAPI — минимальный интерфейс Telegram-клиента для киковки (для тестов).
type kickerAPI interface {
	BanChatMember(ctx context.Context, params *telegrambot.BanChatMemberParams) (bool, error)
	UnbanChatMember(ctx context.Context, params *telegrambot.UnbanChatMemberParams) (bool, error)
}

// ChatMemberKicker удаляет пользователя из публичного чата: ban + сразу unban,
// чтобы человек мог вернуться по ссылке (kick, не бан).
type ChatMemberKicker struct {
	api    kickerAPI
	chatID int64
}

// NewChatMemberKicker создаёт kicker поверх готового Telegram-клиента.
func NewChatMemberKicker(api kickerAPI, chatID int64) *ChatMemberKicker {
	return &ChatMemberKicker{api: api, chatID: chatID}
}

// NewChatMemberKickerFromToken создаёт Telegram-клиент для kicker.
// Возвращает nil, если токен или chatID не заданы (функция отключена).
func NewChatMemberKickerFromToken(token string, chatID int64) (*ChatMemberKicker, error) {
	if strings.TrimSpace(token) == "" || chatID == 0 {
		return nil, nil
	}
	api, err := telegrambot.New(token, telegrambot.WithSkipGetMe())
	if err != nil {
		return nil, fmt.Errorf("create telegram client for chat member kicker: %w", err)
	}
	return NewChatMemberKicker(api, chatID), nil
}

// Kick удаляет пользователя из чата. Если участника уже нет в чате, возвращает
// command.ErrMemberNotInChat (не фатально — вызывающий считает это пропуском).
func (k *ChatMemberKicker) Kick(ctx context.Context, userID int64) error {
	if _, err := k.api.BanChatMember(ctx, &telegrambot.BanChatMemberParams{
		ChatID: k.chatID,
		UserID: userID,
	}); err != nil {
		if isMemberNotInChatError(err) {
			return command.ErrMemberNotInChat
		}
		return fmt.Errorf("ban chat member user_id=%d: %w", userID, err)
	}

	if _, err := k.api.UnbanChatMember(ctx, &telegrambot.UnbanChatMemberParams{
		ChatID:       k.chatID,
		UserID:       userID,
		OnlyIfBanned: true,
	}); err != nil {
		// Бан прошёл, разбан — нет: человек забанен и не вернётся сам.
		// Логируем и отдаём ошибку, чтобы попало в failed.
		log.Printf("WARN chat member unban failed after ban: target_user_id=%d error=%v", userID, err)
		return fmt.Errorf("unban chat member user_id=%d: %w", userID, err)
	}

	return nil
}

func isMemberNotInChatError(err error) bool {
	msg := strings.ToUpper(err.Error())
	for _, marker := range []string{
		"USER_NOT_PARTICIPANT",
		"PARTICIPANT_ID_INVALID",
		"USER_ID_INVALID",
		"MEMBER NOT FOUND",
		"USER NOT FOUND",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
