package query

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// Дисквалифицированный участник исключён из любых призов: по критериям, общих и
// по местам.
func TestPrizeDistributionExcludesDisqualifiedFromAllGifts(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
		prizeDistributionParticipantWithID(1),
	})
	participants[1].Status = valueobject.ParticipantStatusDisqualified
	setPrizeResultCriteria(results, 1, 1)

	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	genericGift := prizeDistributionApprovedGift(20)
	placeGift := prizeDistributionApprovedGift(30)
	placeGift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1})

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift, genericGift, placeGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{})
}

// Сошедший с дистанции (DNF) получает приз по чистым критериям.
func TestPrizeDistributionDNFReceivesPureCriteriaGift(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
		prizeDistributionParticipantWithID(1),
	})
	participants[1].Status = valueobject.ParticipantStatusDNF
	setPrizeResultCriteria(results, 1, 1)

	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 10, ruleType: "none", assignedRank: 1}},
	})
}

// DNF не получает призы по местам, общие и критерии-с-местом (привязанные к рангу).
func TestPrizeDistributionDNFExcludedFromPlaceGenericAndCriteriaPlace(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
		prizeDistributionParticipantWithID(1),
	})
	participants[1].Status = valueobject.ParticipantStatusDNF
	setPrizeResultCriteria(results, 1, 1)

	placeGift := prizeDistributionApprovedGift(10)
	placeGift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1})
	genericGift := prizeDistributionApprovedGift(20)
	criteriaPlaceGift := prizeDistributionApprovedGift(30)
	criteriaPlaceGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	criteriaPlaceGift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1})

	output := h.distributePrizeSlots(results, []*entity.Gift{placeGift, genericGift, criteriaPlaceGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{})
}

// DNF посреди зачёта не занимает место: активные участники получают
// последовательные ранги (1, 2), как будто DNF в зачёте нет.
func TestPrizeDistributionDNFDoesNotOccupyPlaceRank(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
		prizeDistributionParticipantWithID(1),
		prizeDistributionParticipantWithID(2),
		prizeDistributionParticipantWithID(3),
	})
	participants[2].Status = valueobject.ParticipantStatusDNF

	gift := prizeDistributionApprovedGift(100)
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1, 2})

	output := h.distributePrizeSlots(results, []*entity.Gift{gift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 100, ruleType: "places", targetRank: 1, assignedRank: 1}},
		3: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
	})
}

// Сквозной тест HandleDetailed: DNF-участник отсутствует в зачёте
// (FindByEventWithPlaces), но добавляется как кандидат на призы по критериям;
// дисквалифицированный исключается из распределения полностью.
func TestPrizeDistributionHandlerAddsDNFCriteriaCandidateAndExcludesDisqualified(t *testing.T) {
	// Активный участник в зачёте, без критериев.
	p1 := prizeDistributionParticipantWithID(1)
	p1.Result = &entity.Result{ID: 1, ParticipantID: 1}
	// DNF: нет в зачёте, но есть текущий результат с критерием 1.
	p2 := prizeDistributionParticipantWithID(2)
	p2.Status = valueobject.ParticipantStatusDNF
	p2.Result = &entity.Result{ID: 200, ParticipantID: 2}
	// Дисквалифицированный: тоже с критерием 1, но должен быть исключён.
	p3 := prizeDistributionParticipantWithID(3)
	p3.Status = valueobject.ParticipantStatusDisqualified
	p3.Result = &entity.Result{ID: 300, ParticipantID: 3}

	elapsed := 1001
	withPlaces := []*repository.ResultWithPlace{
		{
			Result:            &entity.Result{ID: 1, ParticipantID: 1, ElapsedTimeSec: &elapsed},
			PlaceAbsolute:     1,
			PlaceByGender:     1,
			PlaceByGenderBike: 1,
		},
	}

	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	genericGift := prizeDistributionApprovedGift(20)

	h := NewGetPrizeDistributionHandler(
		&statusResultRepoFake{withPlaces: withPlaces},
		&statusGiftRepoFake{gifts: []*entity.Gift{criteriaGift, genericGift}},
		&statusParticipantRepoFake{participants: []*entity.Participant{p1, p2, p3}},
		&statusCriteriaRepoFake{
			byResult: map[uint][]*entity.Criteria{
				200: {prizeDistributionCriteria(1)},
				300: {prizeDistributionCriteria(1)},
			},
			byGift: map[uint][]*entity.Criteria{
				10: {prizeDistributionCriteria(1)},
			},
		},
	)

	output, err := h.HandleDetailed(context.Background(), GetPrizeDistributionQuery{EventID: 77})
	if err != nil {
		t.Fatalf("HandleDetailed: %v", err)
	}

	assertDistributionGiftIDs(t, output.Results, map[uint][]uint{
		1: {20}, // активный получает общий приз
		2: {10}, // DNF получает приз по критериям
	})

	var p2row, p3row *PrizeDistributionResult
	for _, r := range output.Results {
		switch r.ParticipantID {
		case 2:
			p2row = r
		case 3:
			p3row = r
		}
	}

	if p3row != nil {
		t.Fatalf("disqualified participant 3 must be absent from distribution, got %+v", p3row)
	}
	if p2row == nil {
		t.Fatalf("DNF participant 2 must be present in distribution")
	}
	if p2row.Status != string(valueobject.ParticipantStatusDNF) {
		t.Fatalf("participant 2 status = %q, want %q", p2row.Status, valueobject.ParticipantStatusDNF)
	}
	if p2row.PlaceAbsolute != 0 {
		t.Fatalf("DNF participant 2 must have no place, got %d", p2row.PlaceAbsolute)
	}
}

func TestPrizeDistributionHandlerExcludesManualGiftsBeforeLoadingGiftCriteria(t *testing.T) {
	participant := prizeDistributionParticipantWithID(1)
	elapsed := 1000
	manualGift := prizeDistributionApprovedGift(10)
	manualGift.ManualDistribution = true

	criteriaRepo := &statusCriteriaRepoFake{
		byGift: map[uint][]*entity.Criteria{
			10: {prizeDistributionCriteria(1)},
		},
	}
	h := NewGetPrizeDistributionHandler(
		&statusResultRepoFake{withPlaces: []*repository.ResultWithPlace{{
			Result:            &entity.Result{ID: 1, ParticipantID: 1, ElapsedTimeSec: &elapsed},
			PlaceAbsolute:     1,
			PlaceByGender:     1,
			PlaceByGenderBike: 1,
		}}},
		&statusGiftRepoFake{gifts: []*entity.Gift{manualGift}},
		&statusParticipantRepoFake{participants: []*entity.Participant{participant}},
		criteriaRepo,
	)

	output, err := h.HandleDetailed(context.Background(), GetPrizeDistributionQuery{EventID: 77})
	if err != nil {
		t.Fatalf("HandleDetailed: %v", err)
	}
	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{})
	if len(output.UnassignedSlots) != 0 {
		t.Fatalf("manual gift must not create unassigned slots: %+v", output.UnassignedSlots)
	}
	if len(criteriaRepo.giftCalls) != 0 {
		t.Fatalf("manual gift criteria must not be loaded, calls=%v", criteriaRepo.giftCalls)
	}
}

// Фейки репозиториев: встраиваем интерфейс, переопределяем только нужные методы.
type statusResultRepoFake struct {
	repository.ResultRepository
	withPlaces []*repository.ResultWithPlace
}

func (f *statusResultRepoFake) FindByEventWithPlaces(_ context.Context, _ uint) ([]*repository.ResultWithPlace, error) {
	return f.withPlaces, nil
}

type statusGiftRepoFake struct {
	repository.GiftRepository
	gifts []*entity.Gift
}

func (f *statusGiftRepoFake) FindByEventAndReviewStatus(_ context.Context, _ uint, _ entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return f.gifts, nil
}

type statusParticipantRepoFake struct {
	repository.ParticipantRepository
	participants []*entity.Participant
}

func (f *statusParticipantRepoFake) FindByEvent(_ context.Context, _ uint) ([]*entity.Participant, error) {
	return f.participants, nil
}

type statusCriteriaRepoFake struct {
	repository.CriteriaRepository
	byResult  map[uint][]*entity.Criteria
	byGift    map[uint][]*entity.Criteria
	giftCalls []uint
}

func (f *statusCriteriaRepoFake) FindByResult(_ context.Context, resultID uint) ([]*entity.Criteria, error) {
	return f.byResult[resultID], nil
}

func (f *statusCriteriaRepoFake) FindByGift(_ context.Context, giftID uint) ([]*entity.Criteria, error) {
	f.giftCalls = append(f.giftCalls, giftID)
	return f.byGift[giftID], nil
}
