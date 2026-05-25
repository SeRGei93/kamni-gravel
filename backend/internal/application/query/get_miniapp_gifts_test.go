package query

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

func TestGetMiniappGiftsHandlerReusesApprovedGiftsQueryAndFiltersCatalog(t *testing.T) {
	giftRepo := &miniappGiftRepoFake{
		gifts: []*entity.Gift{
			{
				ID:             1,
				UserID:         123,
				EventID:        77,
				Description:    "Bottle cage",
				GenderFilter:   "male",
				BikeTypeFilter: "gravel",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 123, FirstName: "Alex"},
			},
			{
				ID:             2,
				UserID:         124,
				EventID:        77,
				Description:    "Women prize",
				GenderFilter:   "female",
				BikeTypeFilter: "gravel",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 124, FirstName: "Kate"},
			},
			{
				ID:             3,
				UserID:         125,
				EventID:        77,
				Description:    "Generic prize",
				GenderFilter:   "",
				BikeTypeFilter: "",
				ReviewStatus:   entity.GiftReviewStatusApproved,
				User:           &entity.User{ID: 125, FirstName: "Sam"},
			},
		},
		attachments: map[uint][]*entity.GiftAttachment{
			1: {{ID: 10, GiftID: 1, TelegramFileID: "file-1", FileType: "photo"}},
		},
	}
	criteriaRepo := &miniappCriteriaRepoFake{
		criteriaByGift: map[uint][]*entity.Criteria{
			1: {{ID: 5, Name: "Speed"}},
		},
	}
	handler := NewGetMiniappGiftsHandler(giftRepo, criteriaRepo)

	gifts, err := handler.Handle(context.Background(), GetMiniappGiftsQuery{
		EventID:  77,
		Gender:   " MALE ",
		BikeType: "GRAVEL",
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if !giftRepo.findByStatusCalled {
		t.Fatal("FindByEventAndReviewStatus was not called")
	}
	if giftRepo.eventID != 77 {
		t.Fatalf("event ID mismatch: got %d, want 77", giftRepo.eventID)
	}
	if giftRepo.reviewStatus != entity.GiftReviewStatusApproved {
		t.Fatalf("review status mismatch: got %s, want %s", giftRepo.reviewStatus, entity.GiftReviewStatusApproved)
	}
	if len(gifts) != 2 {
		t.Fatalf("gift count mismatch: got %d, want 2", len(gifts))
	}
	if len(gifts[0].Criteria) != 1 || gifts[0].Criteria[0].ID != 5 {
		t.Fatalf("criteria mismatch: %#v", gifts[0].Criteria)
	}
	if len(gifts[0].Attachments) != 1 || gifts[0].Attachments[0].ID != 10 {
		t.Fatalf("attachments mismatch: %#v", gifts[0].Attachments)
	}
	if gifts[1].ID != 3 {
		t.Fatalf("generic gift should match selected filters, got gift ID %d", gifts[1].ID)
	}
}

func TestGetMiniappGiftsHandlerDefaultsFiltersToAll(t *testing.T) {
	giftRepo := &miniappGiftRepoFake{}
	handler := NewGetMiniappGiftsHandler(giftRepo, &miniappCriteriaRepoFake{})

	if _, err := handler.Handle(context.Background(), GetMiniappGiftsQuery{EventID: 77}); err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if !giftRepo.findByStatusCalled {
		t.Fatal("FindByEventAndReviewStatus was not called")
	}
}

func TestGetMiniappGiftsHandlerRejectsInvalidFilters(t *testing.T) {
	tests := []struct {
		name    string
		query   GetMiniappGiftsQuery
		wantErr error
	}{
		{
			name:    "invalid gender",
			query:   GetMiniappGiftsQuery{EventID: 77, Gender: "everyone", BikeType: "gravel"},
			wantErr: ErrInvalidMiniappGiftGenderFilter,
		},
		{
			name:    "invalid bike type",
			query:   GetMiniappGiftsQuery{EventID: 77, Gender: "male", BikeType: "cx"},
			wantErr: ErrInvalidMiniappGiftBikeTypeFilter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			giftRepo := &miniappGiftRepoFake{}
			handler := NewGetMiniappGiftsHandler(giftRepo, &miniappCriteriaRepoFake{})

			_, err := handler.Handle(context.Background(), tt.query)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error mismatch: got %v, want %v", err, tt.wantErr)
			}
			if giftRepo.findByStatusCalled {
				t.Fatal("repository should not be called for invalid filters")
			}
		})
	}
}

type miniappGiftRepoFake struct {
	findByStatusCalled bool
	eventID            uint
	reviewStatus       entity.GiftReviewStatus
	gifts              []*entity.Gift
	attachments        map[uint][]*entity.GiftAttachment
}

func (r *miniappGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *miniappGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	return nil
}
func (r *miniappGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *miniappGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	return nil
}
func (r *miniappGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *miniappGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *miniappGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	r.findByStatusCalled = true
	r.eventID = eventID
	r.reviewStatus = reviewStatus
	return r.gifts, nil
}
func (r *miniappGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *miniappGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *miniappGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *miniappGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	if r.attachments == nil {
		return nil, nil
	}
	return r.attachments[giftID], nil
}

type miniappCriteriaRepoFake struct {
	criteriaByGift map[uint][]*entity.Criteria
}

func (r *miniappCriteriaRepoFake) Create(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *miniappCriteriaRepoFake) Update(ctx context.Context, criteria *entity.Criteria) error {
	return nil
}
func (r *miniappCriteriaRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *miniappCriteriaRepoFake) FindByID(ctx context.Context, id uint) (*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappCriteriaRepoFake) FindAll(ctx context.Context) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappCriteriaRepoFake) FindByType(ctx context.Context, criteriaType valueobject.CriteriaType) ([]*entity.Criteria, error) {
	return nil, nil
}
func (r *miniappCriteriaRepoFake) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	if r.criteriaByGift == nil {
		return nil, nil
	}
	return r.criteriaByGift[giftID], nil
}
func (r *miniappCriteriaRepoFake) FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error) {
	return nil, nil
}
