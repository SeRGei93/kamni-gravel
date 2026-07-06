package telegram

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
)

const (
	adminNotifyMissingGiftCommand  = "notify_missing_gift"
	adminNotifyCallbackPrefix      = "notify_gift:"
	adminNotifyConfirmCallbackData = "notify_gift:confirm"
	adminNotifyCancelCallbackData  = "notify_gift:cancel"
	adminNotifyPendingTTL          = 10 * time.Minute
	adminNotifyPreviewMaxLines     = 50
	adminNotifyMissingGiftUsage    = "Используйте /notify_missing_gift <пароль>"
)

// adminNotifyRecipient — снапшот получателя напоминания на момент превью.
type adminNotifyRecipient struct {
	userID    int64
	firstName string
	label     string
}

// adminNotifyPendingState — превью рассылки, ожидающее подтверждения в админ-чате.
type adminNotifyPendingState struct {
	recipients  []adminNotifyRecipient
	eventID     uint
	eventName   string
	chatID      int64
	messageID   int
	previewText string
	createdAt   time.Time
}

// handleAdminChatUpdate перехватывает команды и коллбэки админ-чата до фильтра
// приватных чатов. Возвращает true, когда обновление обработано (или сознательно
// проглочено) и дальнейший роутинг не нужен.
func (b *Bot) handleAdminChatUpdate(ctx context.Context, update *models.Update) bool {
	if b == nil || update == nil {
		return false
	}

	if update.CallbackQuery != nil {
		if !strings.HasPrefix(update.CallbackQuery.Data, adminNotifyCallbackPrefix) {
			return false
		}

		msgRef, ok := callbackMessage(update.CallbackQuery)
		if !ok || !b.isAdminChat(msgRef.ChatID) {
			log.Printf("WARN Admin notify callback ignored outside admin chat: data=%s", update.CallbackQuery.Data)
			_ = b.AnswerCallback(ctx, update.CallbackQuery.ID, "Недоступно")
			return true
		}

		switch update.CallbackQuery.Data {
		case adminNotifyConfirmCallbackData:
			log.Printf("INFO Admin command routed: callback=%s chat=admin", update.CallbackQuery.Data)
			b.handleAdminNotifyConfirmCallback(ctx, update.CallbackQuery, msgRef)
		case adminNotifyCancelCallbackData:
			log.Printf("INFO Admin command routed: callback=%s chat=admin", update.CallbackQuery.Data)
			b.handleAdminNotifyCancelCallback(ctx, update.CallbackQuery, msgRef)
		default:
			log.Printf("INFO Admin notify callback rejected: data=%s reason=unsupported chat=admin", update.CallbackQuery.Data)
			_ = b.AnswerCallback(ctx, update.CallbackQuery.ID, "Недоступно")
		}
		return true
	}

	if update.Message == nil {
		return false
	}

	command := messageCommand(update.Message)
	if command != adminNotifyMissingGiftCommand {
		return false
	}
	if !b.isAdminChat(update.Message.Chat.ID) {
		log.Printf("WARN Admin command ignored outside admin chat: command=%s chat=%s", command, b.chatLogMarker(update.Message.Chat.ID))
		return true
	}

	log.Printf("INFO Admin command routed: command=%s chat=admin", command)
	b.handleNotifyMissingGiftCommand(ctx, update.Message)
	return true
}

// configureAdminChatCommands регистрирует меню команд, доступное только в админ-чате.
func (b *Bot) configureAdminChatCommands(ctx context.Context) {
	if b.adminChatID == 0 {
		return
	}

	ok, err := b.api.SetMyCommands(ctx, &telegrambot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     adminNotifyMissingGiftCommand,
				Description: "Напомнить финишёрам без приза",
			},
		},
		Scope: &models.BotCommandScopeChat{ChatID: b.adminChatID},
	})
	if err != nil {
		log.Printf("WARN Telegram admin commands setup failed: chat=admin error=%v", err)
		return
	}
	if !ok {
		log.Printf("WARN Telegram admin commands setup failed: chat=admin result=false")
		return
	}

	log.Printf("INFO Telegram admin commands configured: chat=admin")
}

func (b *Bot) handleNotifyMissingGiftCommand(ctx context.Context, msg *models.Message) {
	if strings.TrimSpace(b.adminActionsPassword) == "" {
		log.Printf("WARN Admin action rejected: command=%s reason=password_not_configured chat=admin", adminNotifyMissingGiftCommand)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Команда не настроена: задайте ADMIN_ACTIONS_PASSWORD.")
		return
	}

	password, ok := messageCommandTail(msg)
	if !ok {
		log.Printf("INFO Admin command rejected: command=%s reason=missing_password chat=admin", adminNotifyMissingGiftCommand)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, adminNotifyMissingGiftUsage)
		return
	}
	if subtle.ConstantTimeCompare([]byte(password), []byte(b.adminActionsPassword)) != 1 {
		log.Printf("WARN Admin action rejected: command=%s reason=bad_password chat=admin", adminNotifyMissingGiftCommand)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Неверный пароль.")
		return
	}

	recipients, event, err := b.missingGiftRecipients(ctx)
	if err != nil {
		log.Printf("ERROR Admin notify recipients lookup failed: command=%s error=%v", adminNotifyMissingGiftCommand, err)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Не удалось подготовить список получателей.")
		return
	}
	if event == nil {
		log.Printf("INFO Admin notify skipped: command=%s reason=no_active_event", adminNotifyMissingGiftCommand)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Нет активного события.")
		return
	}
	if len(recipients) == 0 {
		log.Printf("INFO Admin notify skipped: command=%s event_id=%d reason=no_recipients", adminNotifyMissingGiftCommand, event.ID)
		_, _ = b.SendMessage(ctx, msg.Chat.ID, "Все финишёры уже добавили призы 🎉")
		return
	}

	previewText := adminNotifyPreviewText(recipients, event.Name)
	sent, err := b.SendMessageWithKeyboard(ctx, msg.Chat.ID, previewText, adminNotifyPreviewKeyboard())
	if err != nil {
		log.Printf("ERROR Admin notify preview failed: event_id=%d recipient_count=%d error=%v", event.ID, len(recipients), err)
		return
	}

	messageID := 0
	if sent != nil {
		messageID = sent.ID
	}

	b.adminNotifyMu.Lock()
	b.adminNotifyPending = &adminNotifyPendingState{
		recipients:  recipients,
		eventID:     event.ID,
		eventName:   event.Name,
		chatID:      msg.Chat.ID,
		messageID:   messageID,
		previewText: previewText,
		createdAt:   time.Now(),
	}
	b.adminNotifyMu.Unlock()

	log.Printf("INFO Admin notify preview built: event_id=%d recipient_count=%d chat=admin", event.ID, len(recipients))
}

// missingGiftRecipients возвращает финишёров активного события без подарка.
// Возвращает (nil, nil, nil), когда активного события нет.
func (b *Bot) missingGiftRecipients(ctx context.Context) ([]adminNotifyRecipient, *entity.Event, error) {
	if b == nil || b.eventRepo == nil || b.participantRepo == nil || b.giftRepo == nil {
		return nil, nil, errors.New("missing event, participant or gift repository")
	}

	event, err := b.eventRepo.FindActive(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("find active event: %w", err)
	}
	if event == nil {
		return nil, nil, nil
	}

	participants, err := b.participantRepo.FindByEvent(ctx, event.ID)
	if err != nil {
		return nil, event, fmt.Errorf("find event participants: %w", err)
	}

	gifts, err := b.giftRepo.FindByEvent(ctx, event.ID)
	if err != nil {
		return nil, event, fmt.Errorf("find event gifts: %w", err)
	}

	giftHolders := make(map[int64]struct{}, len(gifts))
	for _, gift := range gifts {
		if gift == nil || gift.UserID <= 0 {
			continue
		}
		giftHolders[gift.UserID] = struct{}{}
	}

	seen := make(map[int64]struct{}, len(participants))
	recipients := make([]adminNotifyRecipient, 0, len(participants))
	for _, participant := range participants {
		if participant == nil || participant.UserID <= 0 || !participant.IsFinished() {
			continue
		}
		if _, ok := giftHolders[participant.UserID]; ok {
			continue
		}
		if _, ok := seen[participant.UserID]; ok {
			continue
		}
		seen[participant.UserID] = struct{}{}

		if b.isUserBlacklistedHandler != nil {
			isBlacklisted, err := b.isUserBlacklistedHandler.Handle(ctx, query.IsUserBlacklistedQuery{TelegramUserID: participant.UserID})
			if err != nil {
				log.Printf("WARN Admin notify recipient skipped: telegram_user_id=%d operation=blacklist_check error=%v", participant.UserID, err)
				continue
			}
			if isBlacklisted {
				log.Printf("INFO Admin notify recipient skipped: telegram_user_id=%d reason=blacklisted", participant.UserID)
				continue
			}
		}

		firstName := ""
		if participant.User != nil {
			firstName = strings.TrimSpace(participant.User.FirstName)
		}
		recipients = append(recipients, adminNotifyRecipient{
			userID:    participant.UserID,
			firstName: firstName,
			label:     adminNotifyRecipientLabel(participant.User, participant.UserID),
		})
	}

	return recipients, event, nil
}

func adminNotifyRecipientLabel(user *entity.User, userID int64) string {
	if user == nil {
		return fmt.Sprintf("id:%d", userID)
	}

	name := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(user.FirstName),
		strings.TrimSpace(user.LastName),
	}, " "))
	username := strings.TrimSpace(user.Username)

	switch {
	case name != "" && username != "":
		return fmt.Sprintf("%s (@%s)", name, username)
	case name != "":
		return name
	case username != "":
		return "@" + username
	default:
		return fmt.Sprintf("id:%d", userID)
	}
}

func adminNotifyPreviewText(recipients []adminNotifyRecipient, eventName string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Напоминание о призе получат %d чел. (событие «%s»):\n", len(recipients), eventName)

	for index, recipient := range recipients {
		if index >= adminNotifyPreviewMaxLines {
			fmt.Fprintf(&sb, "… и ещё %d", len(recipients)-adminNotifyPreviewMaxLines)
			break
		}
		fmt.Fprintf(&sb, "%d. %s\n", index+1, recipient.label)
	}

	return strings.TrimRight(sb.String(), "\n")
}

func adminNotifyPreviewKeyboard() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "✅ Отправить", CallbackData: adminNotifyConfirmCallbackData},
				{Text: "❌ Отмена", CallbackData: adminNotifyCancelCallbackData},
			},
		},
	}
}

// takeAdminNotifyPending атомарно забирает превью, если оно соответствует
// сообщению коллбэка и не протухло. Возвращает nil, когда подтверждать нечего.
func (b *Bot) takeAdminNotifyPending(messageID int) *adminNotifyPendingState {
	b.adminNotifyMu.Lock()
	defer b.adminNotifyMu.Unlock()

	pending := b.adminNotifyPending
	if pending == nil {
		return nil
	}
	if pending.messageID != messageID {
		log.Printf("INFO Admin notify pending mismatch: pending_message_id=%d callback_message_id=%d", pending.messageID, messageID)
		return nil
	}
	if time.Since(pending.createdAt) > adminNotifyPendingTTL {
		log.Printf("INFO Admin notify pending expired: message_id=%d age=%s", pending.messageID, time.Since(pending.createdAt))
		b.adminNotifyPending = nil
		return nil
	}

	b.adminNotifyPending = nil
	return pending
}

func (b *Bot) handleAdminNotifyConfirmCallback(ctx context.Context, callback *models.CallbackQuery, msgRef callbackMessageRef) {
	pending := b.takeAdminNotifyPending(msgRef.MessageID)
	if pending == nil {
		_ = b.AnswerCallback(ctx, callback.ID, "Превью устарело, запустите команду заново")
		return
	}

	_ = b.AnswerCallback(ctx, callback.ID, "Отправляю…")
	_ = b.editMessageWithKeyboard(ctx, pending.chatID, pending.messageID, pending.previewText+"\n\n⏳ Отправляю…", nil)

	b.deliverAdminNotify(ctx, pending)
}

func (b *Bot) handleAdminNotifyCancelCallback(ctx context.Context, callback *models.CallbackQuery, msgRef callbackMessageRef) {
	pending := b.takeAdminNotifyPending(msgRef.MessageID)
	if pending == nil {
		_ = b.AnswerCallback(ctx, callback.ID, "Превью устарело, запустите команду заново")
		return
	}

	_ = b.AnswerCallback(ctx, callback.ID, "Отменено")
	_ = b.editMessageWithKeyboard(ctx, pending.chatID, pending.messageID, pending.previewText+"\n\n❌ Отменено.", nil)
	log.Printf("INFO Admin notify cancelled: event_id=%d recipient_count=%d", pending.eventID, len(pending.recipients))
}

// deliverAdminNotify шлёт напоминания получателям через тот же конвейер, что и
// /broadcast_participants: интервал между отправками и ретрай на rate limit.
func (b *Bot) deliverAdminNotify(ctx context.Context, pending *adminNotifyPendingState) {
	log.Printf("INFO Admin notify delivery started: event_id=%d recipient_count=%d", pending.eventID, len(pending.recipients))

	miniappLink, _ := b.miniappTelegramLink()

	delivered := 0
	failed := 0
	for index, recipient := range pending.recipients {
		if index > 0 && !b.waitProxyBroadcastInterval(ctx) {
			log.Printf("WARN Admin notify delivery cancelled: delivered=%d failed=%d recipient_count=%d error=%v", delivered, failed, len(pending.recipients), ctx.Err())
			return
		}

		text := adminNotifyReminderText(recipient.firstName, pending.eventName, miniappLink)
		if err := b.sendProxyBroadcastMessage(ctx, recipient.userID, text, nil); err != nil {
			failed++
			log.Printf("WARN Admin notify delivery failed: target_user_id=%d error=%v", recipient.userID, err)
			continue
		}
		delivered++
	}

	log.Printf("INFO Admin notify delivered: delivered=%d failed=%d recipient_count=%d event_id=%d", delivered, failed, len(pending.recipients), pending.eventID)
	_, _ = b.SendMessage(ctx, pending.chatID, fmt.Sprintf("Напоминания отправлены. Доставлено: %d/%d. Ошибок: %d.", delivered, len(pending.recipients), failed))
}

func adminNotifyReminderText(firstName, eventName, miniappLink string) string {
	name := strings.TrimSpace(firstName)
	if name == "" {
		name = "друг"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 %s, поздравляем с финишем «%s»!\n\n", name, eventName)
	sb.WriteString("🎁 Мы пока не нашли твой приз в призовом фонде. Каждый участник добавляет приз для розыгрыша — добавь свой командой /add_gift.")
	if miniappLink != "" {
		fmt.Fprintf(&sb, "\n\nПосмотреть призовой фонд: %s", miniappLink)
	}

	return sb.String()
}
