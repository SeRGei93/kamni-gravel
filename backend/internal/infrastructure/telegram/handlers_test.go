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

func TestBotNotifyAdminAboutGiftCopiesSourcesInOrderAndSendsSummary(t *testing.T) {
	api := &telegramAPIFake{}
	b := &Bot{
		api:         api,
		adminChatID: 900,
	}
	gift := &entity.Gift{ID: 10, EventID: 77, UserID: 123, ReviewStatus: entity.GiftReviewStatusPendingReview}

	err := b.notifyAdminAboutGift(context.Background(), gift, []giftSourceRef{
		{ChatID: 500, MessageID: 10, UpdateKind: "text"},
		{ChatID: 500, MessageID: 11, UpdateKind: "photo"},
	})
	if err != nil {
		t.Fatalf("notifyAdminAboutGift error: %v", err)
	}
	if len(api.copyMessages) != 2 {
		t.Fatalf("copy count mismatch: got %d, want 2", len(api.copyMessages))
	}
	if api.copyMessages[0].MessageID != 10 || api.copyMessages[1].MessageID != 11 {
		t.Fatalf("copy order mismatch: %#v", api.copyMessages)
	}
	if len(api.forwardMessages) != 0 {
		t.Fatalf("unexpected forward calls: %d", len(api.forwardMessages))
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("summary send count mismatch: got %d, want 1", len(api.sentMessages))
	}
	if !strings.Contains(api.sentMessages[0].Text, "ID подарка: 10") {
		t.Fatalf("summary text mismatch: %q", api.sentMessages[0].Text)
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

func TestBotNotifyAdminAboutGiftSendsSummaryFallbackWhenCopyAndForwardFail(t *testing.T) {
	api := &telegramAPIFake{
		copyErr:    errors.New("copy failed"),
		forwardErr: errors.New("forward failed"),
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
		Attachments:    []entity.GiftAttachment{{ID: 1}},
	}

	err := b.notifyAdminAboutGift(context.Background(), gift, []giftSourceRef{
		{ChatID: 500, MessageID: 10, UpdateKind: "text"},
	})
	if err != nil {
		t.Fatalf("notifyAdminAboutGift should keep user flow successful with summary fallback, got %v", err)
	}
	if len(api.copyMessages) != 1 || len(api.forwardMessages) != 1 {
		t.Fatalf("copy/forward calls mismatch: copies=%d forwards=%d", len(api.copyMessages), len(api.forwardMessages))
	}
	if len(api.sentMessages) != 1 {
		t.Fatalf("summary send count mismatch: got %d, want 1", len(api.sentMessages))
	}
	if !strings.Contains(api.sentMessages[0].Text, "ID подарка: 10") || !strings.Contains(api.sentMessages[0].Text, "Фото: 1") {
		t.Fatalf("summary text mismatch: %q", api.sentMessages[0].Text)
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
	copyMessages    []*telegrambot.CopyMessageParams
	forwardMessages []*telegrambot.ForwardMessageParams
	editMessages    []*telegrambot.EditMessageTextParams
	answerCallbacks []*telegrambot.AnswerCallbackQueryParams
	deleteMessages  []*telegrambot.DeleteMessageParams
	sendErr         error
	copyErr         error
	forwardErr      error
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

func (a *telegramAPIFake) CopyMessage(ctx context.Context, params *telegrambot.CopyMessageParams) (*models.MessageID, error) {
	a.copyMessages = append(a.copyMessages, params)
	if a.copyErr != nil {
		return nil, a.copyErr
	}
	return &models.MessageID{ID: params.MessageID}, nil
}

func (a *telegramAPIFake) ForwardMessage(ctx context.Context, params *telegrambot.ForwardMessageParams) (*models.Message, error) {
	a.forwardMessages = append(a.forwardMessages, params)
	if a.forwardErr != nil {
		return nil, a.forwardErr
	}
	a.nextMessageID++
	return &models.Message{ID: a.nextMessageID, Chat: models.Chat{ID: chatIDFromAny(params.ChatID)}}, nil
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
