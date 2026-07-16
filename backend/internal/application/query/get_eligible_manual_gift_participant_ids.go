package query

import (
	"context"
	"fmt"
	"log"

	"gravel_bot/internal/domain/repository"
)

// GetEligibleManualGiftParticipantIDsQuery requests participants who may receive
// a manual gift regardless of prizes they have already received.
type GetEligibleManualGiftParticipantIDsQuery struct {
	EventID uint
}

// GetEligibleManualGiftParticipantIDsHandler returns the presentation-neutral
// candidate list for the random-recipient action that includes awarded users.
type GetEligibleManualGiftParticipantIDsHandler struct {
	participantRepo repository.ParticipantRepository
}

// NewGetEligibleManualGiftParticipantIDsHandler builds the candidate query for
// manual gifts that may be assigned to an already awarded participant.
func NewGetEligibleManualGiftParticipantIDsHandler(
	participantRepo repository.ParticipantRepository,
) *GetEligibleManualGiftParticipantIDsHandler {
	return &GetEligibleManualGiftParticipantIDsHandler{participantRepo: participantRepo}
}

// Handle returns IDs of finished or DNF participants in the requested event.
// It deliberately does not read automatic distribution or manual-recipient
// counts: existing prizes must not exclude a candidate for this action.
func (h *GetEligibleManualGiftParticipantIDsHandler) Handle(
	ctx context.Context,
	query GetEligibleManualGiftParticipantIDsQuery,
) ([]uint, error) {
	log.Printf("DEBUG eligible manual gift participant IDs query started: event_id=%d", query.EventID)
	participants, err := h.participantRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		log.Printf("ERROR eligible manual gift participant IDs query failed: event_id=%d stage=find_by_event error=%v", query.EventID, err)
		return nil, fmt.Errorf("find participants for eligible manual gift candidates in event %d: %w", query.EventID, err)
	}

	participantIDs := make([]uint, 0, len(participants))
	excludedCount := 0
	for _, participant := range participants {
		if participant == nil || !participant.IsEligibleForManualGift() {
			excludedCount++
			continue
		}
		participantIDs = append(participantIDs, participant.ID)
	}

	log.Printf(
		"DEBUG eligible manual gift participant IDs query completed: event_id=%d source_count=%d candidate_count=%d excluded_count=%d",
		query.EventID,
		len(participants),
		len(participantIDs),
		excludedCount,
	)
	return participantIDs, nil
}
