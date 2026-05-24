package telegram

import (
	"log"
	"strings"

	"github.com/go-telegram/bot/models"
)

type callbackMessageRef struct {
	ChatID    int64
	MessageID int
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
