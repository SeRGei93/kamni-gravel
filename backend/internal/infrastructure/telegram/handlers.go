package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gravel_bot/internal/infrastructure/telegram/handler"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// handleCommand обрабатывает команды бота
func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		b.handleStartCommand(ctx, msg)
	default:
		b.SendMessage(msg.Chat.ID, "Неизвестная команда. Используйте /start")
	}
}

// handleStartCommand обрабатывает команду /start
func (b *Bot) handleStartCommand(ctx context.Context, msg *tgbotapi.Message) {
	startHandler := handler.NewStartHandler(b.userRepo, b.eventRepo)
	text, keyboard := startHandler.Handle(ctx, msg)

	if keyboard != nil {
		b.SendMessageWithKeyboard(msg.Chat.ID, text, *keyboard)
	} else {
		b.SendMessage(msg.Chat.ID, text)
	}
}

// handleCallback обрабатывает callback-запросы (нажатия на inline кнопки)
func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	data := callback.Data

	// Обрабатываем отмену
	if data == "cancel" {
		b.sessionManager.ResetState(userID)
		b.AnswerCallback(callback.ID, "Отменено")
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, "Действие отменено.")
		return
	}

	// Обрабатываем основные действия
	switch data {
	case "register":
		b.handleRegisterCallback(ctx, callback)
	case "add_gift":
		b.handleAddGiftCallback(ctx, callback)
	case "submit_result":
		b.handleSubmitResultCallback(ctx, callback)
	case "info":
		b.handleInfoCallback(ctx, callback)
	default:
		// Обрабатываем callback в зависимости от состояния сессии
		b.handleStatefulCallback(ctx, callback)
	}
}

// handleRegisterCallback обрабатывает начало регистрации
func (b *Bot) handleRegisterCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	registrationHandler := handler.NewRegistrationHandler(
		b.sessionManager,
		b.eventRepo,
		b.participantRepo,
		b.registerParticipantHandler,
	)

	text, keyboard := registrationHandler.StartRegistration(ctx, callback.From.ID)

	b.AnswerCallback(callback.ID, "")
	if keyboard != nil {
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text)
		b.SendMessageWithKeyboard(callback.Message.Chat.ID, text, *keyboard)
	} else {
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text)
	}
}

// handleAddGiftCallback обрабатывает начало добавления подарка
func (b *Bot) handleAddGiftCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	giftHandler := handler.NewGiftHandler(
		b.sessionManager,
		b.eventRepo,
		b.addGiftHandler,
	)

	text, keyboard := giftHandler.StartAddGift(ctx, callback.From.ID)

	b.AnswerCallback(callback.ID, "")
	
	// Удаляем предыдущее сообщение
	b.DeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
	
	// Сохраняем ID нового сообщения для последующего удаления
	var messageIDs []int
	if keyboard != nil {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
		msg.ReplyMarkup = keyboard
		sentMsg, err := b.api.Send(msg)
		if err == nil {
			messageIDs = append(messageIDs, sentMsg.MessageID)
			b.sessionManager.SetData(callback.From.ID, "gift_message_ids", messageIDs)
		}
	}
}

// handleSubmitResultCallback обрабатывает начало отправки результата
func (b *Bot) handleSubmitResultCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	resultHandler := handler.NewResultHandler(
		b.sessionManager,
		b.eventRepo,
		b.participantRepo,
		b.submitResultHandler,
	)

	text, keyboard := resultHandler.StartSubmitResult(ctx, callback.From.ID)

	b.AnswerCallback(callback.ID, "")
	if keyboard != nil {
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text)
		b.SendMessageWithKeyboard(callback.Message.Chat.ID, text, *keyboard)
	} else {
		b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text)
	}
}

// handleInfoCallback обрабатывает запрос информации
func (b *Bot) handleInfoCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	// Получаем активное событие
	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("Error finding active event: %v", err)
		b.AnswerCallback(callback.ID, "Ошибка")
		return
	}

	if event == nil {
		b.AnswerCallback(callback.ID, "Нет активных событий")
		return
	}

	// Формируем сообщение с информацией о событии
	text := fmt.Sprintf(`ℹ️ Информация о событии

📌 %s

%s`, event.Name, event.Description)

	b.AnswerCallback(callback.ID, "")
	b.SendMessage(callback.Message.Chat.ID, text)
}

// handleStatefulCallback обрабатывает callback в зависимости от состояния сессии
func (b *Bot) handleStatefulCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) {
	userID := callback.From.ID
	state := b.sessionManager.GetState(userID)
	data := callback.Data

	switch state {
	case session.StateAwaitingBikeType:
		if strings.HasPrefix(data, "bike_") {
			bikeType := strings.TrimPrefix(data, "bike_")
			registrationHandler := handler.NewRegistrationHandler(
				b.sessionManager,
				b.eventRepo,
				b.participantRepo,
				b.registerParticipantHandler,
			)
			text, keyboard := registrationHandler.HandleBikeTypeSelection(ctx, userID, bikeType)
			b.AnswerCallback(callback.ID, "")
			b.EditMessage(callback.Message.Chat.ID, callback.Message.MessageID, text)
			if keyboard != nil {
				b.SendMessageWithKeyboard(callback.Message.Chat.ID, text, *keyboard)
			}
		}

	case session.StateAwaitingGender:
		if strings.HasPrefix(data, "gender_") {
			gender := strings.TrimPrefix(data, "gender_")
			registrationHandler := handler.NewRegistrationHandler(
				b.sessionManager,
				b.eventRepo,
				b.participantRepo,
				b.registerParticipantHandler,
			)
			text, err := registrationHandler.HandleGenderSelection(ctx, userID, gender)
			b.AnswerCallback(callback.ID, "")
			
			// Удаляем сообщение с кнопками выбора
			b.DeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
			
			// Получаем стартовую клавиатуру
			keyboard := b.getStartKeyboard(ctx)
			
			// Отправляем сообщение с результатом и стартовыми кнопками
			if err != nil {
				if keyboard != nil {
					b.SendMessageWithKeyboard(callback.Message.Chat.ID, text, *keyboard)
				} else {
					b.SendMessage(callback.Message.Chat.ID, text)
				}
			} else {
				if keyboard != nil {
					b.SendMessageWithKeyboard(callback.Message.Chat.ID, text, *keyboard)
				} else {
					b.SendMessage(callback.Message.Chat.ID, text)
				}
			}
		}

	case session.StateAwaitingGiftGender:
		if strings.HasPrefix(data, "gift_gender_") {
			gender := strings.TrimPrefix(data, "gift_gender_")
			giftHandler := handler.NewGiftHandler(
				b.sessionManager,
				b.eventRepo,
				b.addGiftHandler,
			)
			text, keyboard := giftHandler.HandleGiftGenderSelection(ctx, userID, gender)
			b.AnswerCallback(callback.ID, "")
			
			// Удаляем предыдущее сообщение
			b.DeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
			
			// Отправляем новое сообщение и сохраняем его ID
			if keyboard != nil {
				msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
				msg.ReplyMarkup = keyboard
				sentMsg, err := b.api.Send(msg)
				if err == nil {
					var messageIDs []int
					messageIDsRaw, ok := b.sessionManager.GetData(userID, "gift_message_ids")
					if ok {
						messageIDs = messageIDsRaw.([]int)
					}
					messageIDs = append(messageIDs, sentMsg.MessageID)
					b.sessionManager.SetData(userID, "gift_message_ids", messageIDs)
				}
			}
		}

	case session.StateAwaitingGiftBikeType:
		if strings.HasPrefix(data, "gift_bike_") {
			bikeType := strings.TrimPrefix(data, "gift_bike_")
			giftHandler := handler.NewGiftHandler(
				b.sessionManager,
				b.eventRepo,
				b.addGiftHandler,
			)
			text, keyboard := giftHandler.HandleGiftBikeTypeSelection(ctx, userID, bikeType)
			b.AnswerCallback(callback.ID, "")
			
			// Удаляем предыдущее сообщение
			b.DeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
			
			// Отправляем новое сообщение и сохраняем его ID
			if keyboard != nil {
				msg := tgbotapi.NewMessage(callback.Message.Chat.ID, text)
				msg.ReplyMarkup = keyboard
				sentMsg, err := b.api.Send(msg)
				if err == nil {
					var messageIDs []int
					messageIDsRaw, ok := b.sessionManager.GetData(userID, "gift_message_ids")
					if ok {
						messageIDs = messageIDsRaw.([]int)
					}
					messageIDs = append(messageIDs, sentMsg.MessageID)
					b.sessionManager.SetData(userID, "gift_message_ids", messageIDs)
				}
			}
		}

	case session.StateAwaitingGiftPhoto:
		if data == "finish_gift" || data == "skip_photos" {
			giftHandler := handler.NewGiftHandler(
				b.sessionManager,
				b.eventRepo,
				b.addGiftHandler,
			)
			text, err := giftHandler.FinishAddGift(ctx, userID)
			b.AnswerCallback(callback.ID, "")
			
			// Удаляем все промежуточные сообщения от бота (кроме фото)
			messageIDsRaw, ok := b.sessionManager.GetData(userID, "gift_message_ids")
			if ok {
				messageIDs := messageIDsRaw.([]int)
				for _, msgID := range messageIDs {
					b.DeleteMessage(callback.Message.Chat.ID, msgID)
				}
			}
			
			// Удаляем текущее сообщение с кнопками
			b.DeleteMessage(callback.Message.Chat.ID, callback.Message.MessageID)
			
			// Получаем стартовую клавиатуру
			keyboard := b.getStartKeyboard(ctx)
			
			// Отправляем финальное сообщение с благодарностью и стартовыми кнопками
			if err != nil {
				if keyboard != nil {
					b.SendMessageWithKeyboard(callback.Message.Chat.ID, text, *keyboard)
				} else {
					b.SendMessage(callback.Message.Chat.ID, text)
				}
			} else {
				if keyboard != nil {
					b.SendMessageWithKeyboard(callback.Message.Chat.ID, text, *keyboard)
				} else {
					b.SendMessage(callback.Message.Chat.ID, text)
				}
			}
		}
	}
}

// handleMessage обрабатывает обычные сообщения
func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	userID := msg.From.ID
	state := b.sessionManager.GetState(userID)

	switch state {
	case session.StateAwaitingGiftDesc:
		// Обрабатываем описание подарка
		giftHandler := handler.NewGiftHandler(
			b.sessionManager,
			b.eventRepo,
			b.addGiftHandler,
		)
		text, keyboard := giftHandler.HandleGiftDescription(ctx, userID, msg.Text)
		
		// Сохраняем ID сообщений от бота для последующего удаления (не сообщения пользователя!)
		var messageIDs []int
		messageIDsRaw, ok := b.sessionManager.GetData(userID, "gift_message_ids")
		if ok {
			messageIDs = messageIDsRaw.([]int)
		}
		
		// Отправляем новое сообщение и сохраняем его ID
		if keyboard != nil {
			msgToSend := tgbotapi.NewMessage(msg.Chat.ID, text)
			msgToSend.ReplyMarkup = keyboard
			sentMsg, err := b.api.Send(msgToSend)
			if err == nil {
				messageIDs = append(messageIDs, sentMsg.MessageID)
			}
		} else {
			sentMsg, err := b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
			if err == nil {
				messageIDs = append(messageIDs, sentMsg.MessageID)
			}
		}
		b.sessionManager.SetData(userID, "gift_message_ids", messageIDs)

	case session.StateAwaitingGiftPhoto:
		// Обрабатываем фото подарка
		if msg.Photo != nil && len(msg.Photo) > 0 {
			// Берём фото наибольшего размера
			photo := msg.Photo[len(msg.Photo)-1]
			giftHandler := handler.NewGiftHandler(
				b.sessionManager,
				b.eventRepo,
				b.addGiftHandler,
			)
			text := giftHandler.HandleGiftPhoto(userID, photo.FileID)
			
			// Отправляем подтверждение и сохраняем его ID для удаления
			sentMsg, err := b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
			if err == nil {
				var messageIDs []int
				messageIDsRaw, ok := b.sessionManager.GetData(userID, "gift_message_ids")
				if ok {
					messageIDs = messageIDsRaw.([]int)
				}
				messageIDs = append(messageIDs, sentMsg.MessageID)
				b.sessionManager.SetData(userID, "gift_message_ids", messageIDs)
			}
		}

	case session.StateAwaitingResultLink:
		// Обрабатываем ссылку на результат
		resultHandler := handler.NewResultHandler(
			b.sessionManager,
			b.eventRepo,
			b.participantRepo,
			b.submitResultHandler,
		)
		text, err := resultHandler.HandleResultLink(ctx, userID, msg.Text)
		if err != nil {
			b.SendMessage(msg.Chat.ID, text)
		} else {
			b.SendMessage(msg.Chat.ID, text)
		}

	default:
		// Если нет активного состояния, предлагаем использовать /start
		b.SendMessage(msg.Chat.ID, "Используйте /start для начала работы с ботом.")
	}
}

// getStartKeyboard возвращает стартовую клавиатуру с основными действиями
func (b *Bot) getStartKeyboard(ctx context.Context) *tgbotapi.InlineKeyboardMarkup {
	// Получаем активное событие
	event, err := b.eventRepo.FindActive(ctx)
	if err != nil || event == nil {
		return nil
	}

	// Создаём клавиатуру с действиями
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚴 Зарегистрироваться", "register"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Добавить подарок", "add_gift"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏁 Отправить результат", "submit_result"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Информация", "info"),
		),
	)

	return &keyboard
}
