package query

import (
	"context"
	"fmt"
	"log"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetEligibleUnawardedParticipantIDsQuery requests participants who may receive
// a random gift: they are eligible for a manual gift and have no prize yet.
type GetEligibleUnawardedParticipantIDsQuery struct {
	EventID uint
}

// GetEligibleUnawardedParticipantIDsHandler provides a presentation-neutral
// random-recipient candidate list. It deliberately returns IDs rather than a
// Mini App DTO so it can also be used from the admin dashboard.
type GetEligibleUnawardedParticipantIDsHandler struct {
	participantRepo          repository.ParticipantRepository
	manualRecipientCountRepo repository.ManualGiftRecipientCountRepository
	prizeDistributionReader  prizeDistributionReader
}

type prizeDistributionReader interface {
	Handle(ctx context.Context, query GetPrizeDistributionQuery) ([]*PrizeDistributionResult, error)
}

// NewGetEligibleUnawardedParticipantIDsHandler builds the random-recipient
// candidate query.
func NewGetEligibleUnawardedParticipantIDsHandler(
	participantRepo repository.ParticipantRepository,
	manualRecipientCountRepo repository.ManualGiftRecipientCountRepository,
	prizeDistributionReader prizeDistributionReader,
) *GetEligibleUnawardedParticipantIDsHandler {
	return &GetEligibleUnawardedParticipantIDsHandler{
		participantRepo:          participantRepo,
		manualRecipientCountRepo: manualRecipientCountRepo,
		prizeDistributionReader:  prizeDistributionReader,
	}
}

// Handle returns eligible participant IDs which have neither an automatic nor
// a manually assigned prize.
func (h *GetEligibleUnawardedParticipantIDsHandler) Handle(
	ctx context.Context,
	query GetEligibleUnawardedParticipantIDsQuery,
) ([]uint, error) {
	log.Printf("DEBUG eligible unawarded participant IDs query started: event_id=%d", query.EventID)
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

	participantIDs := make([]uint, 0, len(states))
	for _, state := range states {
		if !state.hasPrize {
			participantIDs = append(participantIDs, state.participant.ID)
		}
	}

	log.Printf(
		"INFO [FIX:manual-recipient-eligibility] eligible unawarded participant IDs filtered: event_id=%d source_count=%d returned_count=%d excluded_count=%d",
		query.EventID,
		sourceCount,
		len(participantIDs),
		excludedCount,
	)
	return participantIDs, nil
}

type eligibleManualGiftParticipantState struct {
	participant *entity.Participant
	hasPrize    bool
}

func loadEligibleManualGiftParticipantStates(
	ctx context.Context,
	eventID uint,
	participantRepo repository.ParticipantRepository,
	manualRecipientCountRepo repository.ManualGiftRecipientCountRepository,
	prizeDistributionReader prizeDistributionReader,
) ([]eligibleManualGiftParticipantState, int, int, error) {
	automaticPrizeCounts, err := automaticPrizeCounts(ctx, eventID, prizeDistributionReader)
	if err != nil {
		return nil, 0, 0, err
	}
	return loadEligibleManualGiftParticipantStatesWithAutomaticPrizeCounts(
		ctx,
		eventID,
		participantRepo,
		manualRecipientCountRepo,
		automaticPrizeCounts,
	)
}

func loadEligibleManualGiftParticipantStatesWithAutomaticPrizeCounts(
	ctx context.Context,
	eventID uint,
	participantRepo repository.ParticipantRepository,
	manualRecipientCountRepo repository.ManualGiftRecipientCountRepository,
	automaticPrizeCounts map[uint]int,
) ([]eligibleManualGiftParticipantState, int, int, error) {
	participants, err := participantRepo.FindByEvent(ctx, eventID)
	if err != nil {
		log.Printf("ERROR manual gift recipient candidates query failed: event_id=%d stage=find_by_event error=%v", eventID, err)
		return nil, 0, 0, fmt.Errorf("find participants for manual gift candidates in event %d: %w", eventID, err)
	}

	manualPrizeCounts, err := manualRecipientCountRepo.ManualRecipientCountsByEvent(ctx, eventID)
	if err != nil {
		log.Printf("ERROR manual gift recipient candidates query failed: event_id=%d stage=manual_recipient_counts error=%v", eventID, err)
		return nil, 0, 0, fmt.Errorf("find manual gift recipients for event %d: %w", eventID, err)
	}
	states := make([]eligibleManualGiftParticipantState, 0, len(participants))
	excludedCount := 0
	for _, participant := range participants {
		if participant == nil || !participant.IsEligibleForManualGift() {
			excludedCount++
			continue
		}
		states = append(states, eligibleManualGiftParticipantState{
			participant: participant,
			hasPrize:    automaticPrizeCounts[participant.ID]+manualPrizeCounts[participant.ID] > 0,
		})
	}
	return states, len(participants), excludedCount, nil
}

func automaticPrizeCounts(
	ctx context.Context,
	eventID uint,
	prizeDistributionReader prizeDistributionReader,
) (map[uint]int, error) {
	distribution, err := prizeDistributionReader.Handle(ctx, GetPrizeDistributionQuery{EventID: eventID})
	if err != nil {
		log.Printf("ERROR manual gift recipient candidates query failed: event_id=%d stage=prize_distribution error=%v", eventID, err)
		return nil, fmt.Errorf("find automatic gift recipients for event %d: %w", eventID, err)
	}

	return automaticPrizeCountsFromDistribution(distribution), nil
}

func automaticPrizeCountsFromDistribution(distribution []*PrizeDistributionResult) map[uint]int {
	counts := make(map[uint]int)
	for _, participant := range distribution {
		if participant == nil {
			continue
		}
		// MatchedGiftAssignments is the current slot-aware representation and
		// MatchedGifts is the legacy compatibility representation. They can
		// describe the same prize, but candidate eligibility needs only the
		// non-zero fact, so counting both protects mixed readers as well.
		counts[participant.ParticipantID] += len(participant.MatchedGiftAssignments)
		counts[participant.ParticipantID] += len(participant.MatchedGifts)
	}
	return counts
}
