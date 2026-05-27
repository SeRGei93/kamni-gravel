package handler

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestGiftHandlerStartAddGiftUsesEventTelegramTexts(t *testing.T) {
	manager := session.NewManager(time.Minute)
	texts := entity.DefaultEventTelegramTexts()
	texts.GiftGenderStep = "custom gift gender step"
	h := NewGiftHandler(
		manager,
		&giftConfirmEventRepoFake{event: &entity.Event{ID: 77, TelegramTexts: texts}},
		nil,
	)

	text, markup := h.StartAddGift(context.Background(), 123)

	if text != "custom gift gender step" {
		t.Fatalf("text mismatch: got %q", text)
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	textsRaw, ok := manager.GetData(123, "event_telegram_texts")
	if !ok {
		t.Fatal("event_telegram_texts missing from session")
	}
	if got, ok := textsRaw.(entity.EventTelegramTexts); !ok || got.GiftGenderStep != "custom gift gender step" {
		t.Fatalf("event_telegram_texts mismatch: got %#v", textsRaw)
	}
}

func TestGiftHandlerDefaultDescriptionPromptMentionsInputField(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)

	text, markup := h.HandleGiftBikeTypeSelection(context.Background(), 123, "gravel")

	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if !strings.Contains(text, "поле ввода ниже") {
		t.Fatalf("text should mention input field, got: %s", text)
	}
}

func TestGiftHandlerHandleGiftDescriptionSetsPhotoState(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)

	text, markup := h.HandleGiftDescription(context.Background(), 123, "  Bottle cage  ")

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if got := manager.GetState(123); got != session.StateAwaitingGiftPhoto {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftPhoto)
	}

	descriptionRaw, ok := manager.GetData(123, "gift_description")
	if !ok {
		t.Fatal("gift_description missing from session")
	}
	if descriptionRaw != "Bottle cage" {
		t.Fatalf("gift_description mismatch: got %v, want Bottle cage", descriptionRaw)
	}
}

func TestGiftHandlerHandleGiftDescriptionRejectsEmptyInput(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)
	userID := int64(123)

	manager.SetState(userID, session.StateAwaitingGiftDesc)

	text, markup := h.HandleGiftDescription(context.Background(), userID, " \n\t ")

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if got := manager.GetState(userID); got != session.StateAwaitingGiftDesc {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftDesc)
	}
	if _, ok := manager.GetData(userID, "gift_description"); ok {
		t.Fatal("gift_description should not be stored for empty input")
	}
}

func TestGiftHandlerDescriptionCaptionAndPhotoStoresBoth(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)
	userID := int64(123)

	manager.SetState(userID, session.StateAwaitingGiftDesc)

	_, _ = h.HandleGiftDescription(context.Background(), userID, "Bottle cage")
	photoCount := h.AppendGiftPhoto(userID, "telegram-file-id")

	if photoCount != 1 {
		t.Fatalf("photo count mismatch: got %d, want 1", photoCount)
	}
	if got := manager.GetState(userID); got != session.StateAwaitingGiftPhoto {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftPhoto)
	}

	descriptionRaw, ok := manager.GetData(userID, "gift_description")
	if !ok {
		t.Fatal("gift_description missing from session")
	}
	if descriptionRaw != "Bottle cage" {
		t.Fatalf("gift_description mismatch: got %v, want Bottle cage", descriptionRaw)
	}

	attachments := giftAttachmentsFromSession(t, manager, userID)
	if len(attachments) != 1 {
		t.Fatalf("attachment count mismatch: got %d, want 1", len(attachments))
	}
	if attachments[0].TelegramFileID != "telegram-file-id" {
		t.Fatalf("telegram file id mismatch: got %q", attachments[0].TelegramFileID)
	}
}

func TestGiftHandlerHandleGiftPhotoTracksAttachment(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)

	text := h.HandleGiftPhoto(123, "telegram-file-id")

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}

	attachmentsRaw, ok := manager.GetData(123, "gift_attachments")
	if !ok {
		t.Fatal("gift_attachments missing from session")
	}

	attachments, ok := attachmentsRaw.([]command.GiftAttachmentData)
	if !ok {
		t.Fatalf("gift_attachments type mismatch: got %T", attachmentsRaw)
	}
	if len(attachments) != 1 {
		t.Fatalf("attachment count mismatch: got %d, want 1", len(attachments))
	}
	if attachments[0].TelegramFileID != "telegram-file-id" {
		t.Fatalf("telegram file id mismatch: got %q", attachments[0].TelegramFileID)
	}
	if attachments[0].FileType != "photo" {
		t.Fatalf("file type mismatch: got %q, want photo", attachments[0].FileType)
	}
}

func TestGiftHandlerAppendGiftPhotoPreservesExistingDescription(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)
	userID := int64(123)

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	manager.SetData(userID, "gift_description", "Bottle cage")

	photoCount := h.AppendGiftPhoto(userID, "telegram-file-id")

	if photoCount != 1 {
		t.Fatalf("photo count mismatch: got %d, want 1", photoCount)
	}
	descriptionRaw, ok := manager.GetData(userID, "gift_description")
	if !ok {
		t.Fatal("gift_description missing from session")
	}
	if descriptionRaw != "Bottle cage" {
		t.Fatalf("gift_description mismatch: got %v, want Bottle cage", descriptionRaw)
	}
}

func TestGiftHandlerGiftPromptsReturnContextualMarkup(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)
	userID := int64(123)

	tests := []struct {
		name string
		run  func() (string, *models.InlineKeyboardMarkup)
	}{
		{name: "gender", run: func() (string, *models.InlineKeyboardMarkup) { return h.GiftGenderPrompt(userID) }},
		{name: "bike", run: func() (string, *models.InlineKeyboardMarkup) { return h.GiftBikeTypePrompt(userID) }},
		{name: "description", run: func() (string, *models.InlineKeyboardMarkup) { return h.GiftDescriptionPrompt(userID) }},
		{name: "photo", run: func() (string, *models.InlineKeyboardMarkup) { return h.GiftPhotoPrompt(userID) }},
		{name: "confirmation", run: func() (string, *models.InlineKeyboardMarkup) { return h.GiftConfirmationPrompt(userID) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, markup := tt.run()
			if text == "" {
				t.Fatal("text mismatch: got empty text")
			}
			if markup == nil {
				t.Fatal("markup mismatch: got nil")
			}
		})
	}
}

func TestGiftHandlerPreviewGiftSetsConfirmationState(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)
	userID := int64(123)

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	manager.SetData(userID, "event_id", uint(77))
	manager.SetData(userID, "gift_gender", "all")
	manager.SetData(userID, "gift_bike_type", "gravel")
	manager.SetData(userID, "gift_description", "Bottle cage")
	manager.SetData(userID, "gift_attachments", []command.GiftAttachmentData{
		{TelegramFileID: "telegram-file-id", FileType: "photo"},
	})

	text, markup := h.PreviewGift(userID)

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}
	if markup == nil {
		t.Fatal("markup mismatch: got nil")
	}
	if got := manager.GetState(userID); got != session.StateAwaitingGiftConfirmation {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateAwaitingGiftConfirmation)
	}
	if got, want := giftCallbackData(*markup), []string{"confirm_gift", "restart_gift", "cancel"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("callback data mismatch: got %v, want %v", got, want)
	}
}

func TestGiftHandlerPreviewGiftRejectsMalformedSessionData(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)
	userID := int64(123)

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	manager.SetData(userID, "event_id", "bad-event-id")
	manager.SetData(userID, "gift_gender", "all")
	manager.SetData(userID, "gift_bike_type", "gravel")
	manager.SetData(userID, "gift_description", "Bottle cage")

	text, markup := h.PreviewGift(userID)

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}
	if markup != nil {
		t.Fatal("markup mismatch: got confirmation markup for invalid session")
	}
	if got := manager.GetState(userID); got != session.StateIdle {
		t.Fatalf("state mismatch: got %s, want %s", got, session.StateIdle)
	}
}

func TestGiftHandlerHandleGiftPhotoDoesNotPanicOnMalformedAttachments(t *testing.T) {
	manager := session.NewManager(time.Minute)
	h := NewGiftHandler(manager, nil, nil)
	userID := int64(123)

	manager.SetState(userID, session.StateAwaitingGiftPhoto)
	manager.SetData(userID, "gift_attachments", "bad-attachments")

	text := h.HandleGiftPhoto(userID, "telegram-file-id")

	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}
	attachmentsRaw, ok := manager.GetData(userID, "gift_attachments")
	if !ok {
		t.Fatal("gift_attachments missing from session")
	}
	attachments, ok := attachmentsRaw.([]command.GiftAttachmentData)
	if !ok {
		t.Fatalf("gift_attachments type mismatch: got %T", attachmentsRaw)
	}
	if len(attachments) != 1 {
		t.Fatalf("attachment count mismatch: got %d, want 1", len(attachments))
	}
	if attachments[0].TelegramFileID != "telegram-file-id" {
		t.Fatalf("telegram file id mismatch: got %q", attachments[0].TelegramFileID)
	}
}

func TestGiftHandlerConfirmAddGiftPersistsTelegramUserIDAndPendingStatus(t *testing.T) {
	manager := session.NewManager(time.Minute)
	giftRepo := &giftConfirmGiftRepoFake{}
	addGiftHandler := command.NewAddGiftHandler(
		&giftConfirmUserRepoFake{user: &entity.User{ID: 12345}},
		&giftConfirmEventRepoFake{event: &entity.Event{ID: 77}},
		giftRepo,
		&giftConfirmUserBlacklistRepoFake{},
	)
	h := NewGiftHandler(manager, nil, addGiftHandler)
	userID := int64(12345)

	manager.SetState(userID, session.StateAwaitingGiftConfirmation)
	manager.SetData(userID, "event_id", uint(77))
	manager.SetData(userID, "gift_gender", "all")
	manager.SetData(userID, "gift_bike_type", "gravel")
	manager.SetData(userID, "gift_description", "Bottle cage")
	manager.SetData(userID, "gift_attachments", []command.GiftAttachmentData{
		{TelegramFileID: "file-1", FileType: "photo"},
	})

	gift, text, err := h.ConfirmAddGift(context.Background(), userID)
	if err != nil {
		t.Fatalf("ConfirmAddGift error: %v", err)
	}
	if gift == nil {
		t.Fatal("gift mismatch: got nil")
	}
	if gift.ID != 99 {
		t.Fatalf("gift id mismatch: got %d, want %d", gift.ID, 99)
	}
	if text == "" {
		t.Fatal("text mismatch: got empty text")
	}
	if giftRepo.createdGift.UserID != userID {
		t.Fatalf("gift user mismatch: got %d, want %d", giftRepo.createdGift.UserID, userID)
	}
	if giftRepo.createdGift.ReviewStatus != entity.GiftReviewStatusPendingReview {
		t.Fatalf("review status mismatch: got %s", giftRepo.createdGift.ReviewStatus)
	}
	if got := manager.GetState(userID); got != session.StateIdle {
		t.Fatalf("state mismatch after confirm: got %s, want %s", got, session.StateIdle)
	}
}

func giftCallbackData(menu models.InlineKeyboardMarkup) []string {
	var data []string
	for _, row := range menu.InlineKeyboard {
		for _, button := range row {
			data = append(data, button.CallbackData)
		}
	}
	return data
}

func giftAttachmentsFromSession(t *testing.T, manager *session.Manager, userID int64) []command.GiftAttachmentData {
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

type giftConfirmUserRepoFake struct {
	user *entity.User
}

func (r *giftConfirmUserRepoFake) Create(ctx context.Context, user *entity.User) error { return nil }
func (r *giftConfirmUserRepoFake) Update(ctx context.Context, user *entity.User) error { return nil }
func (r *giftConfirmUserRepoFake) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	return r.user, nil
}
func (r *giftConfirmUserRepoFake) Delete(ctx context.Context, id int64) error { return nil }
func (r *giftConfirmUserRepoFake) GetAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

type giftConfirmUserBlacklistRepoFake struct {
	blacklisted bool
}

func (r *giftConfirmUserBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *giftConfirmUserBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *giftConfirmUserBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	return r.blacklisted, nil
}
func (r *giftConfirmUserBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	return nil
}
func (r *giftConfirmUserBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *giftConfirmUserBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return nil
}

type giftConfirmEventRepoFake struct {
	event *entity.Event
}

func (r *giftConfirmEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *giftConfirmEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *giftConfirmEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *giftConfirmEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *giftConfirmEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *giftConfirmEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *giftConfirmEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type giftConfirmGiftRepoFake struct {
	createdGift *entity.Gift
}

func (r *giftConfirmGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *giftConfirmGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	r.createdGift = gift
	gift.ID = 99
	for i, attachment := range attachments {
		attachment.ID = uint(i + 1)
		attachment.GiftID = gift.ID
	}
	return nil
}
func (r *giftConfirmGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *giftConfirmGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *giftConfirmGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *giftConfirmGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *giftConfirmGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *giftConfirmGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *giftConfirmGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *giftConfirmGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *giftConfirmGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}
