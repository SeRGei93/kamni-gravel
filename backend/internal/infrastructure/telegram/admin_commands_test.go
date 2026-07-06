package telegram

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
)

const (
	adminNotifyTestChatID   = int64(-200)
	adminNotifyTestPassword = "secret"
)

func newAdminNotifyTestBot(api *telegramAPIFake, participants []*entity.Participant, gifts []*entity.Gift, blacklisted map[int64]bool) *Bot {
	return &Bot{
		api:                      api,
		adminChatID:              adminNotifyTestChatID,
		adminActionsPassword:     adminNotifyTestPassword,
		eventRepo:                &telegramEventRepoFake{event: &entity.Event{ID: 77, Active: true, Name: "КАМНИ 200"}},
		participantRepo:          &telegramParticipantRepoFake{participants: participants},
		giftRepo:                 &telegramGiftRepoFake{gifts: gifts},
		isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(&telegramBlacklistRepoFake{blacklisted: blacklisted}),
	}
}

func adminNotifyCommandUpdate(text string, chatID int64) *models.Update {
	commandLength := len(text)
	if space := strings.Index(text, " "); space >= 0 {
		commandLength = space
	}
	return &models.Update{
		ID:      1,
		Message: commandMessageWithLength(text, commandLength, 900, chatID),
	}
}

func finishedParticipant(userID int64, firstName string, username string) *entity.Participant {
	return &entity.Participant{
		UserID: userID,
		Result: &entity.Result{ID: uint(userID)},
		User:   &entity.User{ID: userID, FirstName: firstName, Username: username},
	}
}

func TestAdminNotifyRecipientsSelection(t *testing.T) {
	participants := []*entity.Participant{
		finishedParticipant(1, "Anna", "anna"),
		finishedParticipant(2, "Boris", "boris"),                    // владелец подарка
		{UserID: 3, User: &entity.User{ID: 3, FirstName: "Céline"}}, // не финишировал
		finishedParticipant(4, "Dima", "dima"),                      // в чёрном списке
		finishedParticipant(1, "Anna", "anna"),                      // дубль
	}
	gifts := []*entity.Gift{{ID: 10, UserID: 2, EventID: 77}}
	b := newAdminNotifyTestBot(&telegramAPIFake{}, participants, gifts, map[int64]bool{4: true})

	recipients, event, err := b.missingGiftRecipients(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event == nil || event.ID != 77 {
		t.Fatalf("active event mismatch: got %+v", event)
	}
	if len(recipients) != 1 {
		t.Fatalf("recipient count mismatch: got %d, want 1 (%+v)", len(recipients), recipients)
	}
	if recipients[0].userID != 1 || recipients[0].firstName != "Anna" {
		t.Fatalf("recipient mismatch: got %+v", recipients[0])
	}
	if recipients[0].label != "Anna (@anna)" {
		t.Fatalf("recipient label mismatch: got %q", recipients[0].label)
	}
}

func TestAdminNotifyRecipientsWithoutActiveEvent(t *testing.T) {
	b := newAdminNotifyTestBot(&telegramAPIFake{}, nil, nil, nil)
	b.eventRepo = &telegramEventRepoFake{}

	recipients, event, err := b.missingGiftRecipients(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != nil || recipients != nil {
		t.Fatalf("expected no event and no recipients, got event=%+v recipients=%+v", event, recipients)
	}
}

func TestAdminNotifyCommandIgnoredOutsideAdminChat(t *testing.T) {
	api := &telegramAPIFake{}
	b := newAdminNotifyTestBot(api, []*entity.Participant{finishedParticipant(1, "Anna", "anna")}, nil, nil)

	handledPrivate := b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", 500))
	handledGroup := b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", -300))

	if !handledPrivate || !handledGroup {
		t.Fatalf("command outside admin chat should be swallowed: private=%t group=%t", handledPrivate, handledGroup)
	}
	if len(api.sentMessages) != 0 {
		t.Fatalf("command outside admin chat should not reply, got %d messages", len(api.sentMessages))
	}

	if handled := b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/menu", adminNotifyTestChatID)); handled {
		t.Fatal("other admin chat commands should not be handled")
	}
}

func TestAdminNotifyPasswordChecks(t *testing.T) {
	api := &telegramAPIFake{}
	b := newAdminNotifyTestBot(api, []*entity.Participant{finishedParticipant(1, "Anna", "anna")}, nil, nil)

	b.adminActionsPassword = ""
	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", adminNotifyTestChatID))
	if got := api.sentMessages[len(api.sentMessages)-1].Text; !strings.Contains(got, "ADMIN_ACTIONS_PASSWORD") {
		t.Fatalf("unconfigured password reply mismatch: got %q", got)
	}

	b.adminActionsPassword = adminNotifyTestPassword
	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift", adminNotifyTestChatID))
	if got := api.sentMessages[len(api.sentMessages)-1].Text; got != adminNotifyMissingGiftUsage {
		t.Fatalf("usage reply mismatch: got %q", got)
	}

	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift wrong", adminNotifyTestChatID))
	if got := api.sentMessages[len(api.sentMessages)-1].Text; got != "Неверный пароль." {
		t.Fatalf("bad password reply mismatch: got %q", got)
	}
	if b.adminNotifyPending != nil {
		t.Fatal("pending state must not be created before password passes")
	}

	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", adminNotifyTestChatID))
	preview := api.sentMessages[len(api.sentMessages)-1]
	if !strings.Contains(preview.Text, "получат 1 чел.") || !strings.Contains(preview.Text, "Anna (@anna)") {
		t.Fatalf("preview text mismatch: got %q", preview.Text)
	}
	markup, ok := preview.ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("preview markup type mismatch: got %T", preview.ReplyMarkup)
	}
	if got := callbackData(markup); len(got) != 2 || got[0] != adminNotifyConfirmCallbackData || got[1] != adminNotifyCancelCallbackData {
		t.Fatalf("preview callback data mismatch: got %v", got)
	}
	if b.adminNotifyPending == nil {
		t.Fatal("pending state must be stored after preview")
	}
}

func TestAdminNotifyConfirmDeliversReminders(t *testing.T) {
	api := &telegramAPIFake{}
	participants := []*entity.Participant{
		finishedParticipant(1, "Anna", "anna"),
		finishedParticipant(2, "", "boris"),
	}
	b := newAdminNotifyTestBot(api, participants, nil, nil)

	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", adminNotifyTestChatID))
	previewMessageID := api.nextMessageID

	handled := b.handleAdminChatUpdate(context.Background(), &models.Update{
		ID:            2,
		CallbackQuery: callbackWithMessage(adminNotifyConfirmCallbackData, 900, adminNotifyTestChatID, previewMessageID),
	})
	if !handled {
		t.Fatal("confirm callback should be handled")
	}

	if len(api.answerCallbacks) != 1 || api.answerCallbacks[0].Text != "Отправляю…" {
		t.Fatalf("confirm answer mismatch: got %+v", api.answerCallbacks)
	}
	if len(api.editMessages) != 1 || !strings.Contains(api.editMessages[0].Text, "Отправляю") {
		t.Fatalf("preview edit mismatch: got %+v", api.editMessages)
	}

	// Сообщения: превью + 2 DM + финальный отчёт в админ-чат.
	if len(api.sentMessages) != 4 {
		t.Fatalf("sent message count mismatch: got %d, want 4", len(api.sentMessages))
	}
	firstDM := api.sentMessages[1]
	if chatIDFromAny(firstDM.ChatID) != 1 || !strings.Contains(firstDM.Text, "Anna, поздравляем с финишем «КАМНИ 200»") {
		t.Fatalf("first reminder mismatch: chat=%v text=%q", firstDM.ChatID, firstDM.Text)
	}
	secondDM := api.sentMessages[2]
	if chatIDFromAny(secondDM.ChatID) != 2 || !strings.Contains(secondDM.Text, "друг, поздравляем") {
		t.Fatalf("second reminder mismatch: chat=%v text=%q", secondDM.ChatID, secondDM.Text)
	}
	report := api.sentMessages[3]
	if chatIDFromAny(report.ChatID) != adminNotifyTestChatID || !strings.Contains(report.Text, "Доставлено: 2/2") {
		t.Fatalf("report mismatch: chat=%v text=%q", report.ChatID, report.Text)
	}
	if b.adminNotifyPending != nil {
		t.Fatal("pending state must be cleared after confirm")
	}

	// Повторный клик по той же кнопке не должен дублировать рассылку.
	b.handleAdminChatUpdate(context.Background(), &models.Update{
		ID:            3,
		CallbackQuery: callbackWithMessage(adminNotifyConfirmCallbackData, 900, adminNotifyTestChatID, previewMessageID),
	})
	if got := api.answerCallbacks[len(api.answerCallbacks)-1].Text; !strings.Contains(got, "Превью устарело") {
		t.Fatalf("stale confirm answer mismatch: got %q", got)
	}
	if len(api.sentMessages) != 4 {
		t.Fatalf("stale confirm must not send messages: got %d", len(api.sentMessages))
	}
}

func TestAdminNotifyCancelClearsPending(t *testing.T) {
	api := &telegramAPIFake{}
	b := newAdminNotifyTestBot(api, []*entity.Participant{finishedParticipant(1, "Anna", "anna")}, nil, nil)

	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", adminNotifyTestChatID))
	previewMessageID := api.nextMessageID

	b.handleAdminChatUpdate(context.Background(), &models.Update{
		ID:            2,
		CallbackQuery: callbackWithMessage(adminNotifyCancelCallbackData, 900, adminNotifyTestChatID, previewMessageID),
	})

	if got := api.answerCallbacks[len(api.answerCallbacks)-1].Text; got != "Отменено" {
		t.Fatalf("cancel answer mismatch: got %q", got)
	}
	if len(api.editMessages) != 1 || !strings.Contains(api.editMessages[0].Text, "Отменено") {
		t.Fatalf("cancel edit mismatch: got %+v", api.editMessages)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("cancel must not send reminders: got %d messages", len(api.sentMessages))
	}
	if b.adminNotifyPending != nil {
		t.Fatal("pending state must be cleared after cancel")
	}
}

func TestAdminNotifyConfirmExpiredPreview(t *testing.T) {
	api := &telegramAPIFake{}
	b := newAdminNotifyTestBot(api, []*entity.Participant{finishedParticipant(1, "Anna", "anna")}, nil, nil)

	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", adminNotifyTestChatID))
	previewMessageID := api.nextMessageID
	b.adminNotifyPending.createdAt = time.Now().Add(-adminNotifyPendingTTL - time.Minute)

	b.handleAdminChatUpdate(context.Background(), &models.Update{
		ID:            2,
		CallbackQuery: callbackWithMessage(adminNotifyConfirmCallbackData, 900, adminNotifyTestChatID, previewMessageID),
	})

	if got := api.answerCallbacks[len(api.answerCallbacks)-1].Text; !strings.Contains(got, "Превью устарело") {
		t.Fatalf("expired confirm answer mismatch: got %q", got)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("expired confirm must not send reminders: got %d messages", len(api.sentMessages))
	}
	if b.adminNotifyPending != nil {
		t.Fatal("expired pending state must be dropped")
	}
}

func TestAdminNotifyRerunReplacesPending(t *testing.T) {
	api := &telegramAPIFake{}
	b := newAdminNotifyTestBot(api, []*entity.Participant{finishedParticipant(1, "Anna", "anna")}, nil, nil)

	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", adminNotifyTestChatID))
	firstPreviewID := api.nextMessageID
	b.handleAdminChatUpdate(context.Background(), adminNotifyCommandUpdate("/notify_missing_gift secret", adminNotifyTestChatID))
	secondPreviewID := api.nextMessageID

	// Кнопки первого превью больше не действуют.
	b.handleAdminChatUpdate(context.Background(), &models.Update{
		ID:            2,
		CallbackQuery: callbackWithMessage(adminNotifyConfirmCallbackData, 900, adminNotifyTestChatID, firstPreviewID),
	})
	if got := api.answerCallbacks[len(api.answerCallbacks)-1].Text; !strings.Contains(got, "Превью устарело") {
		t.Fatalf("first preview confirm answer mismatch: got %q", got)
	}

	// Второе превью подтверждается и рассылает напоминания.
	b.handleAdminChatUpdate(context.Background(), &models.Update{
		ID:            3,
		CallbackQuery: callbackWithMessage(adminNotifyConfirmCallbackData, 900, adminNotifyTestChatID, secondPreviewID),
	})
	last := api.sentMessages[len(api.sentMessages)-1]
	if chatIDFromAny(last.ChatID) != adminNotifyTestChatID || !strings.Contains(last.Text, "Доставлено: 1/1") {
		t.Fatalf("second preview delivery report mismatch: chat=%v text=%q", last.ChatID, last.Text)
	}
}
