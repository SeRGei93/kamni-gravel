package query

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/entity"
)

func TestGetGiftsHandlerPaginates(t *testing.T) {
	approved := entity.GiftReviewStatusApproved
	giftRepo := &miniappGiftRepoFake{
		gifts: []*entity.Gift{
			{ID: 1, ReviewStatus: approved},
			{ID: 2, ReviewStatus: approved},
		},
	}
	criteriaRepo := &miniappCriteriaRepoFake{}
	h := NewGetGiftsHandler(giftRepo, criteriaRepo)

	gifts, total, err := h.Handle(context.Background(), GetGiftsQuery{
		EventID:      77,
		ReviewStatus: &approved,
		Limit:        50,
		Offset:       100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if giftRepo.gotLimit != 50 || giftRepo.gotOffset != 100 {
		t.Fatalf("limit/offset not passed through: got limit=%d offset=%d", giftRepo.gotLimit, giftRepo.gotOffset)
	}
	if total != 2 {
		t.Fatalf("total mismatch: got %d, want 2", total)
	}
	if len(gifts) != 2 {
		t.Fatalf("gifts mismatch: got %d, want 2", len(gifts))
	}
}

func TestGetGiftsHandlerRejectsInvalidStatus(t *testing.T) {
	giftRepo := &miniappGiftRepoFake{}
	criteriaRepo := &miniappCriteriaRepoFake{}
	h := NewGetGiftsHandler(giftRepo, criteriaRepo)

	bad := entity.GiftReviewStatus("nonsense")
	_, _, err := h.Handle(context.Background(), GetGiftsQuery{EventID: 77, ReviewStatus: &bad, Limit: 50})
	if err == nil {
		t.Fatal("expected error for invalid review status")
	}
}

func TestGetGiftsHandlerFiltersByOwnerAndSearch(t *testing.T) {
	ownerUserID := int64(101)
	otherUserID := int64(202)
	giftRepo := &miniappGiftRepoFake{
		gifts: []*entity.Gift{
			{
				ID:          1,
				UserID:      ownerUserID,
				Description: "Шлем",
				User:        &entity.User{Username: "rider", FirstName: "Иван"},
			},
			{
				ID:          2,
				UserID:      otherUserID,
				Description: "Фляга",
				User:        &entity.User{Username: "other", FirstName: "Пётр"},
			},
		},
	}
	h := NewGetGiftsHandler(giftRepo, &miniappCriteriaRepoFake{})

	gifts, total, err := h.Handle(context.Background(), GetGiftsQuery{
		EventID:     77,
		OwnerUserID: &ownerUserID,
		SearchQuery: "rider",
		Limit:       50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(gifts) != 1 || gifts[0].ID != 1 {
		t.Fatalf("filtered gifts = %#v, total=%d; want gift 1", gifts, total)
	}
}
