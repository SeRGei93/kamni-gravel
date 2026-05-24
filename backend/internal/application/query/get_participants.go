package query

import (
	"context"
	"fmt"
	"sort"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetParticipantsQuery представляет запрос на получение участников
type GetParticipantsQuery struct {
	EventID      uint
	BikeType     *string // фильтр по типу велосипеда (опционально)
	Gender       *string // фильтр по полу (опционально)
	IsFinished   *bool   // фильтр по статусу завершения (опционально)
}

// ParticipantWithPlace представляет участника с рассчитанным местом
type ParticipantWithPlace struct {
	*entity.Participant
	Place int // место в зачёте (0 если не финишировал или нет времени)
}

// GetParticipantsHandler обрабатывает запрос на получение участников
type GetParticipantsHandler struct {
	participantRepo repository.ParticipantRepository
}

// NewGetParticipantsHandler создаёт новый handler
func NewGetParticipantsHandler(
	participantRepo repository.ParticipantRepository,
) *GetParticipantsHandler {
	return &GetParticipantsHandler{
		participantRepo: participantRepo,
	}
}

// Handle выполняет запрос на получение участников с расчётом мест
func (h *GetParticipantsHandler) Handle(ctx context.Context, query GetParticipantsQuery) ([]*ParticipantWithPlace, error) {
	// Получаем всех участников события
	participants, err := h.participantRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to find participants: %w", err)
	}

	// Применяем фильтры
	filtered := h.applyFilters(participants, query)

	// Рассчитываем места для финишировавших участников
	result := h.calculatePlaces(filtered)

	return result, nil
}

// applyFilters применяет фильтры к списку участников
func (h *GetParticipantsHandler) applyFilters(participants []*entity.Participant, query GetParticipantsQuery) []*entity.Participant {
	var filtered []*entity.Participant

	for _, p := range participants {
		// Фильтр по типу велосипеда
		if query.BikeType != nil && string(p.BikeType) != *query.BikeType {
			continue
		}

		// Фильтр по полу
		if query.Gender != nil && string(p.Gender) != *query.Gender {
			continue
		}

		// Фильтр по статусу завершения
		if query.IsFinished != nil && p.IsFinished() != *query.IsFinished {
			continue
		}

		filtered = append(filtered, p)
	}

	return filtered
}

// calculatePlaces рассчитывает места для финишировавших участников
func (h *GetParticipantsHandler) calculatePlaces(participants []*entity.Participant) []*ParticipantWithPlace {
	// Разделяем на финишировавших с временем и остальных
	var finished []*entity.Participant
	var others []*entity.Participant

	for _, p := range participants {
		elapsedTime := p.GetElapsedTimeSec()
		if p.IsFinished() && elapsedTime != nil && *elapsedTime > 0 {
			finished = append(finished, p)
		} else {
			others = append(others, p)
		}
	}

	// Сортируем финишировавших по времени (по возрастанию)
	sort.Slice(finished, func(i, j int) bool {
		timeI := finished[i].GetElapsedTimeSec()
		timeJ := finished[j].GetElapsedTimeSec()
		if timeI == nil {
			return false
		}
		if timeJ == nil {
			return true
		}
		return *timeI < *timeJ
	})

	// Создаём результат с местами
	result := make([]*ParticipantWithPlace, 0, len(participants))

	// Добавляем финишировавших с местами
	for i, p := range finished {
		result = append(result, &ParticipantWithPlace{
			Participant: p,
			Place:       i + 1,
		})
	}

	// Добавляем остальных без места
	for _, p := range others {
		result = append(result, &ParticipantWithPlace{
			Participant: p,
			Place:       0,
		})
	}

	return result
}

// GetParticipantByIDQuery представляет запрос на получение участника по ID
type GetParticipantByIDQuery struct {
	ParticipantID uint
}

// GetParticipantByIDHandler обрабатывает запрос на получение участника по ID
type GetParticipantByIDHandler struct {
	participantRepo repository.ParticipantRepository
}

// NewGetParticipantByIDHandler создаёт новый handler
func NewGetParticipantByIDHandler(
	participantRepo repository.ParticipantRepository,
) *GetParticipantByIDHandler {
	return &GetParticipantByIDHandler{
		participantRepo: participantRepo,
	}
}

// Handle выполняет запрос на получение участника по ID
func (h *GetParticipantByIDHandler) Handle(ctx context.Context, query GetParticipantByIDQuery) (*entity.Participant, error) {
	participant, err := h.participantRepo.FindByID(ctx, query.ParticipantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find participant: %w", err)
	}
	return participant, nil
}
