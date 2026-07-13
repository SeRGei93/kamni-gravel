package query

import (
	"context"
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
	prizeDistributionReader  prizeDistributionReader
}

// NewGetMiniappParticipantsHandler builds the protected recipient-options query.
func NewGetMiniappParticipantsHandler(
	participantRepo repository.ParticipantRepository,
	manualRecipientCountRepo repository.ManualGiftRecipientCountRepository,
	prizeDistributionReader prizeDistributionReader,
) *GetMiniappParticipantsHandler {
	return &GetMiniappParticipantsHandler{
		participantRepo:          participantRepo,
		manualRecipientCountRepo: manualRecipientCountRepo,
		prizeDistributionReader:  prizeDistributionReader,
	}
}

func (h *GetMiniappParticipantsHandler) Handle(ctx context.Context, query GetMiniappParticipantsQuery) ([]*MiniappParticipantOption, error) {
	log.Printf("DEBUG miniapp participant options query started: event_id=%d", query.EventID)
	states, sourceCount, excludedCount, err := loadEligibleManualGiftParticipantStates(
		ctx,
		query.EventID,
		h.participantRepo,
		h.manualRecipientCountRepo,
		h.prizeDistributionReader,
	)
	if err != nil {
		return nil, err
	}

	options := make([]*MiniappParticipantOption, 0, len(states))
	for _, state := range states {
		option := newMiniappParticipantOption(state.participant)
		option.HasPrize = state.hasPrize
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
		sourceCount,
		len(options),
		excludedCount,
	)
	return options, nil
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
