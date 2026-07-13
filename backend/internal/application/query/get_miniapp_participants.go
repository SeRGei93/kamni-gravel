package query

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// MiniappParticipantOption is a privacy-safe recipient option. It excludes
// Telegram user ID, notes, registration dates, result details, and award
// details; HasPrize is only the boolean selection hint for gift owners.
type MiniappParticipantOption struct {
	ID          uint
	DisplayName string
	Username    string
	Status      string
	HasPrize    bool
}

// GetMiniappParticipantsQuery requests participant options for the active event.
type GetMiniappParticipantsQuery struct {
	EventID uint
}

// GetMiniappParticipantsHandler returns a minimal, deterministic recipient list.
type GetMiniappParticipantsHandler struct {
	participantRepo          repository.ParticipantRepository
	manualRecipientCountRepo repository.ManualGiftRecipientCountRepository
	prizeDistributionReader  miniappPrizeDistributionReader
}

type miniappPrizeDistributionReader interface {
	Handle(ctx context.Context, query GetPrizeDistributionQuery) ([]*PrizeDistributionResult, error)
}

// NewGetMiniappParticipantsHandler builds the protected recipient-options query.
func NewGetMiniappParticipantsHandler(
	participantRepo repository.ParticipantRepository,
	manualRecipientCountRepo repository.ManualGiftRecipientCountRepository,
	prizeDistributionReader miniappPrizeDistributionReader,
) *GetMiniappParticipantsHandler {
	return &GetMiniappParticipantsHandler{
		participantRepo:          participantRepo,
		manualRecipientCountRepo: manualRecipientCountRepo,
		prizeDistributionReader:  prizeDistributionReader,
	}
}

func (h *GetMiniappParticipantsHandler) Handle(ctx context.Context, query GetMiniappParticipantsQuery) ([]*MiniappParticipantOption, error) {
	log.Printf("DEBUG miniapp participant options query started: event_id=%d", query.EventID)
	participants, err := h.participantRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		log.Printf("ERROR miniapp participant options query failed: event_id=%d stage=find_by_event error=%v", query.EventID, err)
		return nil, fmt.Errorf("find miniapp participants for event %d: %w", query.EventID, err)
	}

	manualPrizeCounts, err := h.manualRecipientCountRepo.ManualRecipientCountsByEvent(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("find manual miniapp prize recipients for event %d: %w", query.EventID, err)
	}
	automaticPrizeCounts, err := miniappAutomaticPrizeCounts(ctx, query.EventID, h.prizeDistributionReader)
	if err != nil {
		return nil, err
	}

	options := make([]*MiniappParticipantOption, 0, len(participants))
	excludedCount := 0
	for _, participant := range participants {
		if !participant.IsEligibleForManualGift() {
			excludedCount++
			continue
		}
		option := newMiniappParticipantOption(participant)
		option.HasPrize = automaticPrizeCounts[participant.ID]+manualPrizeCounts[participant.ID] > 0
		options = append(options, option)
	}
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].HasPrize != options[j].HasPrize {
			return !options[i].HasPrize
		}
		leftName := strings.ToLower(options[i].DisplayName)
		rightName := strings.ToLower(options[j].DisplayName)
		if leftName == rightName {
			return options[i].ID < options[j].ID
		}
		return leftName < rightName
	})

	log.Printf(
		"INFO [FIX:manual-recipient-eligibility] miniapp participant options filtered: event_id=%d source_count=%d returned_count=%d excluded_count=%d",
		query.EventID,
		len(participants),
		len(options),
		excludedCount,
	)
	return options, nil
}

func miniappAutomaticPrizeCounts(
	ctx context.Context,
	eventID uint,
	prizeDistributionReader miniappPrizeDistributionReader,
) (map[uint]int, error) {
	distribution, err := prizeDistributionReader.Handle(ctx, GetPrizeDistributionQuery{EventID: eventID})
	if err != nil {
		return nil, fmt.Errorf("find automatic miniapp prize recipients for event %d: %w", eventID, err)
	}

	counts := make(map[uint]int)
	for _, participant := range distribution {
		if len(participant.MatchedGiftAssignments) > 0 {
			counts[participant.ParticipantID] += len(participant.MatchedGiftAssignments)
			continue
		}
		counts[participant.ParticipantID] += len(participant.MatchedGifts)
	}
	return counts, nil
}

func newMiniappParticipantOption(participant *entity.Participant) *MiniappParticipantOption {
	status := participant.Status
	if status == "" {
		status = valueobject.ParticipantStatusActive
	}
	return &MiniappParticipantOption{
		ID:          participant.ID,
		DisplayName: manualGiftRecipientDisplayName(participant),
		Username:    manualGiftRecipientUsername(participant),
		Status:      string(status),
	}
}
