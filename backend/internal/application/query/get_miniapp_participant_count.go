package query

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetMiniappParticipantCountQuery представляет запрос количества участников для фильтра Mini App.
type GetMiniappParticipantCountQuery struct {
	EventID  uint
	Gender   string
	BikeType string
}

// GetMiniappParticipantCountHandler считает зарегистрированных участников для фильтра Mini App.
type GetMiniappParticipantCountHandler struct {
	participantRepo repository.ParticipantRepository
}

func NewGetMiniappParticipantCountHandler(
	participantRepo repository.ParticipantRepository,
) *GetMiniappParticipantCountHandler {
	return &GetMiniappParticipantCountHandler{
		participantRepo: participantRepo,
	}
}

func (h *GetMiniappParticipantCountHandler) Handle(ctx context.Context, query GetMiniappParticipantCountQuery) (int, error) {
	genderFilter, err := normalizeMiniappGenderFilter(query.Gender)
	if err != nil {
		return 0, err
	}
	bikeTypeFilter, err := normalizeMiniappBikeTypeFilter(query.BikeType)
	if err != nil {
		return 0, err
	}

	participants, err := h.participantRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		return 0, fmt.Errorf("failed to find miniapp participants for event %d gender=%s bike_type=%s: %w", query.EventID, genderFilter, bikeTypeFilter, err)
	}

	return countMiniappParticipants(participants, genderFilter, bikeTypeFilter), nil
}

func countMiniappParticipants(participants []*entity.Participant, genderFilter, bikeTypeFilter string) int {
	count := 0
	for _, participant := range participants {
		if participant == nil {
			continue
		}
		if genderFilter != "all" && string(participant.Gender) != genderFilter {
			continue
		}
		if bikeTypeFilter != "all" && string(participant.BikeType) != bikeTypeFilter {
			continue
		}
		count++
	}
	return count
}
