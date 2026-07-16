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
	},
		attachments: map[uint][]*entity.GiftAttachment{
			2: {{ID: 7, GiftID: 2, TelegramFileID: "photo-7", FileType: "photo"}},
		},
	}
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
	if len(gifts[0].Attachments) != 1 || gifts[0].Attachments[0].TelegramFileID != "photo-7" {
		t.Fatalf("attachments = %+v", gifts[0].Attachments)
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
	place := 3
	repo := &manualGiftsRepoFake{ownerGifts: []*entity.Gift{
		{ID: 1, EventID: 77, ReviewStatus: entity.GiftReviewStatusPendingReview, ManualDistribution: true},
		{ID: 2, EventID: 77, GenderFilter: "female", BikeTypeFilter: "gravel", Place: &place, ReviewStatus: entity.GiftReviewStatusApproved},
	}}
	criteriaRepo := &manualGiftCriteriaRepoFake{criteriaByGift: map[uint][]*entity.Criteria{
		2: {{ID: 4, Name: "Самый быстрый", CriteriaType: "speed"}},
	}}
	handler := NewGetOwnerManualGiftsHandler(repo, criteriaRepo, nil, nil)

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
	if gifts[1].GenderFilter != "female" || gifts[1].BikeTypeFilter != "gravel" || gifts[1].Place == nil || *gifts[1].Place != place || len(gifts[1].Criteria) != 1 || gifts[1].Criteria[0].ID != 4 {
		t.Fatalf("automatic gift conditions = %+v", gifts[1])
	}
}

func TestGetOwnerManualGiftsHandlerIncludesAutomaticRecipientsWhenRequested(t *testing.T) {
	automaticGift := &entity.Gift{ID: 2, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved}
	repo := &manualGiftsRepoFake{ownerGifts: []*entity.Gift{automaticGift}}
	distributionReader := &manualGiftPrizeDistributionReaderFake{results: []*PrizeDistributionResult{
		{
			ParticipantID:   10,
			ParticipantName: "Ivan",
			Status:          "active",
			MatchedGiftAssignments: []*PrizeGiftAssignment{
				{ParticipantID: 10, Gift: automaticGift},
			},
			MatchedGifts: []*entity.Gift{automaticGift},
		},
		{
			ParticipantID:   11,
			ParticipantName: "Maria",
			Status:          "dnf",
			MatchedGiftAssignments: []*PrizeGiftAssignment{
				{ParticipantID: 11, Gift: automaticGift},
			},
		},
	}}
	handler := NewGetOwnerManualGiftsHandler(repo, &manualGiftCriteriaRepoFake{}, nil, distributionReader)

	gifts, err := handler.Handle(context.Background(), GetOwnerManualGiftsQuery{
		OwnerTelegramUserID:        100,
		EventID:                    77,
		IncludeAutomaticRecipients: true,
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if distributionReader.calls != 1 || distributionReader.eventID != 77 {
		t.Fatalf("prize distribution calls = %d, event_id = %d", distributionReader.calls, distributionReader.eventID)
	}
	if len(gifts) != 1 || len(gifts[0].Recipients) != 2 {
		t.Fatalf("automatic recipients = %+v", gifts)
	}
	if gifts[0].Recipients[0].DisplayName != "Ivan" || gifts[0].Recipients[1].DisplayName != "Maria" {
		t.Fatalf("automatic recipients = %+v", gifts[0].Recipients)
	}
}

func TestGetOwnerManualGiftsHandlerSkipsAutomaticRecipientsWhenNotRequested(t *testing.T) {
	repo := &manualGiftsRepoFake{ownerGifts: []*entity.Gift{{ID: 2, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved}}}
	distributionReader := &manualGiftPrizeDistributionReaderFake{err: errors.New("must not be called")}
	handler := NewGetOwnerManualGiftsHandler(repo, &manualGiftCriteriaRepoFake{}, nil, distributionReader)

	gifts, err := handler.Handle(context.Background(), GetOwnerManualGiftsQuery{
		OwnerTelegramUserID: 100,
		EventID:             77,
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if distributionReader.calls != 0 {
		t.Fatalf("prize distribution calls = %d, want 0", distributionReader.calls)
	}
	if len(gifts) != 1 || len(gifts[0].Recipients) != 0 {
		t.Fatalf("automatic recipients must be omitted: %+v", gifts)
	}
}

func TestGetManualGiftsHandlersPropagateRepositoryFailures(t *testing.T) {
	repoErr := errors.New("database unavailable")
	adminHandler := NewGetManualGiftsHandler(&manualGiftsRepoFake{eventErr: repoErr})
	if _, err := adminHandler.Handle(context.Background(), GetManualGiftsQuery{EventID: 77}); !errors.Is(err, repoErr) {
		t.Fatalf("admin query error = %v, want wrapped repository error", err)
	}

	ownerRepo := &manualGiftsRepoFake{ownerErr: repoErr}
	ownerHandler := NewGetOwnerManualGiftsHandler(ownerRepo, &manualGiftCriteriaRepoFake{}, nil, nil)
	if _, err := ownerHandler.Handle(context.Background(), GetOwnerManualGiftsQuery{OwnerTelegramUserID: 100, EventID: 77}); !errors.Is(err, repoErr) {
		t.Fatalf("owner query error = %v, want wrapped repository error", err)
	}

	hasOwnerHandler := NewHasOwnerGiftsHandler(&manualGiftsRepoFake{hasOwnerErr: repoErr})
	if _, err := hasOwnerHandler.Handle(context.Background(), HasOwnerGiftsQuery{OwnerTelegramUserID: 100, EventID: 77}); !errors.Is(err, repoErr) {
		t.Fatalf("owner gift presence error = %v, want wrapped repository error", err)
	}
}

func TestGetOwnerManualGiftsHandlerReusesDistributionForRecipientsAndParticipantOptions(t *testing.T) {
	automaticGift := &entity.Gift{ID: 2, EventID: 77, ReviewStatus: entity.GiftReviewStatusApproved}
	repo := &manualGiftsRepoFake{
		ownerGifts:            []*entity.Gift{automaticGift},
		manualRecipientCounts: map[uint]int{11: 1},
	}
	participantRepo := &manualGiftParticipantRepoFake{participants: []*entity.Participant{
		{ID: 10, EventID: 77, Result: &entity.Result{}, User: &entity.User{FirstName: "Ivan"}},
		{ID: 11, EventID: 77, Result: &entity.Result{}, User: &entity.User{FirstName: "Maria"}},
		{ID: 12, EventID: 77, Result: &entity.Result{}, User: &entity.User{FirstName: "Alex"}},
	}}
	distributionReader := &manualGiftPrizeDistributionReaderFake{results: []*PrizeDistributionResult{{
		ParticipantID:   10,
		ParticipantName: "Ivan",
		MatchedGiftAssignments: []*PrizeGiftAssignment{{
			ParticipantID: 10,
			Gift:          automaticGift,
		}},
	}}}
	handler := NewGetOwnerManualGiftsHandler(repo, &manualGiftCriteriaRepoFake{}, participantRepo, distributionReader)

	output, err := handler.HandleDetailed(context.Background(), GetOwnerManualGiftsQuery{
		OwnerTelegramUserID:        100,
		EventID:                    77,
		IncludeAutomaticRecipients: true,
		IncludeParticipantOptions:  true,
	})
	if err != nil {
		t.Fatalf("HandleDetailed error: %v", err)
	}
	if distributionReader.calls != 1 || distributionReader.eventID != 77 {
		t.Fatalf("prize distribution calls = %d, event_id = %d", distributionReader.calls, distributionReader.eventID)
	}
	if len(output.Gifts) != 1 || len(output.Gifts[0].Recipients) != 1 || output.Gifts[0].Recipients[0].ID != 10 {
		t.Fatalf("automatic recipients = %+v", output.Gifts)
	}
	if len(output.ParticipantOptions) != 3 || output.ParticipantOptions[0].ID != 12 || output.ParticipantOptions[0].HasPrize {
		t.Fatalf("participant options = %+v", output.ParticipantOptions)
	}
	if !output.ParticipantOptions[1].HasPrize || !output.ParticipantOptions[2].HasPrize {
		t.Fatalf("awarded participant options = %+v", output.ParticipantOptions)
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
	eventID               uint
	eventGifts            []*entity.Gift
	eventErr              error
	ownerID               int64
	ownerEventID          uint
	ownerGifts            []*entity.Gift
	ownerErr              error
	attachments           map[uint][]*entity.GiftAttachment
	hasOwnerGifts         bool
	hasOwnerErr           error
	manualRecipientCounts map[uint]int
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

func (r *manualGiftsRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return r.attachments[giftID], nil
}

func (r *manualGiftsRepoFake) ManualRecipientCountsByEvent(ctx context.Context, eventID uint) (map[uint]int, error) {
	return r.manualRecipientCounts, nil
}

type manualGiftParticipantRepoFake struct {
	repository.ParticipantRepository
	participants []*entity.Participant
}

func (r *manualGiftParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return r.participants, nil
}

type manualGiftCriteriaRepoFake struct {
	repository.CriteriaRepository
	criteriaByGift map[uint][]*entity.Criteria
}

type manualGiftPrizeDistributionReaderFake struct {
	results []*PrizeDistributionResult
	err     error
	calls   int
	eventID uint
}

func (r *manualGiftPrizeDistributionReaderFake) Handle(_ context.Context, query GetPrizeDistributionQuery) ([]*PrizeDistributionResult, error) {
	r.calls++
	r.eventID = query.EventID
	return r.results, r.err
}

func (r *manualGiftCriteriaRepoFake) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	return r.criteriaByGift[giftID], nil
}
