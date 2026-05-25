package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/infrastructure/telegram/handler"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// handleCommand обрабатывает команды бота
func (b *Bot) handleCommand(ctx context.Context, msg *models.Message) {
	if msg == nil {
		log.Printf("Telegram command ignored: nil message")
		return
	}

	switch messageCommand(msg) {
	case "start":
		b.handleStartCommand(ctx, msg)
	default:
		if _, err := b.SendMessage(ctx, msg.Chat.ID, "Неизвестная команда. Используйте /start"); err != nil {
			return
		}
	}
}

// handleStartCommand обрабатывает команду /start
func (b *Bot) handleStartCommand(ctx context.Context, msg *models.Message) {
	startHandler := handler.NewStartHandler(b.userRepo, b.eventRepo, b.miniappURL)
	text, markup := startHandler.Handle(ctx, msg)

	if _, err := b.sendWithOptionalKeyboard(ctx, msg.Chat.ID, text, markup); err != nil {
		return
	}
}

// handleCallback обрабатывает callback-запросы (нажатия на inline кнопки)
func (b *Bot) handleCallback(ctx context.Context, callback *models.CallbackQuery) {
	if callback == nil {
		log.Printf("Telegram callback ignored: nil callback")
		return
	}

	userID := callback.From.ID
	data := callback.Data

	// Обрабатываем отмену
	if data == "cancel" {
		b.sessionManager.ResetState(userID)
		if err := b.AnswerCallback(ctx, callback.ID, "Отменено"); err != nil {
			return
		}

		msgRef, ok := callbackMessage(callback)
		if !ok {
			return
		}

		_ = b.EditMessage(ctx, msgRef.ChatID, msgRef.MessageID, "Действие отменено.")
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
func (b *Bot) handleRegisterCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	registrationHandler := handler.NewRegistrationHandler(
		b.sessionManager,
		b.eventRepo,
		b.participantRepo,
		b.registerParticipantHandler,
	)

	text, markup := registrationHandler.StartRegistration(ctx, callback.From.ID)

	if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
		return
	}

	if markup != nil {
		_ = b.EditMessage(ctx, msgRef.ChatID, msgRef.MessageID, text)
		_, _ = b.SendMessageWithKeyboard(ctx, msgRef.ChatID, text, *markup)
		return
	}

	_ = b.EditMessage(ctx, msgRef.ChatID, msgRef.MessageID, text)
}

// handleAddGiftCallback обрабатывает начало добавления подарка
func (b *Bot) handleAddGiftCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	giftHandler := handler.NewGiftHandler(
		b.sessionManager,
		b.eventRepo,
		b.addGiftHandler,
	)

	text, markup := giftHandler.StartAddGift(ctx, callback.From.ID)

	if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
		return
	}

	// Удаляем предыдущее сообщение
	_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)

	// Сохраняем ID нового сообщения для последующего удаления
	if markup != nil {
		sentMsg, err := b.SendMessageWithKeyboard(ctx, msgRef.ChatID, text, *markup)
		if err == nil && sentMsg != nil {
			b.setGiftMessageIDs(callback.From.ID, []int{sentMsg.ID})
		}
	}
}

// handleSubmitResultCallback обрабатывает начало отправки результата
func (b *Bot) handleSubmitResultCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	resultHandler := handler.NewResultHandler(
		b.sessionManager,
		b.eventRepo,
		b.participantRepo,
		b.submitResultHandler,
	)

	text, markup := resultHandler.StartSubmitResult(ctx, callback.From.ID)

	if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
		return
	}

	if markup != nil {
		_ = b.EditMessage(ctx, msgRef.ChatID, msgRef.MessageID, text)
		_, _ = b.SendMessageWithKeyboard(ctx, msgRef.ChatID, text, *markup)
		return
	}

	_ = b.EditMessage(ctx, msgRef.ChatID, msgRef.MessageID, text)
}

// handleInfoCallback обрабатывает запрос информации
func (b *Bot) handleInfoCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	// Получаем активное событие
	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("Error finding active event: %v", err)
		_ = b.AnswerCallback(ctx, callback.ID, "Ошибка")
		return
	}

	if event == nil {
		_ = b.AnswerCallback(ctx, callback.ID, "Нет активных событий")
		return
	}

	// Формируем сообщение с информацией о событии
	text := fmt.Sprintf(`ℹ️ Информация о событии

📌 %s

%s`, event.Name, event.Description)

	if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
		return
	}

	_, _ = b.SendMessage(ctx, msgRef.ChatID, text)
}

// handleStatefulCallback обрабатывает callback в зависимости от состояния сессии
func (b *Bot) handleStatefulCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

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
			text, markup := registrationHandler.HandleBikeTypeSelection(ctx, userID, bikeType)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}
			_ = b.EditMessage(ctx, msgRef.ChatID, msgRef.MessageID, text)
			if markup != nil {
				_, _ = b.SendMessageWithKeyboard(ctx, msgRef.ChatID, text, *markup)
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
			text, _ := registrationHandler.HandleGenderSelection(ctx, userID, gender)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}

			// Удаляем сообщение с кнопками выбора
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)

			// Отправляем сообщение с результатом и стартовыми кнопками
			_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, b.getStartKeyboard(ctx))
		}

	case session.StateAwaitingGiftGender:
		if strings.HasPrefix(data, "gift_gender_") {
			gender := strings.TrimPrefix(data, "gift_gender_")
			giftHandler := handler.NewGiftHandler(
				b.sessionManager,
				b.eventRepo,
				b.addGiftHandler,
			)
			text, markup := giftHandler.HandleGiftGenderSelection(ctx, userID, gender)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}

			// Удаляем предыдущее сообщение
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)

			// Отправляем новое сообщение и сохраняем его ID
			if markup != nil {
				sentMsg, err := b.SendMessageWithKeyboard(ctx, msgRef.ChatID, text, *markup)
				if err == nil && sentMsg != nil {
					b.appendGiftMessageID(userID, sentMsg.ID)
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
			text, markup := giftHandler.HandleGiftBikeTypeSelection(ctx, userID, bikeType)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}

			// Удаляем предыдущее сообщение
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)

			// Отправляем новое сообщение и сохраняем его ID
			if markup != nil {
				sentMsg, err := b.SendMessageWithKeyboard(ctx, msgRef.ChatID, text, *markup)
				if err == nil && sentMsg != nil {
					b.appendGiftMessageID(userID, sentMsg.ID)
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
			messageIDs := b.giftMessageIDs(userID)
			text, markup := giftHandler.PreviewGift(userID)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}

			// Удаляем все промежуточные сообщения от бота (кроме фото)
			for _, msgID := range messageIDs {
				if msgID == msgRef.MessageID {
					continue
				}
				_ = b.DeleteMessage(ctx, msgRef.ChatID, msgID)
			}

			// Удаляем текущее сообщение с кнопками
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)

			// Отправляем сводку с явным подтверждением перед сохранением.
			sentMsg, err := b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, markup)
			if err == nil && sentMsg != nil && markup != nil {
				b.setGiftMessageIDs(userID, []int{sentMsg.ID})
			}
		}

	case session.StateAwaitingGiftConfirmation:
		giftHandler := handler.NewGiftHandler(
			b.sessionManager,
			b.eventRepo,
			b.addGiftHandler,
		)

		switch data {
		case "confirm_gift":
			messageIDs := b.giftMessageIDs(userID)
			text, err := giftHandler.ConfirmAddGift(ctx, userID)
			if err != nil {
				_ = b.AnswerCallback(ctx, callback.ID, "Ошибка")
				_, _ = b.SendMessage(ctx, msgRef.ChatID, text+"\n\nДанные сохранены. Попробуйте подтвердить ещё раз или отмените добавление.")
				return
			}
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}
			for _, msgID := range messageIDs {
				if msgID == msgRef.MessageID {
					continue
				}
				_ = b.DeleteMessage(ctx, msgRef.ChatID, msgID)
			}
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)
			_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, b.getStartKeyboard(ctx))

		case "restart_gift":
			messageIDs := b.giftMessageIDs(userID)
			text, markup := giftHandler.RestartAddGift(ctx, userID)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}
			for _, msgID := range messageIDs {
				if msgID == msgRef.MessageID {
					continue
				}
				_ = b.DeleteMessage(ctx, msgRef.ChatID, msgID)
			}
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)
			sentMsg, err := b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, markup)
			if err == nil && sentMsg != nil && markup != nil {
				b.setGiftMessageIDs(userID, []int{sentMsg.ID})
			}

		default:
			b.logDebug("Unsupported gift confirmation callback: user_id=%d data=%s", userID, data)
		}

	default:
		b.logDebug("Unsupported Telegram callback state: user_id=%d state=%s data=%s", userID, state, data)
	}
}

// handleMessage обрабатывает обычные сообщения
func (b *Bot) handleMessage(ctx context.Context, msg *models.Message) {
	if msg == nil {
		log.Printf("Telegram message ignored: nil message")
		return
	}

	sender, ok := messageSender(msg)
	if !ok {
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Не удалось определить пользователя. Отправьте /start ещё раз.")
		return
	}

	userID := sender.ID
	state := b.sessionManager.GetState(userID)

	switch state {
	case session.StateAwaitingGiftDesc:
		// Обрабатываем описание подарка
		giftHandler := handler.NewGiftHandler(
			b.sessionManager,
			b.eventRepo,
			b.addGiftHandler,
		)
		text, markup := giftHandler.HandleGiftDescription(ctx, userID, msg.Text)

		// Отправляем новое сообщение и сохраняем его ID для последующего удаления (не сообщения пользователя!)
		sentMsg, err := b.sendWithOptionalKeyboard(ctx, msg.Chat.ID, text, markup)
		if err == nil && sentMsg != nil {
			b.appendGiftMessageID(userID, sentMsg.ID)
		}

	case session.StateAwaitingGiftPhoto:
		// Обрабатываем фото подарка
		if len(msg.Photo) > 0 {
			// Берём фото наибольшего размера
			photo := msg.Photo[len(msg.Photo)-1]
			giftHandler := handler.NewGiftHandler(
				b.sessionManager,
				b.eventRepo,
				b.addGiftHandler,
			)
			text := giftHandler.HandleGiftPhoto(userID, photo.FileID)

			// Отправляем подтверждение и сохраняем его ID для удаления
			sentMsg, err := b.SendMessage(ctx, msg.Chat.ID, text)
			if err == nil && sentMsg != nil {
				b.appendGiftMessageID(userID, sentMsg.ID)
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
		text, _ := resultHandler.HandleResultLink(ctx, userID, msg.Text)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, text)

	default:
		// Если нет активного состояния, предлагаем использовать /start
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Используйте /start для начала работы с ботом.")
	}
}

// getStartKeyboard возвращает стартовую клавиатуру с основными действиями
func (b *Bot) getStartKeyboard(ctx context.Context) *models.InlineKeyboardMarkup {
	// Получаем активное событие
	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("Error finding active event for start keyboard: %v", err)
		return nil
	}
	if event == nil {
		return nil
	}

	// Создаём клавиатуру с действиями
	markup := keyboard.MainMenu(b.miniappURL)
	return &markup
}

func (b *Bot) sendWithOptionalKeyboard(ctx context.Context, chatID int64, text string, markup *models.InlineKeyboardMarkup) (*models.Message, error) {
	if markup != nil {
		return b.SendMessageWithKeyboard(ctx, chatID, text, *markup)
	}

	return b.SendMessage(ctx, chatID, text)
}

func (b *Bot) setGiftMessageIDs(userID int64, messageIDs []int) {
	b.sessionManager.SetData(userID, "gift_message_ids", messageIDs)
}

func (b *Bot) appendGiftMessageID(userID int64, messageID int) {
	messageIDs := b.giftMessageIDs(userID)
	messageIDs = append(messageIDs, messageID)
	b.setGiftMessageIDs(userID, messageIDs)
}

func (b *Bot) giftMessageIDs(userID int64) []int {
	messageIDsRaw, ok := b.sessionManager.GetData(userID, "gift_message_ids")
	if !ok {
		return nil
	}

	messageIDs, ok := messageIDsRaw.([]int)
	if !ok {
		log.Printf("Invalid gift message IDs state: user_id=%d", userID)
		return nil
	}

	return messageIDs
}
