package query

import (
	"context"
	"fmt"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetPrizeDistributionQuery представляет запрос на получение распределения призов
type GetPrizeDistributionQuery struct {
	EventID uint
}

// GetPrizeDistributionHandler обрабатывает запрос на получение распределения призов
type GetPrizeDistributionHandler struct {
	resultRepo     repository.ResultRepository
	giftRepo       repository.GiftRepository
	participantRepo repository.ParticipantRepository
	criteriaRepo   repository.CriteriaRepository
}

// NewGetPrizeDistributionHandler создаёт новый handler
func NewGetPrizeDistributionHandler(
	resultRepo repository.ResultRepository,
	giftRepo repository.GiftRepository,
	participantRepo repository.ParticipantRepository,
	criteriaRepo repository.CriteriaRepository,
) *GetPrizeDistributionHandler {
	return &GetPrizeDistributionHandler{
		resultRepo:     resultRepo,
		giftRepo:       giftRepo,
		participantRepo: participantRepo,
		criteriaRepo:   criteriaRepo,
	}
}

// Handle выполняет запрос на получение распределения призов
func (h *GetPrizeDistributionHandler) Handle(ctx context.Context, query GetPrizeDistributionQuery) ([]*PrizeDistributionResult, error) {
	// Получаем результаты с местами
	resultsWithPlaces, err := h.resultRepo.FindByEventWithPlaces(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results with places: %w", err)
	}

	// Получаем все подарки события
	gifts, err := h.giftRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gifts: %w", err)
	}

	// Загружаем критерии для подарков
	for _, gift := range gifts {
		criteria, err := h.criteriaRepo.FindByGift(ctx, gift.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get criteria for gift %d: %w", gift.ID, err)
		}
		gift.Criteria = criteria
	}

	// Загружаем критерии для результатов
	for _, rwp := range resultsWithPlaces {
		criteria, err := h.criteriaRepo.FindByResult(ctx, rwp.Result.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get criteria for result %d: %w", rwp.Result.ID, err)
		}
		rwp.Result.Criteria = criteria
	}

	// Получаем участников для маппинга
	participants, err := h.participantRepo.FindByEvent(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	// Создаём мапу участников по ID
	participantMap := make(map[uint]*entity.Participant)
	for _, p := range participants {
		participantMap[p.ID] = p
	}

	// Распределяем призы
	distribution := h.distributePrizes(resultsWithPlaces, gifts, participantMap)

	return distribution, nil
}

// PrizeDistributionResult представляет результат распределения приза
type PrizeDistributionResult struct {
	ParticipantID   uint
	ParticipantName string
	Gender          string
	BikeType        string
	PlaceAbsolute   int
	PlaceByGender   int
	ResultCriteria  []*entity.Criteria
	MatchedGifts    []*entity.Gift // Все подходящие подарки
	MatchReason     string // "criteria", "place", "no_match"
}

// distributePrizes распределяет призы по алгоритму matching
func (h *GetPrizeDistributionHandler) distributePrizes(
	resultsWithPlaces []*repository.ResultWithPlace,
	gifts []*entity.Gift,
	participantMap map[uint]*entity.Participant,
) []*PrizeDistributionResult {
	var distribution []*PrizeDistributionResult
	usedGifts := make(map[uint]bool) // Отслеживаем использованные подарки

	// Для каждого результата находим ВСЕ подходящие подарки
	for _, rwp := range resultsWithPlaces {
		participant := participantMap[rwp.Result.ParticipantID]
		if participant == nil {
			continue
		}

		result := &PrizeDistributionResult{
			ParticipantID:   participant.ID,
			ParticipantName: participant.User.Username,
			Gender:          string(participant.Gender),
			BikeType:        string(participant.BikeType),
			PlaceAbsolute:   rwp.PlaceAbsolute,
			PlaceByGender:   rwp.PlaceByGender,
			ResultCriteria:  rwp.Result.Criteria,
			MatchedGifts:    []*entity.Gift{},
			MatchReason:     "no_match",
		}

		// Ищем ВСЕ подходящие подарки
		matchedGifts := h.findAllMatchingGifts(rwp, participant, gifts, usedGifts)
		if len(matchedGifts) > 0 {
			result.MatchedGifts = matchedGifts
			result.MatchReason = h.getMatchReason(matchedGifts[0], rwp)
			// Помечаем все подарки как использованные
			for _, gift := range matchedGifts {
				usedGifts[gift.ID] = true
			}
		}

		distribution = append(distribution, result)
	}

	return distribution
}

// findAllMatchingGifts находит ВСЕ подходящие подарки
func (h *GetPrizeDistributionHandler) findAllMatchingGifts(
	rwp *repository.ResultWithPlace,
	participant *entity.Participant,
	gifts []*entity.Gift,
	usedGifts map[uint]bool,
) []*entity.Gift {
	var matchedGifts []*entity.Gift

	for _, gift := range gifts {
		// Пропускаем уже использованные
		if usedGifts[gift.ID] {
			continue
		}

		// 1. Проверяем bike_type_filter (пустая строка = "all")
		if gift.BikeTypeFilter != "" && gift.BikeTypeFilter != "all" && gift.BikeTypeFilter != string(participant.BikeType) {
			continue
		}

		// 2. Проверяем gender_filter (пустая строка = "all")
		if gift.GenderFilter != "" && gift.GenderFilter != "all" && gift.GenderFilter != string(participant.Gender) {
			continue
		}

		// 3. Проверяем критерии (все должны совпадать)
		if len(gift.Criteria) > 0 {
			if !h.allCriteriaMatch(gift.Criteria, rwp.Result.Criteria) {
				continue
			}
		}

		// 4. Проверяем place
		if gift.Place != nil {
			expectedPlace := h.getExpectedPlace(gift, rwp)
			if *gift.Place != expectedPlace {
				continue
			}
		}

		// Подарок подходит - добавляем в список
		matchedGifts = append(matchedGifts, gift)
	}

	return matchedGifts
}

// findMatchingGift находит первый подходящий подарок (для обратной совместимости)
func (h *GetPrizeDistributionHandler) findMatchingGift(
	rwp *repository.ResultWithPlace,
	participant *entity.Participant,
	gifts []*entity.Gift,
	usedGifts map[uint]bool,
) *entity.Gift {
	matchedGifts := h.findAllMatchingGifts(rwp, participant, gifts, usedGifts)
	if len(matchedGifts) == 0 {
		return nil
	}
	return matchedGifts[0]
}

// Старая логика для совместимости
func (h *GetPrizeDistributionHandler) findMatchingGiftOld(
	rwp *repository.ResultWithPlace,
	participant *entity.Participant,
	gifts []*entity.Gift,
	usedGifts map[uint]bool,
) *entity.Gift {
	// Фильтруем подарки по приоритету
	var candidates []*entity.Gift

	for _, gift := range gifts {
		// Пропускаем уже использованные
		if usedGifts[gift.ID] {
			continue
		}

		// 1. Проверяем bike_type_filter
		if gift.BikeTypeFilter != "all" && gift.BikeTypeFilter != string(participant.BikeType) {
			continue
		}

		// 2. Проверяем gender_filter
		if gift.GenderFilter != "all" && gift.GenderFilter != string(participant.Gender) {
			continue
		}

		// 3. Проверяем критерии (все должны совпадать)
		if len(gift.Criteria) > 0 {
			if !h.allCriteriaMatch(gift.Criteria, rwp.Result.Criteria) {
				continue
			}
		}

		// 4. Проверяем place
		if gift.Place != nil {
			expectedPlace := h.getExpectedPlace(gift, rwp)
			if *gift.Place != expectedPlace {
				continue
			}
		}

		candidates = append(candidates, gift)
	}

	// Приоритет: сначала с критериями, потом с местом, потом остальные
	if len(candidates) == 0 {
		return nil
	}

	// Сортируем по приоритету
	var bestGift *entity.Gift
	for _, gift := range candidates {
		if bestGift == nil {
			bestGift = gift
			continue
		}

		// Приоритет: критерии > место > остальные
		bestHasCriteria := len(bestGift.Criteria) > 0
		giftHasCriteria := len(gift.Criteria) > 0
		bestHasPlace := bestGift.Place != nil
		giftHasPlace := gift.Place != nil

		if giftHasCriteria && !bestHasCriteria {
			bestGift = gift
		} else if giftHasCriteria == bestHasCriteria {
			if giftHasPlace && !bestHasPlace {
				bestGift = gift
			}
		}
	}

	return bestGift
}

// allCriteriaMatch проверяет, что все критерии подарка присутствуют в результате
func (h *GetPrizeDistributionHandler) allCriteriaMatch(giftCriteria, resultCriteria []*entity.Criteria) bool {
	if len(giftCriteria) == 0 {
		return true
	}

	if len(resultCriteria) == 0 {
		return false
	}

	// Создаём мапу критериев результата
	resultCriteriaMap := make(map[uint]bool)
	for _, c := range resultCriteria {
		resultCriteriaMap[c.ID] = true
	}

	// Проверяем, что все критерии подарка есть в результате
	for _, gc := range giftCriteria {
		if !resultCriteriaMap[gc.ID] {
			return false
		}
	}

	return true
}

// getExpectedPlace возвращает ожидаемое место в зависимости от gender_filter подарка
func (h *GetPrizeDistributionHandler) getExpectedPlace(gift *entity.Gift, rwp *repository.ResultWithPlace) int {
	if gift.GenderFilter == "all" {
		return rwp.PlaceAbsolute
	}
	return rwp.PlaceByGender
}

// getMatchReason определяет причину совпадения
func (h *GetPrizeDistributionHandler) getMatchReason(gift *entity.Gift, rwp *repository.ResultWithPlace) string {
	if len(gift.Criteria) > 0 {
		return "criteria"
	}
	if gift.Place != nil {
		return "place"
	}
	return "match"
}
