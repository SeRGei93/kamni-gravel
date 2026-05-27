package telegram

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/infrastructure/telegram/session"
)

type callbackMessageRef struct {
	ChatID    int64
	ChatType  models.ChatType
	MessageID int
}

type giftMessageReply int

const (
	giftMessageReplyNone giftMessageReply = iota
	giftMessageReplyGiftGenderStep
	giftMessageReplyGiftBikeStep
	giftMessageReplyGiftDescriptionStep
	giftMessageReplyGiftPhotoStep
	giftMessageReplyGiftPhotoAdded
	giftMessageReplyGiftConfirmationStep
)

type giftMessageActionResult struct {
	Description        string
	PhotoFileID        string
	Reply              giftMessageReply
	ProcessDescription bool
	ProcessPhoto       bool
	SuppressReply      bool
	OutOfOrder         bool
	MissingInput       bool
	UpdateKind         string
}

type giftMediaGroupReplyState struct {
	MediaGroupID   string
	ChatID         int64
	FirstMessageID int
	State          session.SessionState
}

type giftSourceRef struct {
	ChatID     int64
	MessageID  int
	UpdateKind string
}

const (
	giftSourceRefsKey = "gift_source_refs"
)

func messageCommand(msg *models.Message) string {
	if msg == nil || msg.Text == "" {
		return ""
	}

	for _, entity := range msg.Entities {
		if entity.Type != models.MessageEntityTypeBotCommand || entity.Offset != 0 {
			continue
		}

		end := entity.Offset + entity.Length
		if entity.Offset < 0 || end > len(msg.Text) || entity.Length <= 1 {
			return ""
		}

		command := strings.TrimPrefix(msg.Text[entity.Offset:end], "/")
		if at := strings.Index(command, "@"); at >= 0 {
			command = command[:at]
		}

		return command
	}

	return ""
}

func messageSender(msg *models.Message) (*models.User, bool) {
	if msg == nil || msg.From == nil {
		log.Printf("Telegram message ignored: missing sender")
		return nil, false
	}

	return msg.From, true
}

func telegramUpdateSender(update *models.Update) (int64, string, bool) {
	if update == nil {
		return 0, "nil", false
	}

	if update.Message != nil {
		sender, ok := messageSender(update.Message)
		if !ok {
			return 0, messageUpdateKind(update.Message), false
		}
		if command := messageCommand(update.Message); command != "" {
			return sender.ID, "command:" + command, true
		}
		return sender.ID, "message:" + messageUpdateKind(update.Message), true
	}

	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID, "callback", true
	}

	return 0, "unsupported", false
}

func callbackMessage(callback *models.CallbackQuery) (callbackMessageRef, bool) {
	if callback == nil {
		log.Printf("Telegram callback ignored: nil callback")
		return callbackMessageRef{}, false
	}

	if callback.Message.Message != nil {
		return callbackMessageRef{
			ChatID:    callback.Message.Message.Chat.ID,
			ChatType:  callback.Message.Message.Chat.Type,
			MessageID: callback.Message.Message.ID,
		}, true
	}

	if callback.Message.InaccessibleMessage != nil {
		log.Printf(
			"Telegram callback message inaccessible: callback_id=%s chat=redacted message_id=%d",
			callback.ID,
			callback.Message.InaccessibleMessage.MessageID,
		)
		return callbackMessageRef{}, false
	}

	log.Printf("Telegram callback ignored: callback_id=%s missing message", callback.ID)
	return callbackMessageRef{}, false
}

func isPrivateTelegramChat(chat models.Chat) bool {
	if chat.Type != "" {
		return chat.Type == models.ChatTypePrivate
	}

	return chat.ID > 0
}

func isPrivateTelegramChatRef(ref callbackMessageRef) bool {
	if ref.ChatType != "" {
		return ref.ChatType == models.ChatTypePrivate
	}

	return ref.ChatID > 0
}

func messageTextOrCaption(msg *models.Message) string {
	if msg == nil {
		return ""
	}

	text := strings.TrimSpace(msg.Text)
	if text != "" {
		return text
	}

	return strings.TrimSpace(msg.Caption)
}

func resultLinkText(msg *models.Message) (string, bool) {
	if msg == nil || len(msg.Photo) > 0 || msg.Document != nil || msg.Video != nil {
		return "", false
	}

	text := strings.TrimSpace(msg.Text)
	return text, text != ""
}

func largestPhotoFileID(msg *models.Message) (string, bool) {
	if msg == nil || len(msg.Photo) == 0 {
		return "", false
	}

	best := msg.Photo[0]
	bestScore := photoSizeScore(best)
	for _, photo := range msg.Photo[1:] {
		score := photoSizeScore(photo)
		if score > bestScore {
			best = photo
			bestScore = score
		}
	}

	fileID := strings.TrimSpace(best.FileID)
	if fileID == "" {
		return "", false
	}

	return fileID, true
}

func photoSizeScore(photo models.PhotoSize) int {
	if photo.FileSize > 0 {
		return photo.FileSize
	}

	return photo.Width * photo.Height
}

func giftMessageAction(state session.SessionState, msg *models.Message, mediaGroupAlreadyReplied bool) giftMessageActionResult {
	action := giftMessageActionResult{
		UpdateKind: messageUpdateKind(msg),
	}
	if msg == nil {
		action.MissingInput = true
		return action
	}

	description := messageTextOrCaption(msg)
	photoFileID, hasPhoto := largestPhotoFileID(msg)

	switch state {
	case session.StateAwaitingGiftGender:
		action.Reply = giftMessageReplyGiftGenderStep
		action.OutOfOrder = true

	case session.StateAwaitingGiftBikeType:
		action.Reply = giftMessageReplyGiftBikeStep
		action.OutOfOrder = true

	case session.StateAwaitingGiftDesc:
		if description != "" {
			action.Description = description
			action.ProcessDescription = true
			action.Reply = giftMessageReplyGiftPhotoStep
			if hasPhoto {
				action.PhotoFileID = photoFileID
				action.ProcessPhoto = true
			}
			break
		}

		action.Reply = giftMessageReplyGiftDescriptionStep
		action.MissingInput = true
		if hasPhoto {
			action.OutOfOrder = true
		}

	case session.StateAwaitingGiftPhoto:
		action.Reply = giftMessageReplyGiftPhotoStep
		if hasPhoto {
			action.PhotoFileID = photoFileID
			action.ProcessPhoto = true
			if msg.MediaGroupID == "" {
				action.Reply = giftMessageReplyGiftPhotoAdded
			}
		} else {
			action.MissingInput = true
			action.OutOfOrder = true
		}

	case session.StateAwaitingGiftConfirmation:
		action.Reply = giftMessageReplyGiftConfirmationStep
		action.OutOfOrder = true

	default:
		return action
	}

	if mediaGroupAlreadyReplied && action.Reply != giftMessageReplyNone {
		action.SuppressReply = true
		action.Reply = giftMessageReplyNone
	}

	return action
}

func messageUpdateKind(msg *models.Message) string {
	if msg == nil {
		return "nil"
	}
	if len(msg.Photo) > 0 {
		return "photo"
	}
	if strings.TrimSpace(msg.Text) != "" {
		return "text"
	}
	if strings.TrimSpace(msg.Caption) != "" {
		return "caption"
	}
	if msg.Document != nil {
		return "document"
	}
	if msg.Video != nil {
		return "video"
	}
	return "unknown"
}

func (r giftSourceRef) String() string {
	return fmt.Sprintf("chat=redacted message_id=%d kind=%s", r.MessageID, r.UpdateKind)
}

func (b *Bot) captureGiftMessageSourceRef(userID int64, msg *models.Message) {
	if msg == nil || b == nil || b.sessionManager == nil {
		return
	}

	updateKind := messageUpdateKind(msg)
	if updateKind == "nil" || updateKind == "unknown" {
		b.logDebug("Gift message source ref skipped: user_id=%d reason=unsupported_update_kind kind=%s", userID, updateKind)
		return
	}

	if msg.Chat.ID == 0 || msg.ID == 0 {
		b.logDebug("Gift message source ref skipped: user_id=%d reason=missing_message_identifier", userID)
		return
	}

	refs := b.giftSourceRefs(userID)
	refs = append(refs, giftSourceRef{
		ChatID:     msg.Chat.ID,
		MessageID:  msg.ID,
		UpdateKind: updateKind,
	})
	b.setGiftSourceRefs(userID, refs)

	b.logDebug("Gift message source ref captured: user_id=%d kind=%s", userID, updateKind)
}

func (b *Bot) giftSourceRefs(userID int64) []giftSourceRef {
	if b == nil || b.sessionManager == nil {
		return nil
	}

	raw, ok := b.sessionManager.GetData(userID, giftSourceRefsKey)
	if !ok {
		return nil
	}

	refs, ok := raw.([]giftSourceRef)
	if !ok {
		log.Printf("WARN Invalid gift source refs state: user_id=%d key=%s type=%T", userID, giftSourceRefsKey, raw)
		return nil
	}

	return refs
}

func (b *Bot) setGiftSourceRefs(userID int64, refs []giftSourceRef) {
	if b == nil || b.sessionManager == nil {
		return
	}

	b.sessionManager.SetData(userID, giftSourceRefsKey, refs)
}
