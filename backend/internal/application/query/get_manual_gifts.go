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
	Recipients         []*ManualGiftRecipientReadModel
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
	OwnerTelegramUserID        int64
	EventID                    uint
	IncludeAutomaticRecipients bool
	IncludeParticipantOptions  bool
}

// OwnerManualGiftsOutput contains the owner gifts and, when requested, the
// privacy-safe participant options used to manage manual gifts.
type OwnerManualGiftsOutput struct {
	Gifts              []*ManualGiftReadModel
	ParticipantOptions []*MiniappParticipantOption
}

// GetOwnerManualGiftsHandler returns the protected owner view used by My Prizes.
type GetOwnerManualGiftsHandler struct {
	giftRepo                repository.ManualGiftRepository
	criteriaRepo            repository.CriteriaRepository
	participantRepo         repository.ParticipantRepository
	prizeDistributionReader prizeDistributionReader
}

func NewGetOwnerManualGiftsHandler(
	giftRepo repository.ManualGiftRepository,
	criteriaRepo repository.CriteriaRepository,
	participantRepo repository.ParticipantRepository,
	prizeDistributionReader prizeDistributionReader,
) *GetOwnerManualGiftsHandler {
	return &GetOwnerManualGiftsHandler{
		giftRepo:                giftRepo,
		criteriaRepo:            criteriaRepo,
		participantRepo:         participantRepo,
		prizeDistributionReader: prizeDistributionReader,
	}
}

func (h *GetOwnerManualGiftsHandler) Handle(ctx context.Context, query GetOwnerManualGiftsQuery) ([]*ManualGiftReadModel, error) {
	output, err := h.HandleDetailed(ctx, query)
	if err != nil {
		return nil, err
	}
	return output.Gifts, nil
}

// HandleDetailed returns the owner gifts together with recipient options. When
// both automatic recipients and participant options are requested, it reuses a
// single automatic prize distribution calculation for both projections.
func (h *GetOwnerManualGiftsHandler) HandleDetailed(ctx context.Context, query GetOwnerManualGiftsQuery) (*OwnerManualGiftsOutput, error) {
	log.Printf("DEBUG manual gifts query started: scope=owner owner_user_id=%d event_id=%d", query.OwnerTelegramUserID, query.EventID)
	gifts, err := h.giftRepo.FindByUserAndEvent(ctx, query.OwnerTelegramUserID, query.EventID)
	if err != nil {
		log.Printf("ERROR manual gifts query failed: scope=owner owner_user_id=%d event_id=%d stage=find_by_user_and_event error=%v", query.OwnerTelegramUserID, query.EventID, err)
		return nil, fmt.Errorf("find owner gifts for user %d event %d: %w", query.OwnerTelegramUserID, query.EventID, err)
	}

	automaticRecipients := make(map[uint][]*ManualGiftRecipientReadModel)
	automaticRecipientCount := 0
	participantOptions := make([]*MiniappParticipantOption, 0)
	if query.IncludeAutomaticRecipients || query.IncludeParticipantOptions {
		if h.prizeDistributionReader == nil {
			return nil, fmt.Errorf("prize distribution reader is unavailable")
		}
		if query.IncludeParticipantOptions && h.participantRepo == nil {
			return nil, fmt.Errorf("participant repository is unavailable")
		}
		log.Printf(
			"DEBUG [FIX:my-gifts-distribution] shared prize distribution started: event_id=%d owner_gift_count=%d include_automatic_recipients=%t include_participant_options=%t",
			query.EventID,
			len(gifts),
			query.IncludeAutomaticRecipients,
			query.IncludeParticipantOptions,
		)
		distribution, distributionErr := h.prizeDistributionReader.Handle(ctx, GetPrizeDistributionQuery{EventID: query.EventID})
		if distributionErr != nil {
			log.Printf("ERROR [FIX:my-gifts-distribution] shared prize distribution failed: event_id=%d error=%v", query.EventID, distributionErr)
			return nil, fmt.Errorf("calculate prize distribution for event %d: %w", query.EventID, distributionErr)
		}

		if query.IncludeAutomaticRecipients {
			automaticRecipients, automaticRecipientCount = automaticGiftRecipientsByGift(distribution)
		}
		if query.IncludeParticipantOptions {
			states, sourceCount, excludedCount, stateErr := loadEligibleManualGiftParticipantStatesWithAutomaticPrizeCounts(
				ctx,
				query.EventID,
				h.participantRepo,
				h.giftRepo,
				automaticPrizeCountsFromDistribution(distribution),
			)
			if stateErr != nil {
				log.Printf("ERROR [FIX:my-gifts-distribution] shared participant options failed: event_id=%d error=%v", query.EventID, stateErr)
				return nil, stateErr
			}
			participantOptions = miniappParticipantOptionsFromStates(states)
			log.Printf(
				"INFO [FIX:manual-recipient-eligibility] miniapp participant options filtered: event_id=%d source_count=%d returned_count=%d excluded_count=%d",
				query.EventID,
				sourceCount,
				len(participantOptions),
				excludedCount,
			)
		}

		log.Printf(
			"INFO [FIX:my-gifts-distribution] shared prize distribution reused: event_id=%d distribution_count=%d automatic_recipient_count=%d participant_option_count=%d",
			query.EventID,
			len(distribution),
			automaticRecipientCount,
			len(participantOptions),
		)
	}

	ownerGifts := make([]*ManualGiftReadModel, 0, len(gifts))
	for _, gift := range gifts {
		ownerGift, err := newManualGiftReadModel(ctx, h.giftRepo, h.criteriaRepo, gift)
		if err != nil {
			return nil, fmt.Errorf("load owner gift %d: %w", gift.ID, err)
		}
		if !gift.ManualDistribution {
			ownerGift.Recipients = automaticRecipients[gift.ID]
		}
		ownerGifts = append(ownerGifts, ownerGift)
	}
	log.Printf("DEBUG manual gifts query completed: scope=owner owner_user_id=%d event_id=%d returned_count=%d", query.OwnerTelegramUserID, query.EventID, len(ownerGifts))
	return &OwnerManualGiftsOutput{Gifts: ownerGifts, ParticipantOptions: participantOptions}, nil
}

func automaticGiftRecipientsByGift(
	distribution []*PrizeDistributionResult,
) (map[uint][]*ManualGiftRecipientReadModel, int) {
	recipientsByGift := make(map[uint][]*ManualGiftRecipientReadModel)
	seenRecipients := make(map[uint]map[uint]struct{})
	recipientCount := 0
	for _, participant := range distribution {
		if participant == nil || participant.ParticipantID == 0 {
			continue
		}

		giftIDs := automaticRecipientGiftIDs(participant)
		for _, giftID := range giftIDs {
			if giftID == 0 {
				continue
			}
			if seenRecipients[giftID] == nil {
				seenRecipients[giftID] = make(map[uint]struct{})
			}
			if _, exists := seenRecipients[giftID][participant.ParticipantID]; exists {
				continue
			}

			status := participant.Status
			if status == "" {
				status = string(valueobject.ParticipantStatusActive)
			}
			displayName := strings.TrimSpace(participant.ParticipantName)
			if displayName == "" {
				displayName = fmt.Sprintf("Участник #%d", participant.ParticipantID)
			}
			recipientsByGift[giftID] = append(recipientsByGift[giftID], &ManualGiftRecipientReadModel{
				ID:          participant.ParticipantID,
				DisplayName: displayName,
				Status:      status,
			})
			seenRecipients[giftID][participant.ParticipantID] = struct{}{}
			recipientCount++
		}
	}

	return recipientsByGift, recipientCount
}

func automaticRecipientGiftIDs(participant *PrizeDistributionResult) []uint {
	if len(participant.MatchedGiftAssignments) > 0 {
		giftIDs := make([]uint, 0, len(participant.MatchedGiftAssignments))
		for _, assignment := range participant.MatchedGiftAssignments {
			if assignment != nil && assignment.Gift != nil {
				giftIDs = append(giftIDs, assignment.Gift.ID)
			}
		}
		if len(giftIDs) > 0 {
			return giftIDs
		}
	}

	giftIDs := make([]uint, 0, len(participant.MatchedGifts))
	for _, gift := range participant.MatchedGifts {
		if gift != nil {
			giftIDs = append(giftIDs, gift.ID)
		}
	}
	return giftIDs
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
