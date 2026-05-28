package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
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
	if !isPrivateTelegramChat(msg.Chat) {
		b.logDebug("Telegram command ignored in service chat: command=%s chat=%s", messageCommand(msg), b.chatLogMarker(msg.Chat.ID))
		return
	}

	switch messageCommand(msg) {
	case "start":
		b.handleStartCommand(ctx, msg)
	case "menu":
		b.handleMenuCommand(ctx, msg)
	default:
		if _, err := b.SendMessage(ctx, msg.Chat.ID, "Неизвестная команда. Используйте /start или /menu"); err != nil {
			return
		}
	}
}

// handleStartCommand обрабатывает команду /start
func (b *Bot) handleStartCommand(ctx context.Context, msg *models.Message) {
	startHandler := handler.NewStartHandler(b.userRepo, b.eventRepo, b.participantRepo, b.miniappURL)
	text, markup := startHandler.Handle(ctx, msg)

	if _, err := b.sendWithOptionalKeyboard(ctx, msg.Chat.ID, text, markup); err != nil {
		return
	}
}

// handleMenuCommand обрабатывает команду /menu
func (b *Bot) handleMenuCommand(ctx context.Context, msg *models.Message) {
	startHandler := handler.NewStartHandler(b.userRepo, b.eventRepo, b.participantRepo, b.miniappURL)
	text, markup := startHandler.Handle(ctx, msg)
	if markup != nil {
		text = "Главное меню:"
	}

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

	msgRef, hasMessage := callbackMessage(callback)
	if !hasMessage {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	userID := callback.From.ID
	data := callback.Data
	chatID := msgRef.ChatID

	if !isPrivateTelegramChatRef(msgRef) {
		b.logDebug("Telegram callback ignored in service chat: data=%s chat=%s", data, b.chatLogMarker(chatID))
		return
	}

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
	case "withdraw_participation":
		b.handleWithdrawParticipationCallback(ctx, callback)
	case "info":
		b.handleInfoCallback(ctx, callback)
	case "event_conditions":
		b.handleEventConditionsCallback(ctx, callback)
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
	if b.isPublicChat(msgRef.ChatID) {
		_ = b.AnswerCallback(ctx, callback.ID, "Откройте чат с ботом")
		_, _ = b.SendMessage(ctx, msgRef.ChatID, "Для регистрации откройте бота в личных сообщениях.")
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
	if b.isPublicChat(msgRef.ChatID) {
		_ = b.AnswerCallback(ctx, callback.ID, "Откройте чат с ботом")
		_, _ = b.SendMessage(ctx, msgRef.ChatID, "Для добавления приза откройте бота в личных сообщениях.")
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

	_, _ = b.replaceGiftControlMessage(ctx, callback.From.ID, msgRef.ChatID, text, markup, msgRef.MessageID)
}

// handleSubmitResultCallback обрабатывает начало отправки результата
func (b *Bot) handleSubmitResultCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}
	if b.isPublicChat(msgRef.ChatID) {
		_ = b.AnswerCallback(ctx, callback.ID, "Откройте чат с ботом")
		_, _ = b.SendMessage(ctx, msgRef.ChatID, "Для отправки результата откройте бота в личных сообщениях.")
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
	b.handleEventConditionsCallback(ctx, callback)
}

// handleEventConditionsCallback обрабатывает условия участия.
func (b *Bot) handleEventConditionsCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("WARN Failed to load active event for conditions callback: user_id=%d error=%v", callback.From.ID, err)
		_ = b.AnswerCallback(ctx, callback.ID, "Ошибка")
		return
	}

	if event == nil {
		_ = b.AnswerCallback(ctx, callback.ID, "Нет активных событий")
		return
	}
	log.Printf("INFO Event conditions requested: telegram_user_id=%d event_id=%d", callback.From.ID, event.ID)

	text := b.eventConditionsText(event)

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
			_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, b.getStartKeyboard(ctx, userID))
		}

	case session.StateAwaitingGiftGender:
		giftHandler := handler.NewGiftHandler(
			b.sessionManager,
			b.eventRepo,
			b.addGiftHandler,
		)
		if strings.HasPrefix(data, "gift_gender_") {
			gender := strings.TrimPrefix(data, "gift_gender_")
			text, markup := giftHandler.HandleGiftGenderSelection(ctx, userID, gender)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}

			_, _ = b.replaceGiftControlMessage(ctx, userID, msgRef.ChatID, text, markup, msgRef.MessageID)
			return
		}
		if data == "restart_gift" {
			b.handleGiftRestartCallback(ctx, callback, msgRef, userID, giftHandler)
			return
		}
		if isGiftFlowCallback(data) {
			log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
			text, markup := giftHandler.GiftGenderPrompt(userID)
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Продолжите добавление", text, markup)
		}

	case session.StateAwaitingGiftBikeType:
		giftHandler := handler.NewGiftHandler(
			b.sessionManager,
			b.eventRepo,
			b.addGiftHandler,
		)
		if strings.HasPrefix(data, "gift_bike_") {
			bikeType := strings.TrimPrefix(data, "gift_bike_")
			text, markup := giftHandler.HandleGiftBikeTypeSelection(ctx, userID, bikeType)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}

			_, _ = b.replaceGiftControlMessage(ctx, userID, msgRef.ChatID, text, markup, msgRef.MessageID)
			return
		}
		if data == "restart_gift" {
			b.handleGiftRestartCallback(ctx, callback, msgRef, userID, giftHandler)
			return
		}
		if isGiftFlowCallback(data) {
			log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
			text, markup := giftHandler.GiftBikeTypePrompt(userID)
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Продолжите добавление", text, markup)
		}

	case session.StateAwaitingGiftDesc:
		giftHandler := handler.NewGiftHandler(
			b.sessionManager,
			b.eventRepo,
			b.addGiftHandler,
		)
		switch data {
		case "finish_gift", "skip_photos":
			log.Printf("WARN Gift finish rejected: user_id=%d state=%s callback_data=%s missing_key=gift_description", userID, state, data)
			text, markup := giftHandler.GiftDraftPrompt(userID)
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Добавьте описание", text, markup)
		case "confirm_gift":
			log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
			text, markup := giftHandler.GiftDraftPrompt(userID)
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Сначала проверьте приз", text, markup)
		case "restart_gift":
			b.handleGiftRestartCallback(ctx, callback, msgRef, userID, giftHandler)
		default:
			if isGiftFlowCallback(data) {
				log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
				text, markup := giftHandler.GiftDraftPrompt(userID)
				b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Продолжите добавление", text, markup)
			}
		}

	case session.StateAwaitingGiftPhoto:
		giftHandler := handler.NewGiftHandler(
			b.sessionManager,
			b.eventRepo,
			b.addGiftHandler,
		)
		switch data {
		case "finish_gift", "skip_photos":
			b.handleGiftFinishCallback(ctx, callback, msgRef, userID, giftHandler)
		case "confirm_gift":
			log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
			text, markup := giftHandler.GiftDraftPrompt(userID)
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Сначала проверьте приз", text, markup)
		case "restart_gift":
			b.handleGiftRestartCallback(ctx, callback, msgRef, userID, giftHandler)
		default:
			if isGiftFlowCallback(data) {
				log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
				text, markup := giftHandler.GiftDraftPrompt(userID)
				b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Продолжите добавление", text, markup)
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
			gift, text, err := giftHandler.ConfirmAddGift(ctx, userID)
			if err != nil {
				_ = b.AnswerCallback(ctx, callback.ID, "Ошибка")
				if text != "" {
					_, _ = b.SendMessage(ctx, msgRef.ChatID, text)
				}
				return
			}
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}
			if gift != nil {
				if notifyErr := b.notifyAdminAboutGift(ctx, gift); notifyErr != nil {
					log.Printf("WARN Failed to notify admin about gift submission: user_id=%d gift_id=%d error=%v", userID, gift.ID, notifyErr)
				}
			}
			for _, msgID := range messageIDs {
				if msgID == msgRef.MessageID {
					continue
				}
				_ = b.DeleteMessage(ctx, msgRef.ChatID, msgID)
			}
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)
			_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, b.getStartKeyboard(ctx, userID))

		case "restart_gift":
			b.handleGiftRestartCallback(ctx, callback, msgRef, userID, giftHandler)

		default:
			if isGiftFlowCallback(data) {
				log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
				text, markup := giftHandler.GiftConfirmationPrompt(userID)
				b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Подтвердите приз", text, markup)
				return
			}
			b.logDebug("Unsupported gift confirmation callback: user_id=%d data=%s", userID, data)
		}

	default:
		if isGiftFlowCallback(data) {
			log.Printf("INFO Gift stale callback ignored: user_id=%d callback_data=%s state=%s", userID, data, state)
			_ = b.AnswerCallback(ctx, callback.ID, "Сначала откройте меню")
			return
		}
		b.logDebug("Unsupported Telegram callback state: user_id=%d state=%s data=%s", userID, state, data)
	}
}

func (b *Bot) handleGiftFinishCallback(ctx context.Context, callback *models.CallbackQuery, msgRef callbackMessageRef, userID int64, giftHandler *handler.GiftHandler) {
	data := callback.Data
	if missingKey, missing := giftHandler.GiftDraftMissingRequiredKey(userID); missing {
		log.Printf("WARN Gift finish rejected: user_id=%d state=%s callback_data=%s missing_key=%s", userID, b.sessionManager.GetState(userID), data, missingKey)
		text, markup := giftHandler.GiftDraftPrompt(userID)
		b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, "Добавьте описание", text, markup)
		return
	}

	text, markup := giftHandler.PreviewGift(userID)
	if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
		return
	}
	_, _ = b.replaceGiftControlMessage(ctx, userID, msgRef.ChatID, text, markup, msgRef.MessageID)
}

func (b *Bot) handleGiftRestartCallback(ctx context.Context, callback *models.CallbackQuery, msgRef callbackMessageRef, userID int64, giftHandler *handler.GiftHandler) {
	text, markup := giftHandler.RestartAddGift(ctx, userID)
	if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
		return
	}
	_, _ = b.replaceGiftControlMessage(ctx, userID, msgRef.ChatID, text, markup, msgRef.MessageID)
}

func (b *Bot) answerAndReplaceGiftControl(ctx context.Context, callback *models.CallbackQuery, msgRef callbackMessageRef, userID int64, answerText string, text string, markup *models.InlineKeyboardMarkup) {
	if err := b.AnswerCallback(ctx, callback.ID, answerText); err != nil {
		return
	}
	_, _ = b.replaceGiftControlMessage(ctx, userID, msgRef.ChatID, text, markup, msgRef.MessageID)
}

func isGiftFlowCallback(data string) bool {
	switch data {
	case "finish_gift", "skip_photos", "confirm_gift", "restart_gift":
		return true
	default:
		return strings.HasPrefix(data, "gift_gender_") || strings.HasPrefix(data, "gift_bike_")
	}
}

// handleMessage обрабатывает обычные сообщения
func (b *Bot) handleMessage(ctx context.Context, msg *models.Message) {
	if msg == nil {
		log.Printf("Telegram message ignored: nil message")
		return
	}
	if !isPrivateTelegramChat(msg.Chat) {
		b.logDebug("Telegram message ignored in service chat: chat=%s kind=%s", b.chatLogMarker(msg.Chat.ID), messageUpdateKind(msg))
		return
	}
	if len(msg.NewChatMembers) > 0 {
		b.handleNewChatMembers(ctx, msg)
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
	case session.StateAwaitingGiftGender,
		session.StateAwaitingGiftBikeType,
		session.StateAwaitingGiftDesc,
		session.StateAwaitingGiftPhoto,
		session.StateAwaitingGiftConfirmation:
		b.handleGiftMessage(ctx, msg, userID, state)

	case session.StateAwaitingResultLink:
		resultLink, ok := resultLinkText(msg)
		if !ok {
			participantID := b.resultSessionUint(userID, "participant_id")
			eventID := b.resultSessionUint(userID, "event_id")
			log.Printf(
				"INFO Invalid result submission input: user_id=%d participant_id=%d event_id=%d update_kind=%s reason=missing_text_link",
				userID,
				participantID,
				eventID,
				messageUpdateKind(msg),
			)
			_, _ = b.SendMessage(ctx, msg.Chat.ID, handler.ResultLinkInvalidInputText(b.resultTelegramTexts(userID)))
			return
		}

		// Обрабатываем ссылку на результат
		resultHandler := handler.NewResultHandler(
			b.sessionManager,
			b.eventRepo,
			b.participantRepo,
			b.submitResultHandler,
		)
		text, _ := resultHandler.HandleResultLink(ctx, userID, resultLink)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, text)

	default:
		// Если нет активного состояния, предлагаем использовать /start
		if b.publicChatID == 0 || msg.Chat.ID != b.publicChatID {
			_, _ = b.SendMessage(ctx, msg.Chat.ID, "Используйте /start для начала работы с ботом.")
		}
	}
}

func (b *Bot) handleNewChatMembers(ctx context.Context, msg *models.Message) {
	if msg == nil {
		return
	}
	if !isPrivateTelegramChat(msg.Chat) {
		b.logDebug("Telegram new chat members ignored outside private chat: chat=%s", b.chatLogMarker(msg.Chat.ID))
		return
	}

	if b.publicChatID == 0 || msg.Chat.ID != b.publicChatID {
		return
	}

	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("WARN Failed to load active event for public chat welcome: chat=public error=%v", err)
		return
	}
	if event == nil {
		log.Printf("WARN Public chat welcome skipped: chat=public reason=no_active_event")
		return
	}

	for _, member := range msg.NewChatMembers {
		if member.IsBot || member.ID == 0 {
			continue
		}

		isBlacklisted, err := b.isUserBlacklistedHandler.Handle(ctx, query.IsUserBlacklistedQuery{TelegramUserID: member.ID})
		if err != nil {
			log.Printf("WARN Public chat welcome skipped: telegram_user_id=%d chat=public operation=blacklist_check error=%v", member.ID, err)
			continue
		}
		if isBlacklisted {
			log.Printf("INFO Public chat welcome skipped: telegram_user_id=%d chat=public reason=blacklisted", member.ID)
			continue
		}

		user := &entity.User{
			ID:        member.ID,
			Username:  member.Username,
			FirstName: member.FirstName,
			LastName:  member.LastName,
		}
		if _, err := b.userRepo.FindByID(ctx, user.ID); err != nil {
			if createErr := b.userRepo.Create(ctx, user); createErr != nil {
				log.Printf("WARN Public chat welcome skipped: telegram_user_id=%d chat=public reason=user_create_failed error=%v", member.ID, createErr)
				continue
			}
		}

		firstName := strings.TrimSpace(member.FirstName)
		if firstName == "" {
			firstName = "друг"
		}

		text := fmt.Sprintf("👋 Привет, %s! Добро пожаловать в %s 🚴", firstName, event.Name)
		markup := keyboard.PublicMenu(
			b.miniappURL,
			b.deepLink("register"),
			b.deepLink("conditions"),
		)

		if _, err := b.SendMessageWithKeyboard(ctx, msg.Chat.ID, text, markup); err != nil {
			log.Printf("WARN Public chat welcome failed: telegram_user_id=%d chat=public event_id=%d error=%v", member.ID, event.ID, err)
			continue
		}

		log.Printf("INFO Public chat welcome sent: telegram_user_id=%d event_id=%d chat=public", member.ID, event.ID)
	}
}

func (b *Bot) handleWithdrawParticipationCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	userID := callback.From.ID
	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("WARN Failed to load active event for withdrawal: user_id=%d error=%v", userID, err)
		_ = b.AnswerCallback(ctx, callback.ID, "Ошибка")
		_, _ = b.SendMessage(ctx, msgRef.ChatID, "Не удалось обработать запрос на выход. Попробуйте позже.")
		return
	}
	if event == nil {
		_ = b.AnswerCallback(ctx, callback.ID, "Нет активных событий")
		_, _ = b.SendMessage(ctx, msgRef.ChatID, "В данный момент нет активных событий.")
		return
	}

	if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
		return
	}

	cmd := command.WithdrawParticipantCommand{UserID: userID, EventID: event.ID}
	_, err = b.withdrawParticipantHandler.Handle(ctx, cmd)
	if err != nil {
		if errors.Is(err, command.ErrParticipantNotFound) {
			_, _ = b.SendMessage(ctx, msgRef.ChatID, "Вы не были зарегистрированы на это событие.")
			_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, "Главное меню:", b.getStartKeyboard(ctx, userID))
			return
		}

		log.Printf("WARN Failed to withdraw participant: telegram_user_id=%d event_id=%d error=%v", userID, event.ID, err)
		_, _ = b.SendMessage(ctx, msgRef.ChatID, "Не удалось отменить участие. Попробуйте позже.")
		return
	}

	_, _ = b.SendMessage(ctx, msgRef.ChatID, "Вы больше не участвуете в текущем соревновании.")
	_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, "Главное меню:", b.getStartKeyboard(ctx, userID))
}

func (b *Bot) resultSessionUint(userID int64, key string) uint {
	value, ok := b.sessionManager.GetData(userID, key)
	if !ok {
		return 0
	}

	typedValue, ok := value.(uint)
	if !ok {
		log.Printf("WARN Invalid result session data: user_id=%d key=%s type=%T", userID, key, value)
		return 0
	}

	return typedValue
}

func (b *Bot) resultTelegramTexts(userID int64) entity.EventTelegramTexts {
	textsRaw, ok := b.sessionManager.GetData(userID, "event_telegram_texts")
	if !ok {
		return entity.NormalizeEventTelegramTexts(entity.EventTelegramTexts{})
	}

	texts, ok := textsRaw.(entity.EventTelegramTexts)
	if !ok {
		log.Printf("WARN Invalid result session data: user_id=%d key=event_telegram_texts type=%T", userID, textsRaw)
		return entity.NormalizeEventTelegramTexts(entity.EventTelegramTexts{})
	}

	return entity.NormalizeEventTelegramTexts(texts)
}

func (b *Bot) handleGiftMessage(ctx context.Context, msg *models.Message, userID int64, state session.SessionState) {
	giftHandler := handler.NewGiftHandler(
		b.sessionManager,
		b.eventRepo,
		b.addGiftHandler,
	)
	mediaGroupID := ""
	chatID := int64(0)
	if msg != nil {
		mediaGroupID = msg.MediaGroupID
		chatID = msg.Chat.ID
	}

	mediaGroupAlreadyReplied := b.giftMediaGroupAlreadyReplied(userID, state, msg)
	action := giftMessageAction(state, msg, mediaGroupAlreadyReplied)
	if action.OutOfOrder {
		b.logDebug(
			"Gift flow out-of-order input: user_id=%d state=%s update_kind=%s media_group_id=%s",
			userID,
			state,
			action.UpdateKind,
			mediaGroupID,
		)
	}
	if action.MissingInput {
		log.Printf(
			"Gift flow input missing expected content: user_id=%d state=%s update_kind=%s media_group_id=%s",
			userID,
			state,
			action.UpdateKind,
			mediaGroupID,
		)
	}
	var replyText string
	var replyMarkup *models.InlineKeyboardMarkup
	if action.ProcessDescription {
		replyText, replyMarkup = giftHandler.HandleGiftDescription(ctx, userID, action.Description)
	}

	photoCount := 0
	if action.ProcessPhoto {
		photoCount = giftHandler.AppendGiftPhoto(userID, action.PhotoFileID)
		b.logDebug(
			"Gift photo message processed: user_id=%d state=%s update_kind=%s media_group_id=%s photo_count=%d",
			userID,
			state,
			action.UpdateKind,
			mediaGroupID,
			photoCount,
		)
	}

	if action.SuppressReply {
		return
	}

	switch action.Reply {
	case giftMessageReplyGiftGenderStep:
		replyText, replyMarkup = giftHandler.GiftGenderPrompt(userID)
	case giftMessageReplyGiftBikeStep:
		replyText, replyMarkup = giftHandler.GiftBikeTypePrompt(userID)
	case giftMessageReplyGiftDescriptionStep:
		replyText, replyMarkup = giftHandler.GiftDescriptionPrompt(userID)
	case giftMessageReplyGiftPhotoStep:
		if replyText == "" {
			replyText, replyMarkup = giftHandler.GiftPhotoPrompt(userID)
		}
	case giftMessageReplyGiftPhotoAdded:
		replyText = giftHandler.GiftPhotoAddedText(userID, photoCount)
		replyMarkup = nil
	case giftMessageReplyGiftDraft:
		replyText, replyMarkup = giftHandler.GiftDraftPrompt(userID)
	case giftMessageReplyGiftConfirmationStep:
		replyText, replyMarkup = giftHandler.GiftConfirmationPrompt(userID)
	case giftMessageReplyNone:
		return
	}

	if replyText == "" {
		b.logDebug("Gift flow response skipped: user_id=%d state=%s reason=empty_reply", userID, state)
		return
	}

	if chatID == 0 {
		log.Printf("Gift flow response skipped: user_id=%d state=%s reason=missing_chat", userID, state)
		return
	}

	_, _ = b.replaceGiftControlMessage(ctx, userID, chatID, replyText, replyMarkup)
}

// getStartKeyboard возвращает стартовую клавиатуру с основными действиями
func (b *Bot) getStartKeyboard(ctx context.Context, userID int64) *models.InlineKeyboardMarkup {
	// Получаем активное событие
	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("WARN Failed to load active event for start keyboard: user_id=%d error=%v", userID, err)
		return nil
	}
	if event == nil {
		return nil
	}

	isRegistered := false
	if b.participantRepo != nil {
		participant, err := b.participantRepo.FindByUserAndEvent(ctx, userID, event.ID)
		if err != nil && !errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf("WARN Failed to load participant status for start menu: user_id=%d event_id=%d error=%v", userID, event.ID, err)
		} else if participant != nil {
			isRegistered = true
		}
	}

	// Создаём клавиатуру с действиями
	markup := keyboard.MainMenu(true, isRegistered, b.miniappURL, nil)
	return &markup
}

func (b *Bot) isPublicChat(chatID int64) bool {
	return b.publicChatID != 0 && chatID == b.publicChatID
}

func (b *Bot) isAdminChat(chatID int64) bool {
	return b.adminChatID != 0 && chatID == b.adminChatID
}

func (b *Bot) botUsernameAlias() string {
	if strings.TrimSpace(b.botUsername) == "" {
		return ""
	}

	return strings.TrimPrefix(b.botUsername, "@")
}

func (b *Bot) eventConditionsText(event *entity.Event) string {
	return handler.EventConditionsText(event)
}

func (b *Bot) deepLink(payload string) string {
	if strings.TrimSpace(payload) == "" {
		return ""
	}

	if strings.TrimSpace(b.botUsername) == "" {
		return ""
	}

	username := strings.TrimPrefix(b.botUsername, "@")
	if username == "" {
		return ""
	}

	return fmt.Sprintf("https://t.me/%s?start=%s", username, url.QueryEscape(payload))
}

func (b *Bot) sendWithOptionalKeyboard(ctx context.Context, chatID int64, text string, markup *models.InlineKeyboardMarkup) (*models.Message, error) {
	if markup != nil {
		return b.SendMessageWithKeyboard(ctx, chatID, text, *markup)
	}

	return b.SendMessage(ctx, chatID, text)
}

func (b *Bot) replaceGiftControlMessage(ctx context.Context, userID int64, chatID int64, text string, markup *models.InlineKeyboardMarkup, extraDeleteMessageIDs ...int) (*models.Message, error) {
	b.deleteGiftControlMessages(ctx, userID, chatID, extraDeleteMessageIDs...)

	sentMsg, err := b.sendWithOptionalKeyboard(ctx, chatID, text, markup)
	if err != nil {
		return nil, err
	}
	if sentMsg != nil && markup != nil {
		b.setGiftMessageIDs(userID, []int{sentMsg.ID})
		return sentMsg, nil
	}

	b.setGiftMessageIDs(userID, []int{})
	return sentMsg, nil
}

func (b *Bot) deleteGiftControlMessages(ctx context.Context, userID int64, chatID int64, extraMessageIDs ...int) {
	messageIDs := append([]int{}, b.giftMessageIDs(userID)...)
	messageIDs = append(messageIDs, extraMessageIDs...)
	seen := make(map[int]struct{}, len(messageIDs))
	for _, messageID := range messageIDs {
		if messageID <= 0 {
			continue
		}
		if _, ok := seen[messageID]; ok {
			continue
		}
		seen[messageID] = struct{}{}
		_ = b.DeleteMessage(ctx, chatID, messageID)
	}

	b.setGiftMessageIDs(userID, []int{})
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
		log.Printf("WARN Invalid gift message IDs state: user_id=%d state=%s", userID, b.sessionManager.GetState(userID))
		return nil
	}

	return messageIDs
}

func (b *Bot) giftMediaGroupAlreadyReplied(userID int64, state session.SessionState, msg *models.Message) bool {
	if msg == nil || strings.TrimSpace(msg.MediaGroupID) == "" {
		return false
	}

	current := giftMediaGroupReplyState{
		MediaGroupID:   msg.MediaGroupID,
		ChatID:         msg.Chat.ID,
		FirstMessageID: msg.ID,
		State:          state,
	}

	const key = "gift_media_group_reply"
	replyStateRaw, ok := b.sessionManager.GetData(userID, key)
	if !ok {
		b.sessionManager.SetData(userID, key, current)
		return false
	}

	replyState, ok := replyStateRaw.(giftMediaGroupReplyState)
	if !ok {
		log.Printf("Invalid gift media group reply state: user_id=%d state=%s key=%s type=%T", userID, state, key, replyStateRaw)
		b.sessionManager.SetData(userID, key, current)
		return false
	}

	if replyState.MediaGroupID == current.MediaGroupID && replyState.ChatID == current.ChatID {
		b.logDebug(
			"Gift media group response suppressed: user_id=%d state=%s media_group_id=%s",
			userID,
			state,
			msg.MediaGroupID,
		)
		return true
	}

	b.sessionManager.SetData(userID, key, current)
	return false
}
