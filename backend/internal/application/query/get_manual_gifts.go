package query

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// ManualGiftReadModel is a privacy-safe application read model. It contains
// no Telegram user ID for the selected recipient.
type ManualGiftReadModel struct {
	ID                 uint
	EventID            uint
	Description        string
	GenderFilter       string
	BikeTypeFilter     string
	ReviewStatus       string
	ManualDistribution bool
	Place              *int
	PlaceRule          valueobject.GiftPlaceRule
	Attachments        []entity.GiftAttachment
	Criteria           []*entity.Criteria
	Recipient          *ManualGiftRecipientReadModel
	CreatedAt          time.Time
}

// ManualGiftRecipientReadModel is the minimum identity needed for recipient selection.
type ManualGiftRecipientReadModel struct {
	ID          uint
	DisplayName string
	Username    string
	Status      string
}

// GetManualGiftsQuery requests the protected dashboard enrichment view.
type GetManualGiftsQuery struct {
	EventID uint
}

// GetManualGiftsHandler returns only manual-mode gifts for one event.
type GetManualGiftsHandler struct {
	giftRepo repository.GiftRepository
}

func NewGetManualGiftsHandler(giftRepo repository.GiftRepository) *GetManualGiftsHandler {
	return &GetManualGiftsHandler{giftRepo: giftRepo}
}

func (h *GetManualGiftsHandler) Handle(ctx context.Context, query GetManualGiftsQuery) ([]*ManualGiftReadModel, error) {
	log.Printf("DEBUG manual gifts query started: scope=admin event_id=%d", query.EventID)
	gifts, err := h.giftRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		log.Printf("ERROR manual gifts query failed: scope=admin event_id=%d stage=find_by_event error=%v", query.EventID, err)
		return nil, fmt.Errorf("find manual gifts for event %d: %w", query.EventID, err)
	}

	manualGifts := make([]*ManualGiftReadModel, 0)
	for _, gift := range gifts {
		if !gift.ManualDistribution {
			continue
		}
		manualGift, err := newManualGiftReadModel(ctx, h.giftRepo, nil, gift)
		if err != nil {
			return nil, fmt.Errorf("load manual gift %d: %w", gift.ID, err)
		}
		manualGifts = append(manualGifts, manualGift)
	}
	log.Printf("DEBUG manual gifts query completed: scope=admin event_id=%d returned_count=%d", query.EventID, len(manualGifts))
	return manualGifts, nil
}

// GetOwnerManualGiftsQuery requests all gifts added by the verified owner for
// the active event. Pending and approved review states are intentionally kept.
type GetOwnerManualGiftsQuery struct {
	OwnerTelegramUserID int64
	EventID             uint
}

// GetOwnerManualGiftsHandler returns the protected owner view used by My Prizes.
type GetOwnerManualGiftsHandler struct {
	giftRepo     repository.ManualGiftRepository
	criteriaRepo repository.CriteriaRepository
}

func NewGetOwnerManualGiftsHandler(
	giftRepo repository.ManualGiftRepository,
	criteriaRepo repository.CriteriaRepository,
) *GetOwnerManualGiftsHandler {
	return &GetOwnerManualGiftsHandler{
		giftRepo:     giftRepo,
		criteriaRepo: criteriaRepo,
	}
}

func (h *GetOwnerManualGiftsHandler) Handle(ctx context.Context, query GetOwnerManualGiftsQuery) ([]*ManualGiftReadModel, error) {
	log.Printf("DEBUG manual gifts query started: scope=owner owner_user_id=%d event_id=%d", query.OwnerTelegramUserID, query.EventID)
	gifts, err := h.giftRepo.FindByUserAndEvent(ctx, query.OwnerTelegramUserID, query.EventID)
	if err != nil {
		log.Printf("ERROR manual gifts query failed: scope=owner owner_user_id=%d event_id=%d stage=find_by_user_and_event error=%v", query.OwnerTelegramUserID, query.EventID, err)
		return nil, fmt.Errorf("find owner gifts for user %d event %d: %w", query.OwnerTelegramUserID, query.EventID, err)
	}

	ownerGifts := make([]*ManualGiftReadModel, 0, len(gifts))
	for _, gift := range gifts {
		ownerGift, err := newManualGiftReadModel(ctx, h.giftRepo, h.criteriaRepo, gift)
		if err != nil {
			return nil, fmt.Errorf("load owner gift %d: %w", gift.ID, err)
		}
		ownerGifts = append(ownerGifts, ownerGift)
	}
	log.Printf("DEBUG manual gifts query completed: scope=owner owner_user_id=%d event_id=%d returned_count=%d", query.OwnerTelegramUserID, query.EventID, len(ownerGifts))
	return ownerGifts, nil
}

func newManualGiftReadModel(
	ctx context.Context,
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
	gift *entity.Gift,
) (*ManualGiftReadModel, error) {
	attachments, err := giftRepo.GetAttachments(ctx, gift.ID)
	if err != nil {
		return nil, fmt.Errorf("get attachments: %w", err)
	}

	model := &ManualGiftReadModel{
		ID:                 gift.ID,
		EventID:            gift.EventID,
		Description:        gift.Description,
		GenderFilter:       gift.GenderFilter,
		BikeTypeFilter:     gift.BikeTypeFilter,
		ReviewStatus:       gift.ReviewStatus.String(),
		ManualDistribution: gift.ManualDistribution,
		Place:              gift.FirstLegacyPlace(),
		PlaceRule:          gift.PlaceRule,
		CreatedAt:          gift.CreatedAt,
	}
	if len(attachments) > 0 {
		model.Attachments = make([]entity.GiftAttachment, len(attachments))
		for index, attachment := range attachments {
			model.Attachments[index] = *attachment
		}
	}
	if criteriaRepo != nil {
		criteria, err := criteriaRepo.FindByGift(ctx, gift.ID)
		if err != nil {
			return nil, fmt.Errorf("get criteria: %w", err)
		}
		model.Criteria = criteria
	}
	if gift.ManualRecipient != nil {
		model.Recipient = newManualGiftRecipientReadModel(gift.ManualRecipient)
	}
	return model, nil
}

func newManualGiftRecipientReadModel(participant *entity.Participant) *ManualGiftRecipientReadModel {
	status := participant.Status
	if status == "" {
		status = valueobject.ParticipantStatusActive
	}
	return &ManualGiftRecipientReadModel{
		ID:          participant.ID,
		DisplayName: manualGiftRecipientDisplayName(participant),
		Username:    manualGiftRecipientUsername(participant),
		Status:      string(status),
	}
}

func manualGiftRecipientDisplayName(participant *entity.Participant) string {
	if participant.User != nil {
		name := strings.TrimSpace(strings.Join([]string{
			strings.TrimSpace(participant.User.FirstName),
			strings.TrimSpace(participant.User.LastName),
		}, " "))
		if name != "" {
			return name
		}
		if username := strings.TrimSpace(participant.User.Username); username != "" {
			return "@" + username
		}
	}
	return fmt.Sprintf("Участник #%d", participant.ID)
}

func manualGiftRecipientUsername(participant *entity.Participant) string {
	if participant.User == nil {
		return ""
	}
	return strings.TrimSpace(participant.User.Username)
}
