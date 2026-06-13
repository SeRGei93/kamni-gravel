package telegram

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/handler"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
	"gravel_bot/internal/infrastructure/telegram/session"
)

const proxyStartUserChatUsage = "Используйте /start_user_chat <telegram_user_id>"

func (b *Bot) handleProxyUpdate(ctx context.Context, update *models.Update) bool {
	if update == nil {
		return false
	}

	if update.CallbackQuery != nil {
		if msgRef, ok := callbackMessage(update.CallbackQuery); ok && b.isBotMessagesChat(msgRef.ChatID) {
			expiredTargetUserID := b.notifyExpiredProxyDialog(ctx, msgRef.ChatID)
			if update.CallbackQuery.Data == proxyCloseCallbackData && expiredTargetUserID != 0 {
				_ = b.AnswerCallback(ctx, update.CallbackQuery.ID, "Чат закрыт")
				return true
			}
		}

		switch {
		case update.CallbackQuery.Data == proxyCloseCallbackData:
			b.handleProxyEndUserChatCallback(ctx, update.CallbackQuery)
			return true
		case strings.HasPrefix(update.CallbackQuery.Data, proxyOpenCallbackPrefix):
			b.handleProxyOpenUserChatCallback(ctx, update.CallbackQuery)
			return true
		}
	}

	if update.Message == nil {
		return false
	}

	command := messageCommand(update.Message)
	if b.isBotMessagesChat(update.Message.Chat.ID) {
		expiredTargetUserID := b.notifyExpiredProxyDialog(ctx, update.Message.Chat.ID)
		if command != "" {
			if command == proxyEndUserChatCommand && expiredTargetUserID != 0 {
				return true
			}
			if isProxyCommand(command) {
				b.handleProxyCommand(ctx, update.Message, command)
				return true
			}
			b.logDebug("Telegram proxy chat command ignored: command=%s chat=%s", command, b.chatLogMarker(update.Message.Chat.ID))
			return true
		}

		b.handleProxyMessage(ctx, update.Message)
		return true
	}

	if !isProxyCommand(command) {
		return false
	}

	b.logDebug("Telegram proxy command ignored outside proxy chat: command=%s chat=%s", command, b.chatLogMarker(update.Message.Chat.ID))
	return true
}

func isProxyCommand(command string) bool {
	return command == proxyStartUserChatCommand || command == proxyEndUserChatCommand || command == proxyBroadcastCommand
}

func (b *Bot) handleProxyCommand(ctx context.Context, msg *models.Message, command string) {
	if msg == nil {
		log.Printf("Telegram proxy command ignored: nil message")
		return
	}

	switch command {
	case proxyStartUserChatCommand:
		b.handleStartUserChatCommand(ctx, msg)
	case proxyEndUserChatCommand:
		b.handleEndUserChatCommand(ctx, msg.Chat.ID)
	case proxyBroadcastCommand:
		b.handleBroadcastParticipantsCommand(ctx, msg)
	default:
		b.logDebug("Unsupported Telegram proxy command: command=%s chat=%s", command, b.chatLogMarker(msg.Chat.ID))
	}
}

func (b *Bot) handleStartUserChatCommand(ctx context.Context, msg *models.Message) {
	targetUserID, reason, ok := parseStartUserChatTarget(msg)
	if !ok {
		log.Printf("INFO Telegram proxy command rejected: command=%s reason=%s chat=%s", proxyStartUserChatCommand, reason, b.chatLogMarker(msg.Chat.ID))
		_, _ = b.SendMessage(ctx, msg.Chat.ID, proxyStartUserChatUsage)
		return
	}

	b.openProxyDialog(ctx, msg.Chat.ID, targetUserID, proxyStartUserChatCommand)
}

func (b *Bot) openProxyDialog(ctx context.Context, chatID int64, targetUserID int64, source string) {
	activeTargetUserID, started := b.startProxyDialog(targetUserID)
	if !started {
		_, _ = b.SendMessageWithKeyboard(
			ctx,
			chatID,
			fmt.Sprintf("Чат уже открыт с %s. Закройте текущий чат командой /end_user_chat или кнопкой.", b.proxyUserLabel(ctx, activeTargetUserID)),
			proxyCloseKeyboard(),
		)
		return
	}

	log.Printf("INFO Telegram proxy dialog routed: source=%s chat=%s target_user_id=%d", source, b.chatLogMarker(chatID), targetUserID)
	_, _ = b.SendMessageWithKeyboard(
		ctx,
		chatID,
		fmt.Sprintf("Чат открыт с %s.\nВсе обычные сообщения в этом чате будут отправлены пользователю.", b.proxyUserLabel(ctx, targetUserID)),
		proxyCloseKeyboard(),
	)
}

func (b *Bot) handleEndUserChatCommand(ctx context.Context, chatID int64) {
	targetUserID, closed := b.endProxyDialog()
	if !closed {
		_, _ = b.SendMessage(ctx, chatID, "Активный чат не открыт.")
		return
	}

	log.Printf("INFO Telegram proxy command routed: command=%s chat=%s target_user_id=%d", proxyEndUserChatCommand, b.chatLogMarker(chatID), targetUserID)
	_, _ = b.SendMessage(ctx, chatID, fmt.Sprintf("Чат с %s закрыт.", b.proxyUserLabel(ctx, targetUserID)))
}

func (b *Bot) handleBroadcastParticipantsCommand(ctx context.Context, msg *models.Message) {
	if msg == nil {
		log.Printf("Telegram proxy broadcast command ignored: nil message")
		return
	}

	text, reason, ok := parseBroadcastParticipantsText(msg)
	if !ok {
		log.Printf("INFO Telegram proxy broadcast command rejected: reason=%s chat=%s", reason, b.chatLogMarker(msg.Chat.ID))
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Используйте /broadcast_participants <текст сообщения>")
		return
	}

	b.handleProxyBroadcastText(ctx, msg.Chat.ID, text)
}

func (b *Bot) handleProxyEndUserChatCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	if !b.isBotMessagesChat(msgRef.ChatID) {
		b.logDebug("Telegram proxy callback ignored outside proxy chat: data=%s chat=%s", callback.Data, b.chatLogMarker(msgRef.ChatID))
		_ = b.AnswerCallback(ctx, callback.ID, "Недоступно")
		return
	}

	targetUserID, closed := b.endProxyDialog()
	if !closed {
		_ = b.AnswerCallback(ctx, callback.ID, "Активный чат не открыт")
		return
	}

	log.Printf("INFO Telegram proxy callback routed: data=%s chat=%s target_user_id=%d", callback.Data, b.chatLogMarker(msgRef.ChatID), targetUserID)
	_ = b.AnswerCallback(ctx, callback.ID, "Чат закрыт")
	_, _ = b.SendMessage(ctx, msgRef.ChatID, fmt.Sprintf("Чат с %s закрыт.", b.proxyUserLabel(ctx, targetUserID)))
}

func (b *Bot) handleProxyOpenUserChatCallback(ctx context.Context, callback *models.CallbackQuery) {
	msgRef, ok := callbackMessage(callback)
	if !ok {
		_ = b.AnswerCallback(ctx, callback.ID, "Сообщение недоступно")
		return
	}

	if !b.isBotMessagesChat(msgRef.ChatID) {
		b.logDebug("Telegram proxy callback ignored outside proxy chat: data=%s chat=%s", callback.Data, b.chatLogMarker(msgRef.ChatID))
		_ = b.AnswerCallback(ctx, callback.ID, "Недоступно")
		return
	}

	targetUserID, reason, ok := parseProxyOpenUserChatCallback(callback.Data)
	if !ok {
		log.Printf("INFO Telegram proxy callback rejected: data=%s reason=%s chat=%s", callback.Data, reason, b.chatLogMarker(msgRef.ChatID))
		_ = b.AnswerCallback(ctx, callback.ID, "Недоступно")
		return
	}

	activeTargetUserID, started := b.startProxyDialog(targetUserID)
	if !started {
		_ = b.AnswerCallback(ctx, callback.ID, "Чат уже открыт")
		_, _ = b.SendMessageWithKeyboard(
			ctx,
			msgRef.ChatID,
			fmt.Sprintf("Чат уже открыт с %s. Закройте текущий чат командой /end_user_chat или кнопкой.", b.proxyUserLabel(ctx, activeTargetUserID)),
			proxyCloseKeyboard(),
		)
		return
	}

	log.Printf("INFO Telegram proxy callback routed: data=%s chat=%s target_user_id=%d", callback.Data, b.chatLogMarker(msgRef.ChatID), targetUserID)
	_ = b.AnswerCallback(ctx, callback.ID, "Чат открыт")
	_, _ = b.SendMessageWithKeyboard(
		ctx,
		msgRef.ChatID,
		fmt.Sprintf("Чат открыт с %s.\nВсе обычные сообщения в этом чате будут отправлены пользователю.", b.proxyUserLabel(ctx, targetUserID)),
		proxyCloseKeyboard(),
	)
}

func (b *Bot) handleProxyMessage(ctx context.Context, msg *models.Message) {
	if msg == nil {
		log.Printf("Telegram proxy message ignored: nil message")
		return
	}

	targetUserID, ok := b.activeProxyTarget()
	if !ok {
		b.logDebug("Telegram proxy message ignored without active dialog: chat=%s update_kind=%s", b.chatLogMarker(msg.Chat.ID), messageUpdateKind(msg))
		return
	}

	updateKind := messageUpdateKind(msg)
	b.touchProxyDialog(targetUserID)
	log.Printf("INFO Telegram proxy message delivery started: chat=%s target_user_id=%d update_kind=%s", b.chatLogMarker(msg.Chat.ID), targetUserID, updateKind)
	if _, err := b.api.CopyMessage(ctx, &telegrambot.CopyMessageParams{
		ChatID:     targetUserID,
		FromChatID: msg.Chat.ID,
		MessageID:  msg.ID,
	}); err != nil {
		log.Printf("ERROR Telegram proxy message delivery failed: chat=%s target_user_id=%d update_kind=%s error=%v", b.chatLogMarker(msg.Chat.ID), targetUserID, updateKind, err)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Не удалось отправить сообщение пользователю user_id=%d.", targetUserID))
		return
	}

	log.Printf("INFO Telegram proxy message delivered: chat=%s target_user_id=%d update_kind=%s", b.chatLogMarker(msg.Chat.ID), targetUserID, updateKind)
}

func (b *Bot) handleProxyBroadcastText(ctx context.Context, proxyChatID int64, text string) {
	log.Printf("INFO Telegram proxy broadcast delivery started: chat=%s text_len=%d", b.chatLogMarker(proxyChatID), len(text))
	recipients, err := b.proxyBroadcastRecipients(ctx)
	if err != nil {
		log.Printf("ERROR Telegram proxy broadcast recipient lookup failed: chat=%s text_len=%d error=%v", b.chatLogMarker(proxyChatID), len(text), err)
		_, _ = b.SendMessage(ctx, proxyChatID, "Не удалось подготовить рассылку участникам.")
		return
	}
	if len(recipients) == 0 {
		log.Printf("INFO Telegram proxy broadcast skipped: chat=%s text_len=%d reason=no_recipients", b.chatLogMarker(proxyChatID), len(text))
		_, _ = b.SendMessage(ctx, proxyChatID, "У активного события нет участников для рассылки.")
		return
	}

	// Все получатели — зарегистрированные участники активного события,
	// поэтому прикрепляем к сообщению рассылки главное меню участника.
	participantMenu := keyboard.MainMenu(true, true, b.miniappURL, nil)

	delivered := 0
	failed := 0
	for index, targetUserID := range recipients {
		if index > 0 && !b.waitProxyBroadcastInterval(ctx) {
			log.Printf("WARN Telegram proxy broadcast cancelled: chat=%s text_len=%d delivered=%d failed=%d recipient_count=%d error=%v", b.chatLogMarker(proxyChatID), len(text), delivered, failed, len(recipients), ctx.Err())
			return
		}

		if err := b.sendProxyBroadcastMessage(ctx, targetUserID, text, &participantMenu); err != nil {
			failed++
			log.Printf("WARN Telegram proxy broadcast delivery failed: chat=%s target_user_id=%d text_len=%d error=%v", b.chatLogMarker(proxyChatID), targetUserID, len(text), err)
			continue
		}
		delivered++
	}

	log.Printf("INFO Telegram proxy broadcast delivered: chat=%s text_len=%d delivered=%d failed=%d recipient_count=%d", b.chatLogMarker(proxyChatID), len(text), delivered, failed, len(recipients))
	_, _ = b.SendMessage(ctx, proxyChatID, fmt.Sprintf("Рассылка завершена. Доставлено: %d/%d. Ошибок: %d.", delivered, len(recipients), failed))
}

func (b *Bot) sendProxyBroadcastMessage(ctx context.Context, targetUserID int64, text string, markup *models.InlineKeyboardMarkup) error {
	if _, err := b.sendWithOptionalKeyboard(ctx, targetUserID, text, markup); err != nil {
		var rateLimitErr *telegrambot.TooManyRequestsError
		if !errors.As(err, &rateLimitErr) {
			return err
		}

		retryAfter := time.Duration(rateLimitErr.RetryAfter) * time.Second
		if retryAfter <= 0 {
			retryAfter = proxyBroadcastRetryWait
		}
		log.Printf("WARN Telegram proxy broadcast rate limited: target_user_id=%d retry_after=%s", targetUserID, retryAfter)
		if !waitProxyBroadcastDuration(ctx, retryAfter) {
			return fmt.Errorf("wait retry after: %w", ctx.Err())
		}
		if _, retryErr := b.sendWithOptionalKeyboard(ctx, targetUserID, text, markup); retryErr != nil {
			return fmt.Errorf("retry after rate limit: %w", retryErr)
		}
	}

	return nil
}

func (b *Bot) waitProxyBroadcastInterval(ctx context.Context) bool {
	if b == nil || b.proxySendInterval <= 0 {
		return true
	}

	return waitProxyBroadcastDuration(ctx, b.proxySendInterval)
}

func waitProxyBroadcastDuration(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (b *Bot) proxyBroadcastRecipients(ctx context.Context) ([]int64, error) {
	if b == nil || b.eventRepo == nil || b.participantRepo == nil {
		return nil, errors.New("missing event or participant repository")
	}

	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("find active event: %w", err)
	}
	if event == nil {
		return nil, nil
	}

	participants, err := b.participantRepo.FindByEvent(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("find event participants: %w", err)
	}

	return uniqueParticipantUserIDs(participants), nil
}

func uniqueParticipantUserIDs(participants []*entity.Participant) []int64 {
	seen := make(map[int64]struct{}, len(participants))
	recipients := make([]int64, 0, len(participants))
	for _, participant := range participants {
		if participant == nil || participant.UserID <= 0 {
			continue
		}
		if _, ok := seen[participant.UserID]; ok {
			continue
		}
		seen[participant.UserID] = struct{}{}
		recipients = append(recipients, participant.UserID)
	}

	return recipients
}

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
		userID := int64(0)
		if sender, ok := messageSender(msg); ok {
			userID = sender.ID
		}
		if !b.sendMainMenu(ctx, msg.Chat.ID, userID, "Главное меню:") {
			_, _ = b.SendMessage(ctx, msg.Chat.ID, "Неизвестная команда. Используйте /start или /menu")
		}
	}
}

// handleStartCommand обрабатывает команду /start
func (b *Bot) handleStartCommand(ctx context.Context, msg *models.Message) {
	startHandler := handler.NewStartHandler(b.userRepo, b.eventRepo, b.participantRepo, b.miniappURL)
	text, markup := startHandler.Handle(ctx, msg)

	if _, err := b.sendLongTextWithOptionalKeyboard(ctx, msg.Chat.ID, text, markup); err != nil {
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

	if err := b.ensureTelegramUser(ctx, callback.From); err != nil {
		log.Printf("ERROR Telegram callback user ensure failed: telegram_user_id=%d data=%s error=%v", userID, data, err)
		_ = b.AnswerCallback(ctx, callback.ID, "Ошибка")
		return
	}
	sessionState := session.SessionState("unknown")
	if b.sessionManager != nil {
		sessionState = b.sessionManager.GetState(userID)
	}
	log.Printf("INFO Telegram callback accepted: telegram_user_id=%d chat=%s data=%s state=%s user_ensured=true", userID, b.chatLogMarker(chatID), data, sessionState)

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

		_ = b.editMessageWithKeyboard(ctx, msgRef.ChatID, msgRef.MessageID, "Действие отменено.", b.getStartKeyboard(ctx, userID))
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

	// Не удаляем приветственное меню (msgRef.MessageID не передаём): оно должно
	// остаться в чате. Управляющие сообщения самого сценария подарка по-прежнему
	// заменяются через отслеживаемые gift_message_ids.
	_, _ = b.replaceGiftControlMessage(ctx, callback.From.ID, msgRef.ChatID, text, markup)
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

	// Отправляем промпт отдельным сообщением и не трогаем приветственное меню,
	// с которого открыли сценарий (раньше его правка/удаление давали дубль промпта).
	if markup != nil {
		_, _ = b.SendMessageWithKeyboard(ctx, msgRef.ChatID, text, *markup)
		return
	}

	// Сценарий не запущен (нет активного события, нет регистрации, результат уже
	// отправлен и т.п.) — показываем причину и главное меню.
	if !b.sendMainMenu(ctx, msgRef.ChatID, callback.From.ID, text) {
		_, _ = b.SendMessage(ctx, msgRef.ChatID, text)
	}
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

	_, _ = b.sendLongTextWithOptionalKeyboard(ctx, msgRef.ChatID, text, nil)
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
			text, markup := registrationHandler.HandleGenderSelection(ctx, userID, gender)
			if err := b.AnswerCallback(ctx, callback.ID, ""); err != nil {
				return
			}

			// Удаляем сообщение с кнопками выбора
			_ = b.DeleteMessage(ctx, msgRef.ChatID, msgRef.MessageID)

			// Отправляем условия участия и кнопки согласия/отказа.
			_, _ = b.sendLongTextWithOptionalKeyboard(ctx, msgRef.ChatID, text, markup)
		}

	case session.StateAwaitingRegistrationConsent:
		registrationHandler := handler.NewRegistrationHandler(
			b.sessionManager,
			b.eventRepo,
			b.participantRepo,
			b.registerParticipantHandler,
		)
		switch data {
		case "registration_accept_conditions":
			text, err := registrationHandler.ConfirmRegistration(ctx, userID)
			if err != nil {
				_ = b.AnswerCallback(ctx, callback.ID, "Ошибка")
				_, _ = b.SendMessage(ctx, msgRef.ChatID, text)
				return
			}
			if err := b.AnswerCallback(ctx, callback.ID, "Согласие принято"); err != nil {
				return
			}
			_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, b.getStartKeyboard(ctx, userID))
		case "registration_decline_conditions":
			text := registrationHandler.DeclineRegistration(userID)
			if err := b.AnswerCallback(ctx, callback.ID, "Регистрация отменена"); err != nil {
				return
			}
			_, _ = b.sendWithOptionalKeyboard(ctx, msgRef.ChatID, text, b.getStartKeyboard(ctx, userID))
		default:
			if isRegistrationConsentCallback(data) {
				_ = b.AnswerCallback(ctx, callback.ID, "Начните регистрацию заново")
			}
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
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackContinueText(userID), text, markup)
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
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackContinueText(userID), text, markup)
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
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackAddDescriptionText(userID), text, markup)
		case "confirm_gift":
			log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
			text, markup := giftHandler.GiftDraftPrompt(userID)
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackReviewDraftText(userID), text, markup)
		case "restart_gift":
			b.handleGiftRestartCallback(ctx, callback, msgRef, userID, giftHandler)
		default:
			if isGiftFlowCallback(data) {
				log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
				text, markup := giftHandler.GiftDraftPrompt(userID)
				b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackContinueText(userID), text, markup)
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
			b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackReviewDraftText(userID), text, markup)
		case "restart_gift":
			b.handleGiftRestartCallback(ctx, callback, msgRef, userID, giftHandler)
		default:
			if isGiftFlowCallback(data) {
				log.Printf("INFO Gift stale callback recovered: user_id=%d callback_data=%s state=%s", userID, data, state)
				text, markup := giftHandler.GiftDraftPrompt(userID)
				b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackContinueText(userID), text, markup)
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
				b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackConfirmText(userID), text, markup)
				return
			}
			b.logDebug("Unsupported gift confirmation callback: user_id=%d data=%s", userID, data)
		}

	default:
		if isGiftFlowCallback(data) {
			giftHandler := handler.NewGiftHandler(
				b.sessionManager,
				b.eventRepo,
				b.addGiftHandler,
			)
			log.Printf("INFO Gift stale callback ignored: user_id=%d callback_data=%s state=%s", userID, data, state)
			_ = b.AnswerCallback(ctx, callback.ID, giftHandler.GiftCallbackOpenMenuText(userID))
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
		b.answerAndReplaceGiftControl(ctx, callback, msgRef, userID, giftHandler.GiftCallbackAddDescriptionText(userID), text, markup)
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

func isRegistrationConsentCallback(data string) bool {
	return data == "registration_accept_conditions" || data == "registration_decline_conditions"
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
		text, participant, _ := resultHandler.HandleResultLink(ctx, userID, resultLink)
		// При успешной отправке (participant != nil) уведомляем чат админов.
		if participant != nil {
			b.notifyAdminAboutResult(ctx, sender, participant)
		}
		// Если отправка завершена (сессия сброшена) — возвращаем главное меню,
		// иначе остаёмся в сценарии и просто отвечаем текстом.
		if b.sessionManager.GetState(userID) == session.StateIdle {
			if !b.sendMainMenu(ctx, msg.Chat.ID, userID, text) {
				_, _ = b.SendMessage(ctx, msg.Chat.ID, text)
			}
			return
		}
		_, _ = b.SendMessage(ctx, msg.Chat.ID, text)

	default:
		// Если нет активного состояния, передаём сообщение менеджерам или показываем меню.
		if b.botMessagesChatID != 0 {
			b.forwardUserMessageToProxy(ctx, msg, sender)
			return
		}
		if b.publicChatID == 0 || msg.Chat.ID != b.publicChatID {
			if !b.sendMainMenu(ctx, msg.Chat.ID, userID, "Главное меню:") {
				_, _ = b.SendMessage(ctx, msg.Chat.ID, "Используйте /start для начала работы с ботом.")
			}
		}
	}
}

func (b *Bot) forwardUserMessageToProxy(ctx context.Context, msg *models.Message, sender *models.User) {
	if msg == nil || sender == nil {
		log.Printf("WARN Telegram proxy user message skipped: reason=missing_message_or_sender")
		return
	}

	b.rememberProxyUser(sender)
	updateKind := messageUpdateKind(msg)
	log.Printf("INFO Telegram proxy user message routing started: telegram_user_id=%d update_kind=%s", sender.ID, updateKind)
	if _, err := b.api.CopyMessage(ctx, &telegrambot.CopyMessageParams{
		ChatID:      b.botMessagesChatID,
		FromChatID:  msg.Chat.ID,
		MessageID:   msg.ID,
		ReplyMarkup: proxyOpenUserChatKeyboard(sender.ID),
	}); err != nil {
		log.Printf("WARN Telegram proxy user message copy failed: telegram_user_id=%d update_kind=%s error=%v", sender.ID, updateKind, err)
		_, _ = b.SendMessage(ctx, b.botMessagesChatID, fmt.Sprintf("Не удалось скопировать сообщение от user_id=%d.", sender.ID))
		return
	}

	_, _ = b.SendMessage(ctx, b.botMessagesChatID, proxyUserMetadataText(sender))
	log.Printf("INFO Telegram proxy user message routed: telegram_user_id=%d update_kind=%s", sender.ID, updateKind)
}

func (b *Bot) notifyExpiredProxyDialog(ctx context.Context, chatID int64) int64 {
	targetUserID, expired := b.expireProxyDialog(time.Now())
	if !expired {
		return 0
	}

	_, _ = b.SendMessage(ctx, chatID, fmt.Sprintf("Чат с %s автоматически закрыт из-за отсутствия активности 2 минуты.", b.proxyUserLabel(ctx, targetUserID)))
	return targetUserID
}

func (b *Bot) proxyUserLabel(ctx context.Context, targetUserID int64) string {
	label := fmt.Sprintf("user_id=%d", targetUserID)
	user := b.findProxyUser(ctx, targetUserID)
	if user == nil {
		return label
	}

	parts := make([]string, 0, 2)
	username := strings.TrimSpace(user.Username)
	if username != "" {
		if !strings.HasPrefix(username, "@") {
			username = "@" + username
		}
		parts = append(parts, fmt.Sprintf("ник=%s", username))
	}

	name := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(user.FirstName),
		strings.TrimSpace(user.LastName),
	}, " "))
	if name != "" {
		parts = append(parts, fmt.Sprintf("имя=%s", name))
	}

	if len(parts) == 0 {
		return label
	}

	return fmt.Sprintf("%s (%s)", label, strings.Join(parts, ", "))
}

func (b *Bot) rememberProxyUser(user *models.User) {
	if b == nil || user == nil || user.ID <= 0 {
		return
	}

	profile := entity.User{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}

	b.proxyUsersMu.Lock()
	defer b.proxyUsersMu.Unlock()

	if b.proxyUsers == nil {
		b.proxyUsers = make(map[int64]entity.User)
	}
	b.proxyUsers[user.ID] = profile
}

func (b *Bot) findProxyUser(ctx context.Context, targetUserID int64) *entity.User {
	if b == nil || targetUserID <= 0 {
		return nil
	}

	if user := b.cachedProxyUser(targetUserID); user != nil {
		return user
	}
	if b.userRepo == nil {
		return nil
	}

	user, err := b.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		b.logDebug("Telegram proxy user profile lookup skipped: target_user_id=%d error=%v", targetUserID, err)
		return nil
	}

	return user
}

func (b *Bot) cachedProxyUser(targetUserID int64) *entity.User {
	b.proxyUsersMu.RLock()
	defer b.proxyUsersMu.RUnlock()

	user, ok := b.proxyUsers[targetUserID]
	if !ok {
		return nil
	}

	return &user
}

func (b *Bot) handleNewChatMembers(ctx context.Context, msg *models.Message) {
	if msg == nil {
		log.Printf("WARN Public chat welcome skipped: reason=nil_message")
		return
	}
	if !b.isPublicChat(msg.Chat.ID) {
		b.logDebug("Telegram new chat members ignored outside configured public chat: chat=%s", b.chatLogMarker(msg.Chat.ID))
		return
	}
	if len(msg.NewChatMembers) == 0 {
		log.Printf("WARN Public chat welcome skipped: chat=public reason=no_new_members")
		return
	}

	log.Printf("INFO Public chat welcome batch accepted: chat=public member_count=%d", len(msg.NewChatMembers))

	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("WARN Failed to load active event for public chat welcome: chat=public error=%v", err)
		return
	}
	if event == nil {
		log.Printf("WARN Public chat welcome skipped: chat=public reason=no_active_event")
		return
	}

	prizeFundLink, registerLink, conditionsLink := b.publicWelcomeLinks()

	for _, member := range msg.NewChatMembers {
		if member.IsBot {
			b.logDebug("Public chat welcome member skipped: chat=public reason=bot")
			continue
		}
		if member.ID == 0 {
			log.Printf("WARN Public chat welcome member skipped: chat=public reason=missing_telegram_user_id")
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

		if err := b.ensureTelegramUser(ctx, member); err != nil {
			log.Printf("WARN Public chat welcome skipped: telegram_user_id=%d chat=public operation=user_upsert error=%v", member.ID, err)
			continue
		}

		firstName := strings.TrimSpace(member.FirstName)
		if firstName == "" {
			firstName = "друг"
		}

		text := fmt.Sprintf("👋 Привет, %s! Добро пожаловать в %s 🚴", firstName, event.Name)
		markup := keyboard.PublicMenu(
			prizeFundLink,
			registerLink,
			conditionsLink,
		)

		var sendErr error
		if len(markup.InlineKeyboard) == 0 {
			_, sendErr = b.SendMessage(ctx, msg.Chat.ID, text)
		} else {
			_, sendErr = b.SendMessageWithKeyboard(ctx, msg.Chat.ID, text, markup)
		}
		if sendErr != nil {
			log.Printf("WARN Public chat welcome failed: telegram_user_id=%d chat=public event_id=%d error=%v", member.ID, event.ID, sendErr)
			continue
		}

		log.Printf("INFO Public chat welcome sent: telegram_user_id=%d event_id=%d chat=public", member.ID, event.ID)
	}
}

func (b *Bot) publicWelcomeLinks() (prizeFundLink string, registerLink string, conditionsLink string) {
	registerLink = b.deepLink("register")
	if registerLink == "" {
		log.Printf("WARN Public chat welcome button skipped: chat=public button=register reason=missing_bot_username")
	}

	conditionsLink = b.deepLink("conditions")
	if conditionsLink == "" {
		log.Printf("WARN Public chat welcome button skipped: chat=public button=conditions reason=missing_bot_username")
	}

	if strings.TrimSpace(b.miniappURL) == "" {
		return "", registerLink, conditionsLink
	}

	link, ok := b.miniappTelegramLink()
	if !ok {
		log.Printf("WARN Public chat welcome button skipped: chat=public button=prize_fund reason=missing_bot_username")
		return "", registerLink, conditionsLink
	}

	return link, registerLink, conditionsLink
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
	if b == nil || b.eventRepo == nil {
		return nil
	}

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

// sendMainMenu отправляет сообщение с главным меню, если есть активное событие.
// Используется, когда пользователь не находится внутри другого сценария, чтобы
// меню оставалось доступным после ответов бота. Возвращает false, если меню
// недоступно (нет активного события) — тогда вызывающий код решает, что отправить.
func (b *Bot) sendMainMenu(ctx context.Context, chatID int64, userID int64, text string) bool {
	markup := b.getStartKeyboard(ctx, userID)
	if markup == nil {
		return false
	}

	_, _ = b.sendWithOptionalKeyboard(ctx, chatID, text, markup)
	return true
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

func (b *Bot) ensureTelegramUser(ctx context.Context, user models.User) error {
	if b == nil || b.userRepo == nil {
		return nil
	}
	if user.ID == 0 {
		return fmt.Errorf("missing telegram user id")
	}

	telegramUser := &entity.User{
		ID:        user.ID,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}
	if err := b.userRepo.Create(ctx, telegramUser); err != nil {
		return fmt.Errorf("upsert telegram user %d: %w", user.ID, err)
	}

	return nil
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

func (b *Bot) sendLongTextWithOptionalKeyboard(ctx context.Context, chatID int64, text string, markup *models.InlineKeyboardMarkup) (*models.Message, error) {
	chunks := splitTelegramText(text, telegramTextLimit)
	if len(chunks) == 0 {
		return b.sendWithOptionalKeyboard(ctx, chatID, text, markup)
	}

	var sent *models.Message
	var err error
	for i, chunk := range chunks {
		if i == len(chunks)-1 {
			sent, err = b.sendWithOptionalKeyboard(ctx, chatID, chunk, markup)
		} else {
			sent, err = b.SendMessage(ctx, chatID, chunk)
		}
		if err != nil {
			return sent, err
		}
	}

	return sent, nil
}

func splitTelegramText(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if limit <= 0 || runeLen(text) <= limit {
		return []string{text}
	}

	chunks := make([]string, 0, runeLen(text)/limit+1)
	remaining := text
	for runeLen(remaining) > limit {
		runes := []rune(remaining)
		prefix := string(runes[:limit])
		splitAt := telegramTextSplitIndex(prefix, limit)
		if splitAt <= 0 {
			chunks = append(chunks, strings.TrimSpace(prefix))
			remaining = strings.TrimSpace(string(runes[limit:]))
			continue
		}

		chunks = append(chunks, strings.TrimSpace(prefix[:splitAt]))
		remaining = strings.TrimSpace(prefix[splitAt:] + string(runes[limit:]))
	}
	if strings.TrimSpace(remaining) != "" {
		chunks = append(chunks, strings.TrimSpace(remaining))
	}

	return chunks
}

func telegramTextSplitIndex(text string, limit int) int {
	minRunes := limit / 2
	for _, separator := range []string{"\n\n", "\n", " "} {
		idx := strings.LastIndex(text, separator)
		if idx <= 0 {
			continue
		}
		if runeLen(text[:idx]) >= minRunes {
			return idx + len(separator)
		}
	}
	return -1
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
