package command

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

func TestUpdateGiftHandlerRejectsInvalidReviewStatusAndFilters(t *testing.T) {
	h := NewUpdateGiftHandler(&updateGiftRepoFake{gift: baseUpdateGift()})

	badStatus := "published"
	if _, err := h.Handle(context.Background(), UpdateGiftCommand{GiftID: 1, ReviewStatus: &badStatus}); !errors.Is(err, ErrInvalidGiftReviewStatus) {
		t.Fatalf("invalid review status error mismatch: got %v", err)
	}

	badGender := "everyone"
	if _, err := h.Handle(context.Background(), UpdateGiftCommand{GiftID: 1, GenderFilter: &badGender}); !errors.Is(err, ErrInvalidGiftGenderFilter) {
		t.Fatalf("invalid gender filter error mismatch: got %v", err)
	}

	badBike := "bmx"
	if _, err := h.Handle(context.Background(), UpdateGiftCommand{GiftID: 1, BikeTypeFilter: &badBike}); !errors.Is(err, ErrInvalidGiftBikeTypeFilter) {
		t.Fatalf("invalid bike filter error mismatch: got %v", err)
	}
}

func TestUpdateGiftHandlerPlacePresenceSemantics(t *testing.T) {
	place := 3
	placeRule, err := valueobject.NewGiftPlaceRulePlaces([]int{place})
	if err != nil {
		t.Fatalf("place rule setup error: %v", err)
	}
	repo := &updateGiftRepoFake{gift: baseUpdateGift()}
	repo.gift.Place = &place
	repo.gift.PlaceRule = placeRule
	h := NewUpdateGiftHandler(repo)

	description := "Updated"
	if _, err := h.Handle(context.Background(), UpdateGiftCommand{GiftID: 1, Description: &description}); err != nil {
		t.Fatalf("preserve place update error: %v", err)
	}
	if repo.updatedGift.Place == nil || *repo.updatedGift.Place != 3 {
		t.Fatalf("place should be preserved when omitted, got %v", repo.updatedGift.Place)
	}

	if _, err := h.Handle(context.Background(), UpdateGiftCommand{GiftID: 1, PlaceSet: true, Place: nil}); err != nil {
		t.Fatalf("clear place update error: %v", err)
	}
	if repo.updatedGift.Place != nil {
		t.Fatalf("place should be cleared by explicit null, got %v", *repo.updatedGift.Place)
	}
	if !repo.updatedGift.PlaceRule.IsNone() {
		t.Fatalf("place rule should be cleared by legacy null, got %s", repo.updatedGift.PlaceRule.Type())
	}
}

func TestUpdateGiftHandlerPlaceRulePresenceSemantics(t *testing.T) {
	existingRule, err := valueobject.NewGiftPlaceRulePlaces([]int{3})
	if err != nil {
		t.Fatalf("existing rule setup error: %v", err)
	}
	repo := &updateGiftRepoFake{gift: baseUpdateGift()}
	repo.gift.PlaceRule = existingRule
	h := NewUpdateGiftHandler(repo)

	description := "Updated"
	if _, err := h.Handle(context.Background(), UpdateGiftCommand{GiftID: 1, Description: &description}); err != nil {
		t.Fatalf("preserve rule update error: %v", err)
	}
	assertUpdateGiftRulePlaces(t, repo.updatedGift.PlaceRule, []int{3})

	if _, err := h.Handle(context.Background(), UpdateGiftCommand{
		GiftID:       1,
		PlaceRuleSet: true,
		PlaceRule:    valueobject.NewGiftPlaceRuleNone(),
	}); err != nil {
		t.Fatalf("clear rule update error: %v", err)
	}
	if !repo.updatedGift.PlaceRule.IsNone() {
		t.Fatalf("place rule should be cleared by explicit null, got %s", repo.updatedGift.PlaceRule.Type())
	}
	if repo.updatedGift.Place != nil {
		t.Fatalf("legacy place should be cleared with rule, got %v", *repo.updatedGift.Place)
	}
}

func TestUpdateGiftHandlerAcceptsStructuredPlaceRules(t *testing.T) {
	placesRule, err := valueobject.NewGiftPlaceRulePlaces([]int{3, 1, 3})
	if err != nil {
		t.Fatalf("places rule setup error: %v", err)
	}
	repo := &updateGiftRepoFake{gift: baseUpdateGift()}
	h := NewUpdateGiftHandler(repo)

	if _, err := h.Handle(context.Background(), UpdateGiftCommand{
		GiftID:       1,
		PlaceRuleSet: true,
		PlaceRule:    placesRule,
	}); err != nil {
		t.Fatalf("places rule update error: %v", err)
	}
	assertUpdateGiftRulePlaces(t, repo.updatedGift.PlaceRule, []int{1, 3})
	if repo.updatedGift.Place == nil || *repo.updatedGift.Place != 1 {
		t.Fatalf("legacy place = %v, want 1", repo.updatedGift.Place)
	}

	lastRule, err := valueobject.NewGiftPlaceRuleLastN(5)
	if err != nil {
		t.Fatalf("last_n rule setup error: %v", err)
	}
	if _, err := h.Handle(context.Background(), UpdateGiftCommand{
		GiftID:       1,
		PlaceRuleSet: true,
		PlaceRule:    lastRule,
	}); err != nil {
		t.Fatalf("last_n rule update error: %v", err)
	}
	if repo.updatedGift.PlaceRule.Type() != valueobject.GiftPlaceRuleTypeLastN || repo.updatedGift.PlaceRule.LastCount() != 5 {
		t.Fatalf("place rule = %s/%d, want last_n/5", repo.updatedGift.PlaceRule.Type(), repo.updatedGift.PlaceRule.LastCount())
	}
	if repo.updatedGift.Place != nil {
		t.Fatalf("legacy place should be nil for last_n, got %v", *repo.updatedGift.Place)
	}
}

func TestUpdateGiftHandlerPlaceRuleWinsOverLegacyPlace(t *testing.T) {
	legacyPlace := 2
	placesRule, err := valueobject.NewGiftPlaceRulePlaces([]int{10, 11})
	if err != nil {
		t.Fatalf("places rule setup error: %v", err)
	}
	repo := &updateGiftRepoFake{gift: baseUpdateGift()}
	h := NewUpdateGiftHandler(repo)

	if _, err := h.Handle(context.Background(), UpdateGiftCommand{
		GiftID:       1,
		Place:        &legacyPlace,
		PlaceSet:     true,
		PlaceRule:    placesRule,
		PlaceRuleSet: true,
	}); err != nil {
		t.Fatalf("place_rule winning update error: %v", err)
	}

	assertUpdateGiftRulePlaces(t, repo.updatedGift.PlaceRule, []int{10, 11})
	if repo.updatedGift.Place == nil || *repo.updatedGift.Place != 10 {
		t.Fatalf("legacy place should mirror place_rule first place, got %v", repo.updatedGift.Place)
	}
}

func TestUpdateGiftHandlerApprovesWithCriteriaAtomically(t *testing.T) {
	repo := &updateGiftRepoFake{gift: baseUpdateGift()}
	h := NewUpdateGiftHandler(repo)
	status := entity.GiftReviewStatusApproved.String()

	if _, err := h.Handle(context.Background(), UpdateGiftCommand{GiftID: 1, ReviewStatus: &status}); !errors.Is(err, ErrGiftCriteriaPayloadRequired) {
		t.Fatalf("missing criteria payload error mismatch: got %v", err)
	}

	criteriaIDs := []uint{10, 20}
	if _, err := h.Handle(context.Background(), UpdateGiftCommand{
		GiftID:         1,
		ReviewStatus:   &status,
		CriteriaIDs:    criteriaIDs,
		CriteriaIDsSet: true,
	}); err != nil {
		t.Fatalf("approve update error: %v", err)
	}
	if !repo.updateWithCriteriaCalled {
		t.Fatal("UpdateWithCriteria was not called")
	}
	if repo.updatedGift.ReviewStatus != entity.GiftReviewStatusApproved {
		t.Fatalf("review status mismatch: got %s", repo.updatedGift.ReviewStatus)
	}
	if len(repo.criteriaIDs) != len(criteriaIDs) {
		t.Fatalf("criteria count mismatch: got %d, want %d", len(repo.criteriaIDs), len(criteriaIDs))
	}
}

func baseUpdateGift() *entity.Gift {
	return &entity.Gift{
		ID:             1,
		UserID:         123,
		EventID:        77,
		Description:    "Gift",
		GenderFilter:   "all",
		BikeTypeFilter: "all",
		ReviewStatus:   entity.GiftReviewStatusPendingReview,
	}
}

type updateGiftRepoFake struct {
	gift                     *entity.Gift
	updatedGift              *entity.Gift
	criteriaIDs              []uint
	updateWithCriteriaCalled bool
}

func (r *updateGiftRepoFake) Create(ctx context.Context, gift *entity.Gift) error { return nil }
func (r *updateGiftRepoFake) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	return nil
}
func (r *updateGiftRepoFake) Update(ctx context.Context, gift *entity.Gift) error {
	r.updatedGift = gift
	return nil
}
func (r *updateGiftRepoFake) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	r.updateWithCriteriaCalled = true
	r.updatedGift = gift
	r.criteriaIDs = append([]uint(nil), criteriaIDs...)
	return nil
}
func (r *updateGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	gift := *r.gift
	return &gift, nil
}
func (r *updateGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *updateGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *updateGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *updateGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *updateGiftRepoFake) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	return nil
}
func (r *updateGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}

func assertUpdateGiftRulePlaces(t *testing.T, rule valueobject.GiftPlaceRule, want []int) {
	t.Helper()

	got := rule.Places()
	if len(got) != len(want) {
		t.Fatalf("place rule places = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("place rule places = %v, want %v", got, want)
		}
	}
}
