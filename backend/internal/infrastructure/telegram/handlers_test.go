package telegram

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	telegrambot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestBotHandleMenuCommandSendsParticipantAwareMenu(t *testing.T) {
	api := &telegramAPIFake{}
	userRepo := newTelegramUserRepoFake()
	userRepo.users[123] = &entity.User{ID: 123, FirstName: "Alex"}
	participantRepo := &telegramParticipantRepoFake{participant: &entity.Participant{ID: 55, UserID: 123, EventID: 77}}
	b := &Bot{
		api:             api,
		userRepo:        userRepo,
		eventRepo:       &telegramEventRepoFake{event: &entity.Event{ID: 77, Active: true, Name: "Spring Gravel"}},
		participantRepo: participantRepo,
	}

	b.handleCommand(context.Background(), commandMessage("/menu", 123, 500))

	if len(api.sentMessages) != 1 {
		t.Fatalf("sent message count mismatch: got %d, want 1", len(api.sentMessages))
	}
	if got := api.sentMessages[0].Text; got != "Главное меню:" {
		t.Fatalf("menu text mismatch: got %q", got)
	}
	markup, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("reply markup type mismatch: got %T", api.sentMessages[0].ReplyMarkup)
	}
	if got, want := callbackData(markup), []string{"withdraw_participation", "add_gift", "submit_result", "event_conditions"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("menu callback data mismatch: got %v, want %v", got, want)
	}
}

func TestBotHandleCommandIgnoresPublicAndAdminChats(t *testing.T) {
	api := &telegramAPIFake{}
	b := &Bot{
		api:          api,
		publicChatID: -100,
		adminChatID:  -200,
	}

	b.handleCommand(context.Background(), commandMessage("/start", 123, -100))
	b.handleCommand(context.Background(), commandMessage("/menu", 123, -200))

	if len(api.sentMessages) != 0 {
		t.Fatalf("service chat commands should be ignored, got %d messages", len(api.sentMessages))
	}
}

func TestBotHandleCommandIgnoresUnconfiguredGroupChatMention(t *testing.T) {
	api := &telegramAPIFake{}
	b := &Bot{api: api}

	b.handleCommand(context.Background(), commandMessage("/start@GravelBot", 123, -300))

	if len(api.sentMessages) != 0 {
		t.Fatalf("group chat command mention should be ignored, got %d messages", len(api.sentMessages))
	}
}

func TestBotHandleUpdateIgnoresPublicCommand(t *testing.T) {
	api := &telegramAPIFake{}
	blacklistRepo := &telegramBlacklistRepoFake{}
	b := &Bot{
		api:                      api,
		publicChatID:             -100,
		isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(blacklistRepo),
	}

	b.handleUpdate(context.Background(), nil, &models.Update{
		ID:      1,
		Message: commandMessage("/start", 123, -100),
	})

	if len(api.sentMessages) != 0 {
		t.Fatalf("public command should be ignored, got %d messages", len(api.sentMessages))
	}
	if len(blacklistRepo.checkedTelegramUserIDs) != 0 {
		t.Fatalf("public command should be ignored before blacklist check, got %v", blacklistRepo.checkedTelegramUserIDs)
	}
}

func TestBotHandleCallbackIgnoresGroupChat(t *testing.T) {
	api := &telegramAPIFake{}
	b := &Bot{api: api}

	b.handleCallback(context.Background(), callbackWithMessage("register", 123, -300, 10))

	if len(api.answerCallbacks) != 0 {
		t.Fatalf("group callback should not be answered, got %d answers", len(api.answerCallbacks))
	}
	if len(api.sentMessages) != 0 {
		t.Fatalf("group callback should not send messages, got %d messages", len(api.sentMessages))
	}
}

func TestBotHandleMessageIgnoresAdminChat(t *testing.T) {
	api := &telegramAPIFake{}
	b := &Bot{
		api:            api,
		adminChatID:    -200,
		sessionManager: session.NewManager(0),
	}

	b.handleMessage(context.Background(), &models.Message{
		ID:   10,
		From: &models.User{ID: 123},
		Chat: models.Chat{ID: -200},
		Text: "hello",
	})

	if len(api.sentMessages) != 0 {
		t.Fatalf("admin chat messages should be ignored, got %d messages", len(api.sentMessages))
	}
}

func TestBotHandleNewChatMembersIgnoresPublicWelcomeOutsidePrivateChat(t *testing.T) {
	api := &telegramAPIFake{}
	userRepo := newTelegramUserRepoFake()
	blacklistRepo := &telegramBlacklistRepoFake{}
	eventRepo := &telegramEventRepoFake{event: &entity.Event{ID: 77, Name: "Spring Gravel"}}
	b := &Bot{
		api:                      api,
		publicChatID:             -100,
		botUsername:              "GravelBot",
		miniappURL:               "https://example.com/miniapp",
		userRepo:                 userRepo,
		eventRepo:                eventRepo,
		isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(blacklistRepo),
	}

	b.handleNewChatMembers(context.Background(), &models.Message{
		ID:   10,
		From: &models.User{ID: 999, FirstName: "Inviter"},
		Chat: models.Chat{ID: -100},
		NewChatMembers: []models.User{
			{ID: 123, FirstName: "Alex", Username: "alex"},
			{ID: 456, FirstName: "Bot", IsBot: true},
		},
	})

	if len(api.sentMessages) != 0 {
		t.Fatalf("public welcome should be ignored outside private chat, got %d messages", len(api.sentMessages))
	}
	if len(userRepo.users) != 0 {
		t.Fatalf("joined user should not be created outside private chat, got users %#v", userRepo.users)
	}
	if len(blacklistRepo.checkedTelegramUserIDs) != 0 {
		t.Fatalf("blacklist should not be checked outside private chat, got %v", blacklistRepo.checkedTelegramUserIDs)
	}
	if eventRepo.findActiveCalls != 0 {
		t.Fatalf("active event should not be loaded outside private chat: got %d calls", eventRepo.findActiveCalls)
	}
}

func TestBotHandleNewChatMembersIgnoresUnrelatedPublicChat(t *testing.T) {
	api := &telegramAPIFake{}
	eventRepo := &telegramEventRepoFake{event: &entity.Event{ID: 77, Name: "Spring Gravel"}}
	b := &Bot{
		api:          api,
		publicChatID: -100,
		eventRepo:    eventRepo,
	}

	b.handleNewChatMembers(context.Background(), &models.Message{
		ID:   10,
		Chat: models.Chat{ID: -200},
		NewChatMembers: []models.User{
			{ID: 123, FirstName: "Alex"},
		},
	})

	if len(api.sentMessages) != 0 {
		t.Fatalf("unexpected public welcome messages: %d", len(api.sentMessages))
	}
	if eventRepo.findActiveCalls != 0 {
		t.Fatalf("active event should not be loaded for unrelated chat: got %d calls", eventRepo.findActiveCalls)
	}
}

func TestBotHandleNewChatMembersDoesNotCheckBlacklistOutsidePrivateChat(t *testing.T) {
	api := &telegramAPIFake{}
	userRepo := newTelegramUserRepoFake()
	blacklistRepo := &telegramBlacklistRepoFake{blacklisted: map[int64]bool{123: true}}
	b := &Bot{
		api:                      api,
		publicChatID:             -100,
		botUsername:              "GravelBot",
		userRepo:                 userRepo,
		eventRepo:                &telegramEventRepoFake{event: &entity.Event{ID: 77, Name: "Spring Gravel"}},
		isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(blacklistRepo),
	}

	b.handleNewChatMembers(context.Background(), &models.Message{
		ID:   10,
		From: &models.User{ID: 999, FirstName: "Inviter"},
		Chat: models.Chat{ID: -100},
		NewChatMembers: []models.User{
			{ID: 123, FirstName: "Blocked"},
		},
	})

	if len(api.sentMessages) != 0 {
		t.Fatalf("blacklisted joined user should not receive welcome, got %d messages", len(api.sentMessages))
	}
	if len(userRepo.users) != 0 {
		t.Fatalf("blacklisted joined user should not be created, got users %#v", userRepo.users)
	}
	if len(blacklistRepo.checkedTelegramUserIDs) != 0 {
		t.Fatalf("blacklist should not be checked outside private chat, got %v", blacklistRepo.checkedTelegramUserIDs)
	}
}

func TestBotHandleUpdateIgnoresNewChatMembersOutsidePrivateChat(t *testing.T) {
	api := &telegramAPIFake{}
	userRepo := newTelegramUserRepoFake()
	blacklistRepo := &telegramBlacklistRepoFake{blacklisted: map[int64]bool{999: true}}
	b := &Bot{
		api:                      api,
		publicChatID:             -100,
		botUsername:              "GravelBot",
		userRepo:                 userRepo,
		eventRepo:                &telegramEventRepoFake{event: &entity.Event{ID: 77, Name: "Spring Gravel"}},
		isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(blacklistRepo),
	}

	b.handleUpdate(context.Background(), nil, &models.Update{
		ID: 1,
		Message: &models.Message{
			ID:   10,
			From: &models.User{ID: 999, FirstName: "Blocked inviter"},
			Chat: models.Chat{ID: -100},
			NewChatMembers: []models.User{
				{ID: 123, FirstName: "Alex"},
			},
		},
	})

	if len(api.sentMessages) != 0 {
		t.Fatalf("new chat members outside private chat should be ignored, got %d messages", len(api.sentMessages))
	}
	if len(userRepo.users) != 0 {
		t.Fatalf("joined user should not be created, got users %#v", userRepo.users)
	}
	if len(blacklistRepo.checkedTelegramUserIDs) != 0 {
		t.Fatalf("blacklist should not be checked, got %v", blacklistRepo.checkedTelegramUserIDs)
	}
}

func TestBotHandleEventConditionsCallbackSendsDescription(t *testing.T) {
	api := &telegramAPIFake{}
	b := &Bot{
		api:       api,
		eventRepo: &telegramEventRepoFake{event: &entity.Event{ID: 77, Description: "Условия участия"}},
	}

	b.handleEventConditionsCallback(context.Background(), callbackWithMessage("event_conditions", 123, 500, 10))

	if len(api.answerCallbacks) != 1 {
		t.Fatalf("answer callback count mismatch: got %d, want 1", len(api.answerCallbacks))
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("sent message count mismatch: got %d, want 1", len(api.sentMessages))
	}
	if got := api.sentMessages[0].Text; got != "Условия участия" {
		t.Fatalf("conditions text mismatch: got %q", got)
	}
}

func TestBotHandleWithdrawParticipationCallbackDeletesParticipant(t *testing.T) {
	api := &telegramAPIFake{}
	participantRepo := &telegramParticipantRepoFake{
		participant: &entity.Participant{ID: 55, UserID: 123, EventID: 77},
	}
	b := &Bot{
		api:                        api,
		sessionManager:             session.NewManager(0),
		eventRepo:                  &telegramEventRepoFake{event: &entity.Event{ID: 77, Active: true}},
		participantRepo:            participantRepo,
		withdrawParticipantHandler: command.NewWithdrawParticipantHandler(participantRepo),
	}

	b.handleWithdrawParticipationCallback(context.Background(), callbackWithMessage("withdraw_participation", 123, 500, 10))

	if participantRepo.deletedParticipantID != 55 {
		t.Fatalf("deleted participant mismatch: got %d, want 55", participantRepo.deletedParticipantID)
	}
	if len(api.answerCallbacks) != 1 {
		t.Fatalf("answer callback count mismatch: got %d, want 1", len(api.answerCallbacks))
	}
	if !sentTextContains(api, "больше не участвуете") {
		t.Fatalf("withdrawal confirmation not sent: %#v", api.sentMessages)
	}
	if len(api.sentMessages) != 2 {
		t.Fatalf("sent message count mismatch: got %d, want 2", len(api.sentMessages))
	}
	if strings.TrimSpace(api.sentMessages[1].Text) == "" {
		t.Fatalf("menu message text must not be empty: %#v", api.sentMessages[1])
	}
	if _, ok := api.sentMessages[1].ReplyMarkup.(models.InlineKeyboardMarkup); !ok {
		t.Fatalf("menu reply markup type mismatch: got %T", api.sentMessages[1].ReplyMarkup)
	}
}

func TestBotNotifyAdminAboutGiftSendsSinglePhotoNotification(t *testing.T) {
	const miniappURL = "https://example.com/miniapp/gifts"
	api := &telegramAPIFake{}
	b := &Bot{
		api:         api,
		adminChatID: 900,
		botUsername: "GravelBot",
		miniappURL:  miniappURL,
	}
	gift := &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		ReviewStatus:   entity.GiftReviewStatusPendingReview,
		User:           &entity.User{ID: 123, FirstName: "Alex", Username: "alex"},
		Attachments: []entity.GiftAttachment{
			{ID: 1, FileType: "photo", TelegramFileID: " photo-1 "},
			{ID: 2, FileType: "document", TelegramFileID: "doc-1"},
			{ID: 3, FileType: "photo"},
		},
	}

	err := b.notifyAdminAboutGift(context.Background(), gift)
	if err != nil {
		t.Fatalf("notifyAdminAboutGift error: %v", err)
	}
	if len(api.sentPhotos) != 1 {
		t.Fatalf("send photo count mismatch: got %d, want 1", len(api.sentPhotos))
	}
	photo, ok := api.sentPhotos[0].Photo.(*models.InputFileString)
	if !ok {
		t.Fatalf("photo type mismatch: got %T", api.sentPhotos[0].Photo)
	}
	if photo.Data != "photo-1" {
		t.Fatalf("photo file id mismatch: got %q", photo.Data)
	}
	if !strings.Contains(api.sentPhotos[0].Caption, "Описание: Bottle cage") || strings.Contains(api.sentPhotos[0].Caption, "ID подарка") {
		t.Fatalf("caption mismatch: %q", api.sentPhotos[0].Caption)
	}
	if len(api.sentMessages) != 0 {
		t.Fatalf("single photo notification should not send text summary, got %d", len(api.sentMessages))
	}
	markup, ok := api.sentPhotos[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("single photo notification should include miniapp button, got %T", api.sentPhotos[0].ReplyMarkup)
	}
	if !publicMenuHasWebApp(markup, adminGiftMiniappLabel, miniappURL) {
		t.Fatalf("single photo miniapp button not found: %#v", markup)
	}
}

func TestBotHandleGiftConfirmationSendsSingleSuccessWithMenu(t *testing.T) {
	api := &telegramAPIFake{}
	userRepo := newTelegramUserRepoFake()
	userRepo.users[123] = &entity.User{ID: 123}
	eventRepo := &telegramEventRepoFake{event: &entity.Event{ID: 77, Active: true}}
	giftRepo := &telegramGiftRepoFake{}
	participantRepo := &telegramParticipantRepoFake{}
	manager := session.NewManager(0)
	b := &Bot{
		api:                      api,
		sessionManager:           manager,
		userRepo:                 userRepo,
		eventRepo:                eventRepo,
		participantRepo:          participantRepo,
		addGiftHandler:           command.NewAddGiftHandler(userRepo, eventRepo, giftRepo, &telegramBlacklistRepoFake{}),
		isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(&telegramBlacklistRepoFake{}),
	}

	manager.SetState(123, session.StateAwaitingGiftConfirmation)
	manager.SetData(123, "event_id", uint(77))
	manager.SetData(123, "gift_gender", "all")
	manager.SetData(123, "gift_bike_type", "gravel")
	manager.SetData(123, "gift_description", "Bottle cage")

	b.handleStatefulCallback(context.Background(), callbackWithMessage("confirm_gift", 123, 500, 10))

	if giftRepo.createdGift == nil {
		t.Fatal("gift was not created")
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("success message should be sent once, got %d messages: %#v", len(api.sentMessages), api.sentMessages)
	}
	if !strings.Contains(api.sentMessages[0].Text, "Bottle cage") {
		t.Fatalf("success text mismatch: %q", api.sentMessages[0].Text)
	}
	if _, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup); !ok {
		t.Fatalf("success message should include menu markup, got %T", api.sentMessages[0].ReplyMarkup)
	}
}

func TestBotHandleGiftConfirmationNotifiesAdminFromPersistedAttachments(t *testing.T) {
	api := &telegramAPIFake{}
	userRepo := newTelegramUserRepoFake()
	userRepo.users[123] = &entity.User{ID: 123, FirstName: "Alex", Username: "alex"}
	eventRepo := &telegramEventRepoFake{event: &entity.Event{ID: 77, Active: true}}
	giftRepo := &telegramGiftRepoFake{}
	manager := session.NewManager(0)
	b := &Bot{
		api:                      api,
		adminChatID:              900,
		sessionManager:           manager,
		userRepo:                 userRepo,
		eventRepo:                eventRepo,
		participantRepo:          &telegramParticipantRepoFake{},
		addGiftHandler:           command.NewAddGiftHandler(userRepo, eventRepo, giftRepo, &telegramBlacklistRepoFake{}),
		isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(&telegramBlacklistRepoFake{}),
	}

	manager.SetState(123, session.StateAwaitingGiftConfirmation)
	manager.SetData(123, "event_id", uint(77))
	manager.SetData(123, "gift_gender", "female")
	manager.SetData(123, "gift_bike_type", "road")
	manager.SetData(123, "gift_description", "Bottle cage")
	manager.SetData(123, "gift_attachments", []command.GiftAttachmentData{
		{TelegramFileID: "photo-1", FileType: "photo"},
		{TelegramFileID: "doc-1", FileType: "document"},
		{TelegramFileID: "photo-2", FileType: "photo"},
	})

	b.handleStatefulCallback(context.Background(), callbackWithMessage("confirm_gift", 123, 500, 10))

	if giftRepo.createdGift == nil {
		t.Fatal("gift was not created")
	}
	if len(api.mediaGroups) != 1 {
		t.Fatalf("admin media group count mismatch: got %d, want 1", len(api.mediaGroups))
	}
	if got := mediaPhotoIDs(api.mediaGroups[0].Media); !reflect.DeepEqual(got, []string{"photo-1", "photo-2"}) {
		t.Fatalf("admin notification should use persisted photo attachments in order, got %v", got)
	}
	first, ok := api.mediaGroups[0].Media[0].(*models.InputMediaPhoto)
	if !ok {
		t.Fatalf("first media type mismatch: got %T", api.mediaGroups[0].Media[0])
	}
	if !strings.Contains(first.Caption, "От: Alex (@alex)") || !strings.Contains(first.Caption, "Описание: Bottle cage") {
		t.Fatalf("admin caption mismatch: %q", first.Caption)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("private user success should be sent once, got %d messages: %#v", len(api.sentMessages), api.sentMessages)
	}
	if !strings.Contains(api.sentMessages[0].Text, "Bottle cage") {
		t.Fatalf("private success text mismatch: %q", api.sentMessages[0].Text)
	}
	if _, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup); !ok {
		t.Fatalf("private success message should include menu markup, got %T", api.sentMessages[0].ReplyMarkup)
	}
}

func TestBotHandleGiftDescriptionPhotoWithoutCaptionStoresPhotoAndKeepsDraft(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingGiftDesc)
	manager.SetData(userID, "gift_gender", "all")
	manager.SetData(userID, "gift_bike_type", "gravel")
	b.setGiftMessageIDs(userID, []int{44, 45})

	b.handleGiftMessage(context.Background(), &models.Message{
		ID:   100,
		From: &models.User{ID: userID},
		Chat: chatWithID(500),
		Photo: []models.PhotoSize{
			{FileID: "small-photo", Width: 10, Height: 10},
			{FileID: "large-photo", Width: 20, Height: 20},
		},
	}, userID, session.StateAwaitingGiftDesc)

	if got := manager.GetState(userID); got != session.StateAwaitingGiftDesc {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftDesc)
	}
	if _, ok := manager.GetData(userID, "gift_description"); ok {
		t.Fatal("gift_description should still be missing")
	}
	attachments := giftAttachmentsFromTelegramSession(t, manager, userID)
	if len(attachments) != 1 || attachments[0].TelegramFileID != "large-photo" {
		t.Fatalf("attachments mismatch: %#v", attachments)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("draft reply count mismatch: got %d", len(api.sentMessages))
	}
	if !strings.Contains(api.sentMessages[0].Text, "Описание: нужно отправить") {
		t.Fatalf("draft text should require description, got: %s", api.sentMessages[0].Text)
	}
	markup, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("draft reply should include markup, got %T", api.sentMessages[0].ReplyMarkup)
	}
	if got, want := callbackData(markup), []string{"restart_gift", "cancel"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("draft callbacks mismatch: got %v, want %v", got, want)
	}
	if got, want := deletedMessageIDs(api), []int{44, 45}; !reflect.DeepEqual(got, want) {
		t.Fatalf("deleted message IDs mismatch: got %v, want %v", got, want)
	}
	if got, want := b.giftMessageIDs(userID), []int{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("gift message IDs mismatch: got %v, want %v", got, want)
	}
}

func TestBotHandleGiftAlbumStoresEveryPhotoAndSendsOneDraft(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingGiftDesc)
	manager.SetData(userID, "gift_gender", "all")
	manager.SetData(userID, "gift_bike_type", "gravel")

	for _, msg := range []*models.Message{
		{
			ID:           100,
			From:         &models.User{ID: userID},
			Chat:         chatWithID(500),
			MediaGroupID: "album-1",
			Photo:        []models.PhotoSize{{FileID: "photo-1", Width: 20, Height: 20}},
		},
		{
			ID:           101,
			From:         &models.User{ID: userID},
			Chat:         chatWithID(500),
			MediaGroupID: "album-1",
			Photo:        []models.PhotoSize{{FileID: "photo-2", Width: 20, Height: 20}},
		},
	} {
		b.handleGiftMessage(context.Background(), msg, userID, session.StateAwaitingGiftDesc)
	}

	attachments := giftAttachmentsFromTelegramSession(t, manager, userID)
	if got, want := attachmentFileIDs(attachments), []string{"photo-1", "photo-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("attachment IDs mismatch: got %v, want %v", got, want)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("draft reply count mismatch: got %d", len(api.sentMessages))
	}
}

func TestBotHandleGiftPhotoStateCaptionDoesNotOverwriteDescription(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	manager.SetData(userID, "gift_gender", "female")
	manager.SetData(userID, "gift_bike_type", "road")
	manager.SetData(userID, "gift_description", "Original description")

	b.handleGiftMessage(context.Background(), &models.Message{
		ID:      100,
		From:    &models.User{ID: userID},
		Chat:    chatWithID(500),
		Caption: "Replacement caption",
		Photo:   []models.PhotoSize{{FileID: "photo-1", Width: 20, Height: 20}},
	}, userID, session.StateAwaitingGiftPhoto)

	descriptionRaw, ok := manager.GetData(userID, "gift_description")
	if !ok {
		t.Fatal("gift_description missing")
	}
	if descriptionRaw != "Original description" {
		t.Fatalf("gift_description mismatch: got %v", descriptionRaw)
	}
	attachments := giftAttachmentsFromTelegramSession(t, manager, userID)
	if len(attachments) != 1 || attachments[0].TelegramFileID != "photo-1" {
		t.Fatalf("attachments mismatch: %#v", attachments)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("draft reply count mismatch: got %d", len(api.sentMessages))
	}
	markup, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("draft reply should include markup, got %T", api.sentMessages[0].ReplyMarkup)
	}
	if got, want := callbackData(markup), []string{"finish_gift", "restart_gift", "cancel"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("draft callbacks mismatch: got %v, want %v", got, want)
	}
}

func TestBotHandleGiftFinishRequiresDescriptionAndKeepsDraftRecoverable(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingGiftDesc)
	manager.SetData(userID, "event_id", uint(77))
	manager.SetData(userID, "gift_gender", "all")
	manager.SetData(userID, "gift_bike_type", "gravel")
	b.setGiftMessageIDs(userID, []int{40})

	b.handleStatefulCallback(context.Background(), callbackWithMessage("finish_gift", userID, 500, 40))

	if got := manager.GetState(userID); got != session.StateAwaitingGiftDesc {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftDesc)
	}
	if len(api.answerCallbacks) != 1 || api.answerCallbacks[0].Text == "" {
		t.Fatalf("callback answer mismatch: %#v", api.answerCallbacks)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("draft message count mismatch: got %d", len(api.sentMessages))
	}
	if !strings.Contains(api.sentMessages[0].Text, "Описание: нужно отправить") {
		t.Fatalf("draft should require description, got: %s", api.sentMessages[0].Text)
	}
	markup, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("draft reply should include markup, got %T", api.sentMessages[0].ReplyMarkup)
	}
	if got, want := callbackData(markup), []string{"restart_gift", "cancel"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("draft callbacks mismatch: got %v, want %v", got, want)
	}
}

func TestBotHandleGiftFinishShowsPreviewWhenDraftIsComplete(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	setCompleteGiftDraft(manager, userID)
	b.setGiftMessageIDs(userID, []int{40})

	b.handleStatefulCallback(context.Background(), callbackWithMessage("finish_gift", userID, 500, 40))

	if got := manager.GetState(userID); got != session.StateAwaitingGiftConfirmation {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftConfirmation)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("preview message count mismatch: got %d", len(api.sentMessages))
	}
	if !strings.Contains(api.sentMessages[0].Text, "Проверьте приз") {
		t.Fatalf("preview text mismatch: %s", api.sentMessages[0].Text)
	}
	markup, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("preview should include markup, got %T", api.sentMessages[0].ReplyMarkup)
	}
	if got, want := callbackData(markup), []string{"confirm_gift", "restart_gift", "cancel"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("preview callbacks mismatch: got %v, want %v", got, want)
	}
	if got, want := deletedMessageIDs(api), []int{40}; !reflect.DeepEqual(got, want) {
		t.Fatalf("deleted message IDs mismatch: got %v, want %v", got, want)
	}
}

func TestBotHandleGiftLegacySkipPhotosUsesPreviewWhenDraftIsComplete(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	setCompleteGiftDraft(manager, userID)

	b.handleStatefulCallback(context.Background(), callbackWithMessage("skip_photos", userID, 500, 40))

	if got := manager.GetState(userID); got != session.StateAwaitingGiftConfirmation {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftConfirmation)
	}
	if len(api.sentMessages) != 1 || !strings.Contains(api.sentMessages[0].Text, "Проверьте приз") {
		t.Fatalf("preview message mismatch: %#v", api.sentMessages)
	}
}

func TestBotHandleGiftConfirmOutsideConfirmationReturnsToDraftWithoutSaving(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	setCompleteGiftDraft(manager, userID)

	b.handleStatefulCallback(context.Background(), callbackWithMessage("confirm_gift", userID, 500, 40))

	if got := manager.GetState(userID); got != session.StateAwaitingGiftPhoto {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftPhoto)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("draft message count mismatch: got %d", len(api.sentMessages))
	}
	if strings.Contains(api.sentMessages[0].Text, "Приз успешно добавлен") {
		t.Fatalf("stale confirm should not save gift: %s", api.sentMessages[0].Text)
	}
	markup, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("draft should include markup, got %T", api.sentMessages[0].ReplyMarkup)
	}
	if got, want := callbackData(markup), []string{"finish_gift", "restart_gift", "cancel"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("draft callbacks mismatch: got %v, want %v", got, want)
	}
}

func TestBotHandleGiftRestartFromUnrelatedStateDoesNotMutateSession(t *testing.T) {
	api := &telegramAPIFake{}
	manager := session.NewManager(0)
	userID := int64(123)
	b := &Bot{
		api:            api,
		sessionManager: manager,
	}

	manager.SetState(userID, session.StateAwaitingResultLink)
	manager.SetData(userID, "participant_id", uint(55))

	b.handleStatefulCallback(context.Background(), callbackWithMessage("restart_gift", userID, 500, 40))

	if got := manager.GetState(userID); got != session.StateAwaitingResultLink {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingResultLink)
	}
	if value, ok := manager.GetData(userID, "participant_id"); !ok || value != uint(55) {
		t.Fatalf("session data mutated: value=%v ok=%t", value, ok)
	}
	if len(api.answerCallbacks) != 1 {
		t.Fatalf("answer callback count mismatch: got %d", len(api.answerCallbacks))
	}
	if len(api.sentMessages) != 0 {
		t.Fatalf("unrelated restart should not send messages, got %#v", api.sentMessages)
	}
}

func TestBotNotifyAdminAboutGiftSendsMediaGroupForMultiplePhotos(t *testing.T) {
	const miniappURL = "https://example.com/miniapp/gifts"
	api := &telegramAPIFake{}
	b := &Bot{
		api:         api,
		adminChatID: 900,
		botUsername: "GravelBot",
		miniappURL:  miniappURL,
	}
	gift := &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		User:           &entity.User{ID: 123, FirstName: "Alex"},
		Attachments: []entity.GiftAttachment{
			{ID: 1, FileType: "photo", TelegramFileID: "photo-1"},
			{ID: 2, FileType: "document", TelegramFileID: "doc-1"},
			{ID: 3, FileType: "photo", TelegramFileID: "photo-2"},
		},
	}

	err := b.notifyAdminAboutGift(context.Background(), gift)
	if err != nil {
		t.Fatalf("notifyAdminAboutGift error: %v", err)
	}
	if len(api.mediaGroups) != 1 {
		t.Fatalf("media group count mismatch: got %d, want 1", len(api.mediaGroups))
	}
	if got := mediaPhotoIDs(api.mediaGroups[0].Media); !reflect.DeepEqual(got, []string{"photo-1", "photo-2"}) {
		t.Fatalf("media photo order mismatch: got %v", got)
	}
	first, ok := api.mediaGroups[0].Media[0].(*models.InputMediaPhoto)
	if !ok {
		t.Fatalf("first media type mismatch: got %T", api.mediaGroups[0].Media[0])
	}
	if !strings.Contains(first.Caption, "Описание: Bottle cage") {
		t.Fatalf("first media caption mismatch: %q", first.Caption)
	}
	if first.ParseMode != models.ParseModeHTML {
		t.Fatalf("first media parse mode mismatch: got %q, want %q", first.ParseMode, models.ParseModeHTML)
	}
	wantLink := `<a href="https://t.me/GravelBot?startapp">призовой фонд</a>`
	if !strings.Contains(first.Caption, wantLink) {
		t.Fatalf("first media caption missing hidden miniapp link %q in %q", wantLink, first.Caption)
	}
	second, ok := api.mediaGroups[0].Media[1].(*models.InputMediaPhoto)
	if !ok {
		t.Fatalf("second media type mismatch: got %T", api.mediaGroups[0].Media[1])
	}
	if second.Caption != "" {
		t.Fatalf("only first media item should have caption: %q", second.Caption)
	}
	if len(api.sentMessages) != 0 {
		t.Fatalf("successful media group should not send text summary, got %d", len(api.sentMessages))
	}
}

func TestBotNotifyAdminAboutGiftSendsTextWhenNoUsablePhotos(t *testing.T) {
	const miniappURL = "https://example.com/miniapp/gifts"
	api := &telegramAPIFake{}
	b := &Bot{
		api:         api,
		adminChatID: 900,
		miniappURL:  miniappURL,
	}
	gift := &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "road",
		User:           &entity.User{ID: 123, Username: "alex"},
		Attachments: []entity.GiftAttachment{
			{ID: 1, FileType: "document", TelegramFileID: "doc-1"},
			{ID: 2, FileType: "photo", TelegramFileID: " "},
		},
	}

	err := b.notifyAdminAboutGift(context.Background(), gift)
	if err != nil {
		t.Fatalf("notifyAdminAboutGift error: %v", err)
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("text notification count mismatch: got %d, want 1", len(api.sentMessages))
	}
	if len(api.sentPhotos) != 0 || len(api.mediaGroups) != 0 {
		t.Fatalf("no usable photos should send text only: photos=%d groups=%d", len(api.sentPhotos), len(api.mediaGroups))
	}
	if !strings.Contains(api.sentMessages[0].Text, "От: @alex") || !strings.Contains(api.sentMessages[0].Text, "Велосипед: 🚴 Шоссе") {
		t.Fatalf("text notification mismatch: %q", api.sentMessages[0].Text)
	}
	markup, ok := api.sentMessages[0].ReplyMarkup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("text notification should include miniapp button, got %T", api.sentMessages[0].ReplyMarkup)
	}
	if !publicMenuHasWebApp(markup, adminGiftMiniappLabel, miniappURL) {
		t.Fatalf("text miniapp button not found: %#v", markup)
	}
}

func TestBotNotifyAdminAboutGiftFallsBackToTextWhenPhotoSendFails(t *testing.T) {
	api := &telegramAPIFake{
		sendPhotoErr: errors.New("photo failed"),
	}
	b := &Bot{
		api:         api,
		adminChatID: 900,
	}
	gift := &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		ReviewStatus:   entity.GiftReviewStatusPendingReview,
		User:           &entity.User{ID: 123, FirstName: "Alex"},
		Attachments:    []entity.GiftAttachment{{ID: 1, FileType: "photo", TelegramFileID: "photo-1"}},
	}

	err := b.notifyAdminAboutGift(context.Background(), gift)
	if err != nil {
		t.Fatalf("notifyAdminAboutGift should keep user flow successful with text fallback, got %v", err)
	}
	if len(api.sentPhotos) != 1 {
		t.Fatalf("send photo should be attempted once, got %d", len(api.sentPhotos))
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("text fallback count mismatch: got %d, want 1", len(api.sentMessages))
	}
	if !strings.Contains(api.sentMessages[0].Text, "Описание: ") || strings.Contains(api.sentMessages[0].Text, "ID подарка") {
		t.Fatalf("fallback text mismatch: %q", api.sentMessages[0].Text)
	}
}

func TestBotNotifyAdminAboutGiftFallsBackToTextWhenMediaGroupSendFails(t *testing.T) {
	api := &telegramAPIFake{
		mediaGroupErr: errors.New("media group failed"),
	}
	b := &Bot{
		api:         api,
		adminChatID: 900,
	}
	gift := &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		User:           &entity.User{ID: 123, FirstName: "Alex"},
		Attachments: []entity.GiftAttachment{
			{ID: 1, FileType: "photo", TelegramFileID: "photo-1"},
			{ID: 2, FileType: "photo", TelegramFileID: "photo-2"},
		},
	}

	err := b.notifyAdminAboutGift(context.Background(), gift)
	if err != nil {
		t.Fatalf("notifyAdminAboutGift should keep user flow successful with text fallback, got %v", err)
	}
	if len(api.mediaGroups) != 1 {
		t.Fatalf("media group should be attempted once, got %d", len(api.mediaGroups))
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("text fallback count mismatch: got %d, want 1", len(api.sentMessages))
	}
	if !strings.Contains(api.sentMessages[0].Text, "Описание: Bottle cage") || strings.Contains(api.sentMessages[0].Text, "ID события") {
		t.Fatalf("fallback text mismatch: %q", api.sentMessages[0].Text)
	}
}

func TestBotNotifyAdminAboutGiftSendsAllPhotosInValidMediaGroupChunks(t *testing.T) {
	api := &telegramAPIFake{}
	b := &Bot{
		api:         api,
		adminChatID: 900,
	}
	attachments := make([]entity.GiftAttachment, 0, 21)
	wantPhotoIDs := make([]string, 0, 21)
	for i := 1; i <= 21; i++ {
		fileID := fmt.Sprintf("photo-%02d", i)
		attachments = append(attachments, entity.GiftAttachment{ID: uint(i), FileType: "photo", TelegramFileID: fileID})
		wantPhotoIDs = append(wantPhotoIDs, fileID)
	}
	gift := &entity.Gift{
		ID:             10,
		EventID:        77,
		UserID:         123,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		User:           &entity.User{ID: 123, FirstName: "Alex"},
		Attachments:    attachments,
	}

	err := b.notifyAdminAboutGift(context.Background(), gift)
	if err != nil {
		t.Fatalf("notifyAdminAboutGift error: %v", err)
	}
	if len(api.mediaGroups) != 3 {
		t.Fatalf("media group chunk count mismatch: got %d, want 3", len(api.mediaGroups))
	}
	if got := mediaGroupSizes(api.mediaGroups); !reflect.DeepEqual(got, []int{10, 9, 2}) {
		t.Fatalf("media group chunk sizes mismatch: got %v", got)
	}
	if got := mediaGroupPhotoIDs(api.mediaGroups); !reflect.DeepEqual(got, wantPhotoIDs) {
		t.Fatalf("media group photo order mismatch: got %v, want %v", got, wantPhotoIDs)
	}
	first, ok := api.mediaGroups[0].Media[0].(*models.InputMediaPhoto)
	if !ok {
		t.Fatalf("first media type mismatch: got %T", api.mediaGroups[0].Media[0])
	}
	if !strings.Contains(first.Caption, "Описание: Bottle cage") {
		t.Fatalf("first chunk caption mismatch: %q", first.Caption)
	}
	for groupIndex, group := range api.mediaGroups[1:] {
		photo, ok := group.Media[0].(*models.InputMediaPhoto)
		if !ok {
			t.Fatalf("chunk %d first media type mismatch: got %T", groupIndex+2, group.Media[0])
		}
		if photo.Caption != "" {
			t.Fatalf("only first media item of first chunk should have caption, chunk=%d caption=%q", groupIndex+2, photo.Caption)
		}
	}
}

func TestBotChatLogMarkerRedactsConfiguredChats(t *testing.T) {
	b := &Bot{adminChatID: 900, publicChatID: -100}

	if got := b.chatLogMarker(900); got != "admin" {
		t.Fatalf("admin marker mismatch: got %q", got)
	}
	if got := b.chatLogMarker(-100); got != "public" {
		t.Fatalf("public marker mismatch: got %q", got)
	}
	if got := b.chatLogMarker(123); got != "private_or_other" {
		t.Fatalf("private marker mismatch: got %q", got)
	}
}

type telegramAPIFake struct {
	sentMessages    []*telegrambot.SendMessageParams
	sentPhotos      []*telegrambot.SendPhotoParams
	mediaGroups     []*telegrambot.SendMediaGroupParams
	editMessages    []*telegrambot.EditMessageTextParams
	answerCallbacks []*telegrambot.AnswerCallbackQueryParams
	deleteMessages  []*telegrambot.DeleteMessageParams
	sendErr         error
	sendPhotoErr    error
	mediaGroupErr   error
	editErr         error
	answerErr       error
	deleteErr       error
	nextMessageID   int
}

func (a *telegramAPIFake) Start(ctx context.Context) {}

func (a *telegramAPIFake) SendMessage(ctx context.Context, params *telegrambot.SendMessageParams) (*models.Message, error) {
	if strings.TrimSpace(params.Text) == "" {
		return nil, errors.New("telegram message text is empty")
	}
	if a.sendErr != nil {
		return nil, a.sendErr
	}
	a.sentMessages = append(a.sentMessages, params)
	a.nextMessageID++
	return &models.Message{ID: a.nextMessageID, Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}, Text: params.Text}, nil
}

func (a *telegramAPIFake) SendPhoto(ctx context.Context, params *telegrambot.SendPhotoParams) (*models.Message, error) {
	if strings.TrimSpace(params.Caption) == "" {
		return nil, errors.New("telegram photo caption is empty")
	}
	a.sentPhotos = append(a.sentPhotos, params)
	if a.sendPhotoErr != nil {
		return nil, a.sendPhotoErr
	}
	a.nextMessageID++
	return &models.Message{ID: a.nextMessageID, Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}, Caption: params.Caption}, nil
}

func (a *telegramAPIFake) SendMediaGroup(ctx context.Context, params *telegrambot.SendMediaGroupParams) ([]*models.Message, error) {
	if len(params.Media) < 2 || len(params.Media) > telegramMediaGroupLimit {
		return nil, fmt.Errorf("telegram media group size invalid: %d", len(params.Media))
	}
	a.mediaGroups = append(a.mediaGroups, params)
	if a.mediaGroupErr != nil {
		return nil, a.mediaGroupErr
	}
	messages := make([]*models.Message, 0, len(params.Media))
	for range params.Media {
		a.nextMessageID++
		messages = append(messages, &models.Message{ID: a.nextMessageID, Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}})
	}
	return messages, nil
}

func (a *telegramAPIFake) EditMessageText(ctx context.Context, params *telegrambot.EditMessageTextParams) (*models.Message, error) {
	a.editMessages = append(a.editMessages, params)
	if a.editErr != nil {
		return nil, a.editErr
	}
	return &models.Message{ID: params.MessageID, Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}, Text: params.Text}, nil
}

func (a *telegramAPIFake) AnswerCallbackQuery(ctx context.Context, params *telegrambot.AnswerCallbackQueryParams) (bool, error) {
	a.answerCallbacks = append(a.answerCallbacks, params)
	if a.answerErr != nil {
		return false, a.answerErr
	}
	return true, nil
}

func (a *telegramAPIFake) DeleteMessage(ctx context.Context, params *telegrambot.DeleteMessageParams) (bool, error) {
	a.deleteMessages = append(a.deleteMessages, params)
	if a.deleteErr != nil {
		return false, a.deleteErr
	}
	return true, nil
}

func chatIDFromAny(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func callbackWithMessage(data string, userID int64, chatID int64, messageID int) *models.CallbackQuery {
	return &models.CallbackQuery{
		ID:   "callback-id",
		From: models.User{ID: userID, FirstName: "Alex"},
		Data: data,
		Message: models.MaybeInaccessibleMessage{
			Message: &models.Message{ID: messageID, Chat: chatWithID(chatID)},
		},
	}
}

func commandMessage(text string, userID int64, chatID int64) *models.Message {
	return &models.Message{
		ID:   10,
		From: &models.User{ID: userID, FirstName: "Alex"},
		Chat: chatWithID(chatID),
		Text: text,
		Entities: []models.MessageEntity{
			{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: len(text)},
		},
	}
}

func chatWithID(chatID int64) models.Chat {
	chatType := models.ChatTypePrivate
	if chatID < 0 {
		chatType = models.ChatTypeSupergroup
	}

	return models.Chat{ID: chatID, Type: chatType}
}

func callbackData(menu models.InlineKeyboardMarkup) []string {
	var data []string
	for _, row := range menu.InlineKeyboard {
		for _, button := range row {
			if button.CallbackData != "" {
				data = append(data, button.CallbackData)
			}
		}
	}
	return data
}

func publicMenuHasURL(markup models.InlineKeyboardMarkup, text string, url string) bool {
	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			if button.Text == text && button.URL == url {
				return true
			}
		}
	}
	return false
}

func publicMenuHasWebApp(markup models.InlineKeyboardMarkup, text string, webAppURL string) bool {
	for _, row := range markup.InlineKeyboard {
		for _, button := range row {
			if button.Text == text && button.WebApp != nil && button.WebApp.URL == webAppURL {
				return true
			}
		}
	}
	return false
}

func sentTextContains(api *telegramAPIFake, needle string) bool {
	for _, msg := range api.sentMessages {
		if strings.Contains(msg.Text, needle) {
			return true
		}
	}
	return false
}

func deletedMessageIDs(api *telegramAPIFake) []int {
	ids := make([]int, 0, len(api.deleteMessages))
	for _, msg := range api.deleteMessages {
		ids = append(ids, msg.MessageID)
	}
	return ids
}

func giftAttachmentsFromTelegramSession(t *testing.T, manager *session.Manager, userID int64) []command.GiftAttachmentData {
	t.Helper()

	attachmentsRaw, ok := manager.GetData(userID, "gift_attachments")
	if !ok {
		t.Fatal("gift_attachments missing from session")
	}
	attachments, ok := attachmentsRaw.([]command.GiftAttachmentData)
	if !ok {
		t.Fatalf("gift_attachments type mismatch: got %T", attachmentsRaw)
	}
	return attachments
}

func attachmentFileIDs(attachments []command.GiftAttachmentData) []string {
	ids := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		ids = append(ids, attachment.TelegramFileID)
	}
	return ids
}

func setCompleteGiftDraft(manager *session.Manager, userID int64) {
	manager.SetData(userID, "event_id", uint(77))
	manager.SetData(userID, "gift_gender", "all")
	manager.SetData(userID, "gift_bike_type", "gravel")
	manager.SetData(userID, "gift_description", "Bottle cage")
}

func mediaPhotoIDs(media []models.InputMedia) []string {
	ids := make([]string, 0, len(media))
	for _, item := range media {
		photo, ok := item.(*models.InputMediaPhoto)
		if !ok {
			continue
		}
		ids = append(ids, photo.Media)
	}
	return ids
}

func mediaGroupSizes(groups []*telegrambot.SendMediaGroupParams) []int {
	sizes := make([]int, 0, len(groups))
	for _, group := range groups {
		sizes = append(sizes, len(group.Media))
	}
	return sizes
}

func mediaGroupPhotoIDs(groups []*telegrambot.SendMediaGroupParams) []string {
	ids := make([]string, 0)
	for _, group := range groups {
		ids = append(ids, mediaPhotoIDs(group.Media)...)
	}
	return ids
}

type telegramUserRepoFake struct {
	users map[int64]*entity.User
}

func newTelegramUserRepoFake() *telegramUserRepoFake {
	return &telegramUserRepoFake{users: make(map[int64]*entity.User)}
}

func (r *telegramUserRepoFake) Create(ctx context.Context, user *entity.User) error {
	if r.users == nil {
		r.users = make(map[int64]*entity.User)
	}
	userCopy := *user
	r.users[user.ID] = &userCopy
	return nil
}

func (r *telegramUserRepoFake) Update(ctx context.Context, user *entity.User) error { return nil }

func (r *telegramUserRepoFake) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	if r.users == nil {
		return nil, fmt.Errorf("user not found")
	}
	user, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (r *telegramUserRepoFake) Delete(ctx context.Context, id int64) error { return nil }
func (r *telegramUserRepoFake) GetAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

type telegramEventRepoFake struct {
	event           *entity.Event
	findActiveErr   error
	findActiveCalls int
}

func (r *telegramEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *telegramEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *telegramEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *telegramEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *telegramEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	r.findActiveCalls++
	if r.findActiveErr != nil {
		return nil, r.findActiveErr
	}
	return r.event, nil
}
func (r *telegramEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *telegramEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type telegramParticipantRepoFake struct {
	participant          *entity.Participant
	deletedParticipantID uint
}

func (r *telegramParticipantRepoFake) Create(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *telegramParticipantRepoFake) Update(ctx context.Context, participant *entity.Participant) error {
	return nil
}
func (r *telegramParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	if r.participant != nil && r.participant.ID == id {
		return r.participant, nil
	}
	return nil, repository.ErrParticipantNotFound
}
func (r *telegramParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	if r.participant != nil && r.participant.UserID == userID && r.participant.EventID == eventID {
		return r.participant, nil
	}
	return nil, repository.ErrParticipantNotFound
}
func (r *telegramParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
func (r *telegramParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *telegramParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *telegramParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	if r.participant == nil || r.participant.ID != id {
		return repository.ErrParticipantNotFound
	}
	r.deletedParticipantID = id
	r.participant = nil
	return nil
}
func (r *telegramParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}

type telegramGiftRepoFake struct {
	createdGift *entity.Gift
}

func (r *telegramGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *telegramGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	gift.ID = 901
	for i, attachment := range attachments {
		attachment.ID = uint(i + 1)
		attachment.GiftID = gift.ID
	}
	r.createdGift = gift
	return nil
}
func (r *telegramGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *telegramGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *telegramGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *telegramGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *telegramGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *telegramGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *telegramGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *telegramGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *telegramGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}

type telegramBlacklistRepoFake struct {
	blacklisted            map[int64]bool
	checkedTelegramUserIDs []int64
}

func (r *telegramBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *telegramBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *telegramBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	r.checkedTelegramUserIDs = append(r.checkedTelegramUserIDs, telegramUserID)
	return r.blacklisted != nil && r.blacklisted[telegramUserID], nil
}
func (r *telegramBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	return nil
}
func (r *telegramBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *telegramBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return nil
}
