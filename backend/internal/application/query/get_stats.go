package query

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// EventStats представляет статистику события
type EventStats struct {
	EventID              uint
	EventName            string
	ParticipantsCount    int
	FinishedCount        int
	GiftsCount           int
	PrizesAssignedCount  int
	ByGender             map[string]int // разбивка по полу
	ByBikeType           map[string]int // разбивка по типу велосипеда
}

// GetStatsQuery представляет запрос на получение статистики
type GetStatsQuery struct {
	EventID *uint // если nil, возвращает статистику по всем событиям
}

// GetStatsHandler обрабатывает запрос на получение статистики
type GetStatsHandler struct {
	eventRepo       repository.EventRepository
	participantRepo repository.ParticipantRepository
	giftRepo        repository.GiftRepository
	resultRepo      repository.ResultRepository
	criteriaRepo    repository.CriteriaRepository
}

// NewGetStatsHandler создаёт новый handler
func NewGetStatsHandler(
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	giftRepo repository.GiftRepository,
	resultRepo repository.ResultRepository,
	criteriaRepo repository.CriteriaRepository,
) *GetStatsHandler {
	return &GetStatsHandler{
		eventRepo:       eventRepo,
		participantRepo: participantRepo,
		giftRepo:        giftRepo,
		resultRepo:      resultRepo,
		criteriaRepo:    criteriaRepo,
	}
}

// Handle выполняет запрос на получение статистики
func (h *GetStatsHandler) Handle(ctx context.Context, query GetStatsQuery) ([]*EventStats, error) {
	var events []*entity.Event
	var err error

	if query.EventID != nil {
		// Статистика для конкретного события
		event, err := h.eventRepo.FindByID(ctx, *query.EventID)
		if err != nil {
			return nil, fmt.Errorf("failed to find event: %w", err)
		}
		if event == nil {
			return nil, fmt.Errorf("event not found")
		}
		events = []*entity.Event{event}
	} else {
		// Статистика для всех событий
		events, err = h.eventRepo.GetAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get all events: %w", err)
		}
	}

	// Собираем статистику для каждого события
	result := make([]*EventStats, 0, len(events))
	for _, event := range events {
		stats, err := h.calculateEventStats(ctx, event)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate stats for event %d: %w", event.ID, err)
		}
		result = append(result, stats)
	}

	return result, nil
}

// calculateEventStats рассчитывает статистику для одного события
func (h *GetStatsHandler) calculateEventStats(ctx context.Context, event *entity.Event) (*EventStats, error) {
	stats := &EventStats{
		EventID:   event.ID,
		EventName: event.Name,
		ByGender:  make(map[string]int),
		ByBikeType: make(map[string]int),
	}

	// Получаем участников
	participants, err := h.participantRepo.FindByEvent(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find participants: %w", err)
	}

	stats.ParticipantsCount = len(participants)

	// Подсчитываем финишировавших и разбивку по категориям
	for _, p := range participants {
		if p.IsFinished() {
			stats.FinishedCount++
		}

		// Разбивка по полу
		gender := string(p.Gender)
		stats.ByGender[gender]++

		// Разбивка по типу велосипеда
		bikeType := string(p.BikeType)
		stats.ByBikeType[bikeType]++
	}

	// Получаем подарки
	gifts, err := h.giftRepo.FindByEvent(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find gifts: %w", err)
	}
	stats.GiftsCount = len(gifts)

	// Считаем призы через prize distribution (автоматическое распределение)
	// Получаем распределение призов для подсчёта
	prizeDistributionHandler := NewGetPrizeDistributionHandler(
		h.resultRepo,
		h.giftRepo,
		h.participantRepo,
		h.criteriaRepo,
	)
	distribution, err := prizeDistributionHandler.Handle(ctx, GetPrizeDistributionQuery{
		EventID: event.ID,
	})
	if err != nil {
		// Если не удалось получить распределение, просто ставим 0
		stats.PrizesAssignedCount = 0
	} else {
		// Считаем участников с подарками
		for _, dist := range distribution {
			if len(dist.MatchedGifts) > 0 {
				stats.PrizesAssignedCount++
			}
		}
	}

	return stats, nil
}
