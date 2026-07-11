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
// Telegram user ID, notes, registration dates, and result details.
type MiniappParticipantOption struct {
	ID          uint
	DisplayName string
	Username    string
	Status      string
}

// GetMiniappParticipantsQuery requests participant options for the active event.
type GetMiniappParticipantsQuery struct {
	EventID uint
}

// GetMiniappParticipantsHandler returns a minimal, deterministic recipient list.
type GetMiniappParticipantsHandler struct {
	participantRepo repository.ParticipantRepository
}

func NewGetMiniappParticipantsHandler(participantRepo repository.ParticipantRepository) *GetMiniappParticipantsHandler {
	return &GetMiniappParticipantsHandler{participantRepo: participantRepo}
}

func (h *GetMiniappParticipantsHandler) Handle(ctx context.Context, query GetMiniappParticipantsQuery) ([]*MiniappParticipantOption, error) {
	log.Printf("DEBUG miniapp participant options query started: event_id=%d", query.EventID)
	participants, err := h.participantRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		log.Printf("ERROR miniapp participant options query failed: event_id=%d stage=find_by_event error=%v", query.EventID, err)
		return nil, fmt.Errorf("find miniapp participants for event %d: %w", query.EventID, err)
	}

	options := make([]*MiniappParticipantOption, 0, len(participants))
	for _, participant := range participants {
		options = append(options, newMiniappParticipantOption(participant))
	}
	sort.SliceStable(options, func(i, j int) bool {
		leftName := strings.ToLower(options[i].DisplayName)
		rightName := strings.ToLower(options[j].DisplayName)
		if leftName == rightName {
			return options[i].ID < options[j].ID
		}
		return leftName < rightName
	})

	log.Printf("DEBUG miniapp participant options query completed: event_id=%d returned_count=%d", query.EventID, len(options))
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
