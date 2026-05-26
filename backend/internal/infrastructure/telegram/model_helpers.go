package telegram

import (
	"log"
	"strings"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/infrastructure/telegram/session"
)

type callbackMessageRef struct {
	ChatID    int64
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

func callbackMessage(callback *models.CallbackQuery) (callbackMessageRef, bool) {
	if callback == nil {
		log.Printf("Telegram callback ignored: nil callback")
		return callbackMessageRef{}, false
	}

	if callback.Message.Message != nil {
		return callbackMessageRef{
			ChatID:    callback.Message.Message.Chat.ID,
			MessageID: callback.Message.Message.ID,
		}, true
	}

	if callback.Message.InaccessibleMessage != nil {
		log.Printf(
			"Telegram callback message inaccessible: callback_id=%s chat_id=%d message_id=%d",
			callback.ID,
			callback.Message.InaccessibleMessage.Chat.ID,
			callback.Message.InaccessibleMessage.MessageID,
		)
		return callbackMessageRef{}, false
	}

	log.Printf("Telegram callback ignored: callback_id=%s missing message", callback.ID)
	return callbackMessageRef{}, false
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
