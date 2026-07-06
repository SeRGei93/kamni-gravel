package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/repository"
)

// ExecuteChatPurgeResult — итог выполнения чистки чата.
type ExecuteChatPurgeResult struct {
	Kicked    int
	Failed    int
	Skipped   int
	Protected int
}

// ErrMemberNotInChat означает, что участника уже нет в чате (устаревший ростер).
// Такой кик не считается ошибкой — только пропуском.
var ErrMemberNotInChat = errors.New("member is not in chat")

// ErrChatPurgeNotConfigured означает, что кикер не настроен (нет токена или чата).
var ErrChatPurgeNotConfigured = errors.New("chat purge is not configured")

// ChatMemberKicker удаляет пользователя из публичного чата (kick = ban+unban).
type ChatMemberKicker interface {
	Kick(ctx context.Context, userID int64) error
}

// chatPurgeKickInterval — пауза между киками (ban/unban — админ-действия, не
// сообщения, поэтому лимит «1 сообщение/сек» не применяется).
const chatPurgeKickInterval = 50 * time.Millisecond

// ExecuteChatPurgeCommand — запрос на киковку выбранных пользователей.
type ExecuteChatPurgeCommand struct {
	EventID uint
	UserIDs []int64
}

// ExecuteChatPurgeHandler выполняет чистку чата с серверной защитой владельцев приза.
type ExecuteChatPurgeHandler struct {
	chatMemberRepo repository.ChatMemberRepository
	giftRepo       repository.GiftRepository
	kicker         ChatMemberKicker
	sleep          func(context.Context, time.Duration) bool
}

// NewExecuteChatPurgeHandler создаёт новый handler. kicker может быть nil, если
// функция не настроена — тогда Handle вернёт ErrChatPurgeNotConfigured.
func NewExecuteChatPurgeHandler(
	chatMemberRepo repository.ChatMemberRepository,
	giftRepo repository.GiftRepository,
	kicker ChatMemberKicker,
) *ExecuteChatPurgeHandler {
	return &ExecuteChatPurgeHandler{
		chatMemberRepo: chatMemberRepo,
		giftRepo:       giftRepo,
		kicker:         kicker,
		sleep:          sleepChatPurge,
	}
}

// Handle кикает выбранных пользователей, никогда не трогая владельцев приза.
func (h *ExecuteChatPurgeHandler) Handle(ctx context.Context, cmd ExecuteChatPurgeCommand) (ExecuteChatPurgeResult, error) {
	var result ExecuteChatPurgeResult

	if h.kicker == nil {
		return result, ErrChatPurgeNotConfigured
	}

	gifts, err := h.giftRepo.FindByEvent(ctx, cmd.EventID)
	if err != nil {
		return result, fmt.Errorf("find event gifts: %w", err)
	}
	giftOwners := make(map[int64]struct{}, len(gifts))
	for _, gift := range gifts {
		if gift != nil && gift.UserID > 0 {
			giftOwners[gift.UserID] = struct{}{}
		}
	}

	first := true
	for _, userID := range cmd.UserIDs {
		if userID <= 0 {
			continue
		}
		// Серверная защита: владельца приза не кикаем, даже если клиент прислал.
		if _, hasGift := giftOwners[userID]; hasGift {
			result.Protected++
			log.Printf("INFO chat purge protected: target_user_id=%d reason=has_gift", userID)
			continue
		}

		if !first {
			if !h.sleep(ctx, chatPurgeKickInterval) {
				log.Printf("WARN chat purge cancelled: event_id=%d kicked=%d failed=%d skipped=%d protected=%d error=%v", cmd.EventID, result.Kicked, result.Failed, result.Skipped, result.Protected, ctx.Err())
				return result, nil
			}
		}
		first = false

		err := h.kicker.Kick(ctx, userID)
		if errors.Is(err, ErrMemberNotInChat) {
			result.Skipped++
			log.Printf("INFO chat purge skipped: target_user_id=%d reason=not_in_chat", userID)
			continue
		}
		if err != nil {
			result.Failed++
			log.Printf("WARN chat purge kick failed: target_user_id=%d error=%v", userID, err)
			continue
		}

		result.Kicked++
		// Удаляем строку сразу; бот-апдейт chat_member тоже её удалит, но API не
		// должен полагаться на межпроцессный тайминг.
		if delErr := h.chatMemberRepo.Delete(ctx, userID); delErr != nil {
			log.Printf("WARN chat purge roster cleanup failed: target_user_id=%d error=%v", userID, delErr)
		}
	}

	log.Printf("INFO chat purge executed: event_id=%d requested=%d kicked=%d failed=%d skipped=%d protected=%d",
		cmd.EventID, len(cmd.UserIDs), result.Kicked, result.Failed, result.Skipped, result.Protected)
	return result, nil
}

func sleepChatPurge(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
