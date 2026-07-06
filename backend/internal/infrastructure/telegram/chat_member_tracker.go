package telegram

import (
	"context"
	"log"
	"strings"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/domain/entity"
)

// handleChatMemberUpdate поддерживает актуальность ростера публичного чата по
// Telegram chat_member апдейтам: upsert при входе/повышении, удаление строки при
// выходе/кике.
func (b *Bot) handleChatMemberUpdate(ctx context.Context, upd *models.ChatMemberUpdated) {
	if upd == nil {
		return
	}
	if !b.isPublicChat(upd.Chat.ID) {
		b.logDebug("Telegram chat_member update ignored outside public chat: chat=%s", b.chatLogMarker(upd.Chat.ID))
		return
	}
	if b.chatMemberRepo == nil {
		return
	}

	user, present, isAdmin := chatMemberStatus(upd.NewChatMember)
	if user == nil || user.ID == 0 {
		log.Printf("WARN Chat member update skipped: chat=public reason=missing_user status=%s", upd.NewChatMember.Type)
		return
	}

	if !present {
		if err := b.chatMemberRepo.Delete(ctx, user.ID); err != nil {
			log.Printf("WARN Chat member remove failed: chat=public target_user_id=%d error=%v", user.ID, err)
			return
		}
		log.Printf("INFO chat member removed: chat=public target_user_id=%d reason=%s", user.ID, upd.NewChatMember.Type)
		return
	}

	member := &entity.ChatMember{
		TelegramUserID: user.ID,
		Username:       strings.TrimSpace(user.Username),
		FirstName:      strings.TrimSpace(user.FirstName),
		LastName:       strings.TrimSpace(user.LastName),
		IsBot:          user.IsBot,
		IsAdmin:        isAdmin,
	}
	if err := b.chatMemberRepo.Upsert(ctx, member); err != nil {
		log.Printf("WARN Chat member upsert failed: chat=public target_user_id=%d error=%v", user.ID, err)
		return
	}
	log.Printf("INFO chat member upserted: chat=public target_user_id=%d is_admin=%t", user.ID, isAdmin)
}

// chatMemberStatus извлекает затронутого пользователя, признак присутствия в чате
// и признак администратора из варианта models.ChatMember.
func chatMemberStatus(member models.ChatMember) (user *models.User, present bool, isAdmin bool) {
	switch member.Type {
	case models.ChatMemberTypeOwner:
		if member.Owner != nil {
			return member.Owner.User, true, true
		}
	case models.ChatMemberTypeAdministrator:
		if member.Administrator != nil {
			return &member.Administrator.User, true, true
		}
	case models.ChatMemberTypeMember:
		if member.Member != nil {
			return member.Member.User, true, false
		}
	case models.ChatMemberTypeRestricted:
		if member.Restricted != nil {
			// restricted всё ещё в чате, только если is_member = true.
			return member.Restricted.User, member.Restricted.IsMember, false
		}
	case models.ChatMemberTypeLeft:
		if member.Left != nil {
			return member.Left.User, false, false
		}
	case models.ChatMemberTypeBanned:
		if member.Banned != nil {
			return member.Banned.User, false, false
		}
	}
	return nil, false, false
}
