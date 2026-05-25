package query

import (
	"context"
	"fmt"
	"sort"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// GetPrizeDistributionQuery представляет запрос на получение распределения призов
type GetPrizeDistributionQuery struct {
	EventID uint
}

// GetPrizeDistributionHandler обрабатывает запрос на получение распределения призов
type GetPrizeDistributionHandler struct {
	resultRepo      repository.ResultRepository
	giftRepo        repository.GiftRepository
	participantRepo repository.ParticipantRepository
	criteriaRepo    repository.CriteriaRepository
}

// NewGetPrizeDistributionHandler создаёт новый handler
func NewGetPrizeDistributionHandler(
	resultRepo repository.ResultRepository,
	giftRepo repository.GiftRepository,
	participantRepo repository.ParticipantRepository,
	criteriaRepo repository.CriteriaRepository,
) *GetPrizeDistributionHandler {
	return &GetPrizeDistributionHandler{
		resultRepo:      resultRepo,
		giftRepo:        giftRepo,
		participantRepo: participantRepo,
		criteriaRepo:    criteriaRepo,
	}
}

// Handle выполняет запрос на получение распределения призов
func (h *GetPrizeDistributionHandler) Handle(ctx context.Context, query GetPrizeDistributionQuery) ([]*PrizeDistributionResult, error) {
	// Получаем результаты с местами
	resultsWithPlaces, err := h.resultRepo.FindByEventWithPlaces(ctx, query.EventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results with places: %w", err)
	}

	// В распределении участвуют только подарки, проверенные администратором.
	gifts, err := h.giftRepo.FindByEventAndReviewStatus(ctx, query.EventID, entity.GiftReviewStatusApproved)
	if err != nil {
		return nil, fmt.Errorf("failed to get approved gifts: %w", err)
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
	MatchReason     string         // "criteria", "place", "no_match"
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

type giftMatchPriority int

const (
	giftMatchPriorityCriteriaPlace giftMatchPriority = iota
	giftMatchPriorityCriteria
	giftMatchPriorityPlace
	giftMatchPriorityGeneric
)

// findAllMatchingGifts находит подходящие подарки лучшего приоритета.
func (h *GetPrizeDistributionHandler) findAllMatchingGifts(
	rwp *repository.ResultWithPlace,
	participant *entity.Participant,
	gifts []*entity.Gift,
	usedGifts map[uint]bool,
) []*entity.Gift {
	matchesByPriority := map[giftMatchPriority][]*entity.Gift{
		giftMatchPriorityCriteriaPlace: {},
		giftMatchPriorityCriteria:      {},
		giftMatchPriorityPlace:         {},
		giftMatchPriorityGeneric:       {},
	}

	for _, gift := range gifts {
		// Пропускаем уже использованные
		if usedGifts[gift.ID] {
			continue
		}

		// Сначала применяем обязательные фильтры допуска: статус проверки, тип велосипеда и пол.
		if gift.ReviewStatus != entity.GiftReviewStatusApproved {
			continue
		}
		if gift.BikeTypeFilter != "" && gift.BikeTypeFilter != "all" && gift.BikeTypeFilter != string(participant.BikeType) {
			continue
		}
		if gift.GenderFilter != "" && gift.GenderFilter != "all" && gift.GenderFilter != string(participant.Gender) {
			continue
		}

		priority, ok := h.classifyGiftMatch(gift, rwp)
		if !ok {
			continue
		}

		matchesByPriority[priority] = append(matchesByPriority[priority], gift)
	}

	// Приоритет бизнес-логики: критерии важнее места. Место используется как
	// вторичный уточняющий сигнал внутри критериальных совпадений, затем идут
	// place-only и generic. Возвращаем все подарки только одного лучшего уровня,
	// чтобы участник с критериальным подарком не получал впридачу generic/place-only.
	priorities := []giftMatchPriority{
		giftMatchPriorityCriteriaPlace,
		giftMatchPriorityCriteria,
		giftMatchPriorityPlace,
		giftMatchPriorityGeneric,
	}
	for _, priority := range priorities {
		matches := matchesByPriority[priority]
		if len(matches) == 0 {
			continue
		}
		sort.SliceStable(matches, func(i, j int) bool {
			return matches[i].ID < matches[j].ID
		})
		return matches
	}

	return nil
}

func (h *GetPrizeDistributionHandler) classifyGiftMatch(gift *entity.Gift, rwp *repository.ResultWithPlace) (giftMatchPriority, bool) {
	hasCriteria := len(gift.Criteria) > 0
	hasPlace := gift.Place != nil

	if hasCriteria && !h.allCriteriaMatch(gift.Criteria, rwp.Result.Criteria) {
		return 0, false
	}

	placeMatches := false
	if hasPlace {
		expectedPlace := h.getExpectedPlace(gift, rwp)
		placeMatches = *gift.Place == expectedPlace
		if !placeMatches {
			return 0, false
		}
	}

	switch {
	case hasCriteria && hasPlace:
		return giftMatchPriorityCriteriaPlace, true
	case hasCriteria:
		return giftMatchPriorityCriteria, true
	case hasPlace:
		return giftMatchPriorityPlace, true
	default:
		return giftMatchPriorityGeneric, true
	}
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
	if gift.GenderFilter == "" || gift.GenderFilter == "all" {
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
