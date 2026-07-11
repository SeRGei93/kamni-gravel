package query

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestGetManualGiftsHandlerReturnsOnlyManualGiftsWithSafeRecipient(t *testing.T) {
	recipientID := uint(30)
	repo := &manualGiftsRepoFake{eventGifts: []*entity.Gift{
		{ID: 1, EventID: 77, Description: "Automatic", ReviewStatus: entity.GiftReviewStatusApproved},
		{
			ID:                           2,
			EventID:                      77,
			Description:                  "Manual",
			ReviewStatus:                 entity.GiftReviewStatusPendingReview,
			ManualDistribution:           true,
			ManualRecipientParticipantID: &recipientID,
			ManualRecipient: &entity.Participant{
				ID:     recipientID,
				UserID: 900,
				User:   &entity.User{ID: 900, FirstName: "Alex", LastName: "Rider", Username: "alex"},
			},
		},
	}}
	handler := NewGetManualGiftsHandler(repo)

	gifts, err := handler.Handle(context.Background(), GetManualGiftsQuery{EventID: 77})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if repo.eventID != 77 || len(gifts) != 1 || gifts[0].ID != 2 {
		t.Fatalf("manual gifts = %+v, event_id=%d", gifts, repo.eventID)
	}
	if gifts[0].Recipient == nil || gifts[0].Recipient.ID != recipientID || gifts[0].Recipient.DisplayName != "Alex Rider" {
		t.Fatalf("recipient = %+v", gifts[0].Recipient)
	}

	body, err := json.Marshal(gifts[0])
	if err != nil {
		t.Fatalf("marshal manual gift: %v", err)
	}
	if containsJSONKey(body, "user_id") || containsJSONKey(body, "telegram_user_id") {
		t.Fatalf("protected recipient response leaks user identity: %s", body)
	}
}

func TestGetOwnerManualGiftsHandlerReturnsPendingAndApprovedGiftsForOwnerAndEvent(t *testing.T) {
	repo := &manualGiftsRepoFake{ownerGifts: []*entity.Gift{
		{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview, ManualDistribution: true},
		{ID: 2, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved},
	}}
	handler := NewGetOwnerManualGiftsHandler(repo)

	gifts, err := handler.Handle(context.Background(), GetOwnerManualGiftsQuery{OwnerTelegramUserID: 100, EventID: 77})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if repo.ownerID != 100 || repo.ownerEventID != 77 || len(gifts) != 2 {
		t.Fatalf("owner gifts = %+v, scope=%d/%d", gifts, repo.ownerID, repo.ownerEventID)
	}
	if gifts[0].ReviewStatus != entity.GiftReviewStatusPendingReview.String() || gifts[1].ReviewStatus != entity.GiftReviewStatusApproved.String() {
		t.Fatalf("review statuses = %q, %q", gifts[0].ReviewStatus, gifts[1].ReviewStatus)
	}
}

func TestGetManualGiftsHandlersPropagateRepositoryFailures(t *testing.T) {
	repoErr := errors.New("database unavailable")
	adminHandler := NewGetManualGiftsHandler(&manualGiftsRepoFake{eventErr: repoErr})
	if _, err := adminHandler.Handle(context.Background(), GetManualGiftsQuery{EventID: 77}); !errors.Is(err, repoErr) {
		t.Fatalf("admin query error = %v, want wrapped repository error", err)
	}

	ownerHandler := NewGetOwnerManualGiftsHandler(&manualGiftsRepoFake{ownerErr: repoErr})
	if _, err := ownerHandler.Handle(context.Background(), GetOwnerManualGiftsQuery{OwnerTelegramUserID: 100, EventID: 77}); !errors.Is(err, repoErr) {
		t.Fatalf("owner query error = %v, want wrapped repository error", err)
	}

	hasOwnerHandler := NewHasOwnerGiftsHandler(&manualGiftsRepoFake{hasOwnerErr: repoErr})
	if _, err := hasOwnerHandler.Handle(context.Background(), HasOwnerGiftsQuery{OwnerTelegramUserID: 100, EventID: 77}); !errors.Is(err, repoErr) {
		t.Fatalf("owner gift presence error = %v, want wrapped repository error", err)
	}
}

func TestHasOwnerGiftsHandlerReportsGiftPresence(t *testing.T) {
	repo := &manualGiftsRepoFake{hasOwnerGifts: true}
	handler := NewHasOwnerGiftsHandler(repo)

	hasGifts, err := handler.Handle(context.Background(), HasOwnerGiftsQuery{OwnerTelegramUserID: 100, EventID: 77})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if !hasGifts || repo.ownerID != 100 || repo.ownerEventID != 77 {
		t.Fatalf("has gifts = %t, scope=%d/%d", hasGifts, repo.ownerID, repo.ownerEventID)
	}
}

func containsJSONKey(body []byte, key string) bool {
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return false
	}
	_, exists := decoded[key]
	return exists
}

type manualGiftsRepoFake struct {
	repository.ManualGiftRepository
	eventID       uint
	eventGifts    []*entity.Gift
	eventErr      error
	ownerID       int64
	ownerEventID  uint
	ownerGifts    []*entity.Gift
	ownerErr      error
	hasOwnerGifts bool
	hasOwnerErr   error
}

func (r *manualGiftsRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	r.eventID = eventID
	return r.eventGifts, r.eventErr
}

func (r *manualGiftsRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) ([]*entity.Gift, error) {
	r.ownerID = userID
	r.ownerEventID = eventID
	return r.ownerGifts, r.ownerErr
}

func (r *manualGiftsRepoFake) HasByUserAndEvent(ctx context.Context, userID int64, eventID uint) (bool, error) {
	r.ownerID = userID
	r.ownerEventID = eventID
	return r.hasOwnerGifts, r.hasOwnerErr
}
