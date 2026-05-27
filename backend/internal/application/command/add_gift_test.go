package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
)

func TestAddGiftHandlerCreatesPendingReviewGiftWithAttachmentsAtomically(t *testing.T) {
	giftRepo := &addGiftRepoFake{}
	h := NewAddGiftHandler(
		&addGiftUserRepoFake{user: &entity.User{ID: 123}},
		&addGiftEventRepoFake{event: &entity.Event{ID: 77}},
		giftRepo,
		&addGiftUserBlacklistRepoFake{},
	)

	gift, err := h.Handle(context.Background(), AddGiftCommand{
		UserID:         123,
		EventID:        77,
		Description:    "Bottle cage",
		GenderFilter:   "all",
		BikeTypeFilter: "gravel",
		Attachments: []GiftAttachmentData{
			{TelegramFileID: "file-1", FileType: "photo"},
		},
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if !giftRepo.createWithAttachmentsCalled {
		t.Fatal("CreateWithAttachments was not called")
	}
	if giftRepo.createCalled {
		t.Fatal("Create should not be used for gift creation")
	}
	if giftRepo.addAttachmentCalled {
		t.Fatal("AddAttachment should not be used for gift creation")
	}
	if gift.ReviewStatus != entity.GiftReviewStatusPendingReview {
		t.Fatalf("review status mismatch: got %s, want %s", gift.ReviewStatus, entity.GiftReviewStatusPendingReview)
	}
	if len(gift.Attachments) != 1 {
		t.Fatalf("attachment count mismatch: got %d, want 1", len(gift.Attachments))
	}
	if giftRepo.createdGift.UserID != 123 {
		t.Fatalf("gift user mismatch: got %d, want 123", giftRepo.createdGift.UserID)
	}
}

func TestAddGiftHandlerRejectsBlacklistedUser(t *testing.T) {
	giftRepo := &addGiftRepoFake{}
	h := NewAddGiftHandler(
		&addGiftUserRepoFake{user: &entity.User{ID: 123}},
		&addGiftEventRepoFake{event: &entity.Event{ID: 77}},
		giftRepo,
		&addGiftUserBlacklistRepoFake{blacklisted: true},
	)

	_, err := h.Handle(context.Background(), AddGiftCommand{
		UserID:      123,
		EventID:     77,
		Description: "Bottle cage",
	})
	if !errors.Is(err, ErrUserBlacklisted) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrUserBlacklisted)
	}
	if giftRepo.createWithAttachmentsCalled || giftRepo.createCalled {
		t.Fatal("gift should not be created for blacklisted user")
	}
}

type addGiftUserRepoFake struct {
	user *entity.User
}

func (r *addGiftUserRepoFake) Create(ctx context.Context, user *entity.User) error { return nil }
func (r *addGiftUserRepoFake) Update(ctx context.Context, user *entity.User) error { return nil }
func (r *addGiftUserRepoFake) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	return r.user, nil
}
func (r *addGiftUserRepoFake) Delete(ctx context.Context, id int64) error { return nil }
func (r *addGiftUserRepoFake) GetAll(ctx context.Context) ([]*entity.User, error) {
	return nil, nil
}

type addGiftUserBlacklistRepoFake struct {
	blacklisted bool
}

func (r *addGiftUserBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *addGiftUserBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *addGiftUserBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	return r.blacklisted, nil
}
func (r *addGiftUserBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	return nil
}
func (r *addGiftUserBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *addGiftUserBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return nil
}

type addGiftEventRepoFake struct {
	event *entity.Event
}

func (r *addGiftEventRepoFake) Create(ctx context.Context, event *entity.Event) error { return nil }
func (r *addGiftEventRepoFake) Update(ctx context.Context, event *entity.Event) error { return nil }
func (r *addGiftEventRepoFake) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	return r.event, nil
}
func (r *addGiftEventRepoFake) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	return nil, nil
}
func (r *addGiftEventRepoFake) FindActive(ctx context.Context) (*entity.Event, error) {
	return r.event, nil
}
func (r *addGiftEventRepoFake) GetAll(ctx context.Context) ([]*entity.Event, error) {
	return nil, nil
}
func (r *addGiftEventRepoFake) Delete(ctx context.Context, id uint) error { return nil }

type addGiftRepoFake struct {
	createCalled                bool
	createWithAttachmentsCalled bool
	addAttachmentCalled         bool
	createdGift                 *entity.Gift
}

func (r *addGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error {
	r.createCalled = true
	return nil
}
func (r *addGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	r.createWithAttachmentsCalled = true
	r.createdGift = gift
	gift.ID = 99
	for i, attachment := range attachments {
		attachment.ID = uint(i + 1)
		attachment.GiftID = gift.ID
	}
	return nil
}
func (r *addGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *addGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *addGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *addGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *addGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *addGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *addGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *addGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	r.addAttachmentCalled = true
	return nil
}
func (r *addGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}
