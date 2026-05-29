package query

import (
	"fmt"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

// Prize distribution test matrix:
// - rule types: none, explicit places/ranges, last_n, cleared rule through update/API tests;
// - group scopes: all/all, gender/all, all/bike, gender/bike after criteria filtering;
// - criteria: no criteria, single criterion, all criteria required, criteria-before-rank;
// - fallback: exact rank, outside group size, occupied target, no duplicate same gift/participant, group isolation;
// - priority/capacity: criteria+place, criteria-only, place-only, generic, same-priority multi-gift, lower-priority blocking;
// - response: legacy matched_gifts plus assignment metadata with rule type, target rank, assigned rank, and fallback flag.

func TestPrizeDistributionAssignsExplicitPlaceRangeAsSlots(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(16)
	gift := prizeDistributionApprovedGift(100)
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{10, 11, 12, 13, 14, 15})

	distribution := h.distributePrizes(results, []*entity.Gift{gift}, participants)

	assertPrizeAssignments(t, distribution, map[uint][]prizeAssignmentExpectation{
		10: {{giftID: 100, ruleType: "places", targetRank: 10, assignedRank: 10}},
		11: {{giftID: 100, ruleType: "places", targetRank: 11, assignedRank: 11}},
		12: {{giftID: 100, ruleType: "places", targetRank: 12, assignedRank: 12}},
		13: {{giftID: 100, ruleType: "places", targetRank: 13, assignedRank: 13}},
		14: {{giftID: 100, ruleType: "places", targetRank: 14, assignedRank: 14}},
		15: {{giftID: 100, ruleType: "places", targetRank: 15, assignedRank: 15}},
	})
}

func TestPrizeDistributionAssignsLastNInsideFemaleGravelGroup(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
		prizeDistributionParticipantWithProfile(1, valueobject.GenderMale, valueobject.BikeTypeGravel),
		prizeDistributionParticipantWithProfile(2, valueobject.GenderFemale, valueobject.BikeTypeGravel),
		prizeDistributionParticipantWithProfile(3, valueobject.GenderFemale, valueobject.BikeTypeMTB),
		prizeDistributionParticipantWithProfile(4, valueobject.GenderFemale, valueobject.BikeTypeGravel),
		prizeDistributionParticipantWithProfile(5, valueobject.GenderFemale, valueobject.BikeTypeGravel),
	})
	gift := prizeDistributionApprovedGift(100)
	gift.GenderFilter = "female"
	gift.BikeTypeFilter = "gravel"
	gift.PlaceRule = mustGiftPlaceRuleLastN(t, 2)

	distribution := h.distributePrizes(results, []*entity.Gift{gift}, participants)

	assertPrizeAssignments(t, distribution, map[uint][]prizeAssignmentExpectation{
		4: {{giftID: 100, ruleType: "last_n", targetRank: 2, assignedRank: 2}},
		5: {{giftID: 100, ruleType: "last_n", targetRank: 3, assignedRank: 3}},
	})
}

func TestPrizeDistributionAppliesCriteriaBeforeRankCalculation(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(2)
	results[1].Result.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	gift := prizeDistributionApprovedGift(100)
	gift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1})

	distribution := h.distributePrizes(results, []*entity.Gift{gift}, participants)

	assertPrizeAssignments(t, distribution, map[uint][]prizeAssignmentExpectation{
		2: {{giftID: 100, ruleType: "places", targetRank: 1, assignedRank: 1}},
	})
}

func TestPrizeDistributionRanksBikeOnlyAbsoluteGroup(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
		prizeDistributionParticipantWithProfile(1, valueobject.GenderMale, valueobject.BikeTypeRoad),
		prizeDistributionParticipantWithProfile(2, valueobject.GenderMale, valueobject.BikeTypeGravel),
		prizeDistributionParticipantWithProfile(3, valueobject.GenderFemale, valueobject.BikeTypeGravel),
	})
	gift := prizeDistributionApprovedGift(100)
	gift.BikeTypeFilter = "gravel"
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{2})

	distribution := h.distributePrizes(results, []*entity.Gift{gift}, participants)

	assertPrizeAssignments(t, distribution, map[uint][]prizeAssignmentExpectation{
		3: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
	})
}

func TestPrizeDistributionFallsBackToNearestFreeRankInsideGroup(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(3)
	gift := prizeDistributionApprovedGift(100)
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{10})

	distribution := h.distributePrizes(results, []*entity.Gift{gift}, participants)

	assertPrizeAssignments(t, distribution, map[uint][]prizeAssignmentExpectation{
		3: {{giftID: 100, ruleType: "places", targetRank: 10, assignedRank: 3, fallback: true}},
	})
}

func TestPrizeDistributionFallbackSkipsParticipantAlreadyAssignedSameGift(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(2)
	gift := prizeDistributionApprovedGift(100)
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{2, 3})

	distribution := h.distributePrizes(results, []*entity.Gift{gift}, participants)

	assertPrizeAssignments(t, distribution, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 100, ruleType: "places", targetRank: 3, assignedRank: 1, fallback: true}},
		2: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
	})
}

func TestPrizeDistributionFallbackDoesNotLeaveEligibleGroupOrDuplicateGiftParticipant(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
		prizeDistributionParticipantWithProfile(1, valueobject.GenderFemale, valueobject.BikeTypeGravel),
		prizeDistributionParticipantWithProfile(2, valueobject.GenderMale, valueobject.BikeTypeGravel),
	})
	gift := prizeDistributionApprovedGift(100)
	gift.GenderFilter = "female"
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1, 2})

	distribution := h.distributePrizes(results, []*entity.Gift{gift}, participants)

	assertPrizeAssignments(t, distribution, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 100, ruleType: "places", targetRank: 1, assignedRank: 1}},
		2: nil,
	})
}

func TestPrizeDistributionCoversGiftVariantMatrix(t *testing.T) {
	tests := []struct {
		name      string
		configure func(results []*repository.ResultWithPlace, gift *entity.Gift)
		want      map[uint][]prizeAssignmentExpectation
	}{
		{
			name: "approved generic no criteria no rule",
			want: map[uint][]prizeAssignmentExpectation{
				1: {{giftID: 100, ruleType: "none", assignedRank: 1}},
			},
		},
		{
			name: "approved criteria only",
			configure: func(results []*repository.ResultWithPlace, gift *entity.Gift) {
				setPrizeResultCriteria(results, 2, 1)
				gift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
			},
			want: map[uint][]prizeAssignmentExpectation{
				2: {{giftID: 100, ruleType: "none", assignedRank: 1}},
			},
		},
		{
			name: "approved legacy place only",
			configure: func(_ []*repository.ResultWithPlace, gift *entity.Gift) {
				place := 2
				gift.Place = &place
			},
			want: map[uint][]prizeAssignmentExpectation{
				2: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
			},
		},
		{
			name: "approved explicit places",
			configure: func(_ []*repository.ResultWithPlace, gift *entity.Gift) {
				gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{2})
			},
			want: map[uint][]prizeAssignmentExpectation{
				2: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
			},
		},
		{
			name: "approved last n",
			configure: func(_ []*repository.ResultWithPlace, gift *entity.Gift) {
				gift.PlaceRule = mustGiftPlaceRuleLastN(t, 2)
			},
			want: map[uint][]prizeAssignmentExpectation{
				3: {{giftID: 100, ruleType: "last_n", targetRank: 3, assignedRank: 3}},
				4: {{giftID: 100, ruleType: "last_n", targetRank: 4, assignedRank: 4}},
			},
		},
		{
			name: "approved criteria plus places",
			configure: func(results []*repository.ResultWithPlace, gift *entity.Gift) {
				setPrizeResultCriteria(results, 3, 1)
				gift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
				gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1})
			},
			want: map[uint][]prizeAssignmentExpectation{
				3: {{giftID: 100, ruleType: "places", targetRank: 1, assignedRank: 1}},
			},
		},
		{
			name: "approved criteria plus last n",
			configure: func(results []*repository.ResultWithPlace, gift *entity.Gift) {
				setPrizeResultCriteria(results, 2, 1)
				setPrizeResultCriteria(results, 4, 1)
				gift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
				gift.PlaceRule = mustGiftPlaceRuleLastN(t, 1)
			},
			want: map[uint][]prizeAssignmentExpectation{
				4: {{giftID: 100, ruleType: "last_n", targetRank: 2, assignedRank: 2}},
			},
		},
		{
			name: "pending gift ignored",
			configure: func(_ []*repository.ResultWithPlace, gift *entity.Gift) {
				gift.ReviewStatus = entity.GiftReviewStatusPendingReview
			},
			want: map[uint][]prizeAssignmentExpectation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &GetPrizeDistributionHandler{}
			results, participants := prizeDistributionRankedScenario(4)
			gift := prizeDistributionApprovedGift(100)
			if tt.configure != nil {
				tt.configure(results, gift)
			}

			output := h.distributePrizeSlots(results, []*entity.Gift{gift}, participants)

			assertOnlyPrizeAssignments(t, output.Results, tt.want)
			if len(output.UnassignedSlots) != 0 {
				t.Fatalf("unexpected unassigned slots: %+v", output.UnassignedSlots)
			}
		})
	}
}

func TestPrizeDistributionCoversFilterScopeMatrix(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(gift *entity.Gift)
		want       map[uint][]prizeAssignmentExpectation
		targetRank int
	}{
		{
			name: "all gender all bike",
			want: map[uint][]prizeAssignmentExpectation{
				2: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
			},
			targetRank: 2,
		},
		{
			name: "gender only",
			configure: func(gift *entity.Gift) {
				gift.GenderFilter = "female"
			},
			want: map[uint][]prizeAssignmentExpectation{
				4: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
			},
			targetRank: 2,
		},
		{
			name: "bike only",
			configure: func(gift *entity.Gift) {
				gift.BikeTypeFilter = "mtb"
			},
			want: map[uint][]prizeAssignmentExpectation{
				4: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
			},
			targetRank: 2,
		},
		{
			name: "gender and bike",
			configure: func(gift *entity.Gift) {
				gift.GenderFilter = "female"
				gift.BikeTypeFilter = "mtb"
			},
			want: map[uint][]prizeAssignmentExpectation{
				4: {{giftID: 100, ruleType: "places", targetRank: 1, assignedRank: 1}},
			},
			targetRank: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &GetPrizeDistributionHandler{}
			results, participants := prizeDistributionRankedScenarioWithParticipants([]*entity.Participant{
				prizeDistributionParticipantWithProfile(1, valueobject.GenderMale, valueobject.BikeTypeGravel),
				prizeDistributionParticipantWithProfile(2, valueobject.GenderFemale, valueobject.BikeTypeGravel),
				prizeDistributionParticipantWithProfile(3, valueobject.GenderMale, valueobject.BikeTypeMTB),
				prizeDistributionParticipantWithProfile(4, valueobject.GenderFemale, valueobject.BikeTypeMTB),
			})
			gift := prizeDistributionApprovedGift(100)
			gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{tt.targetRank})
			if tt.configure != nil {
				tt.configure(gift)
			}

			output := h.distributePrizeSlots(results, []*entity.Gift{gift}, participants)

			assertOnlyPrizeAssignments(t, output.Results, tt.want)
		})
	}
}

func TestPrizeDistributionRequiresAllGiftCriteria(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(3)
	setPrizeResultCriteria(results, 1, 1)
	setPrizeResultCriteria(results, 2, 1, 2)
	setPrizeResultCriteria(results, 3, 2)
	gift := prizeDistributionApprovedGift(100)
	gift.Criteria = []*entity.Criteria{
		prizeDistributionCriteria(1),
		prizeDistributionCriteria(2),
	}

	output := h.distributePrizeSlots(results, []*entity.Gift{gift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		2: {{giftID: 100, ruleType: "none", assignedRank: 1}},
	})
}

func TestPrizeDistributionKeepsPlaceSlotsWhenResultHasCriteriaGift(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(2)
	setPrizeResultCriteria(results, 1, 1)
	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	placeGift := prizeDistributionApprovedGift(20)
	placeGift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1})
	genericGift := prizeDistributionApprovedGift(30)

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift, placeGift, genericGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {
			{giftID: 10, ruleType: "none", assignedRank: 1},
			{giftID: 20, ruleType: "places", targetRank: 1, assignedRank: 1},
		},
		2: {{giftID: 30, ruleType: "none", assignedRank: 2}},
	})
}

func TestPrizeDistributionKeepsMultiplePlaceSlotsWhenTargetsHaveCriteriaGifts(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(4)
	setPrizeResultCriteria(results, 1, 1)
	setPrizeResultCriteria(results, 2, 2)
	criteriaGift1 := prizeDistributionApprovedGift(10)
	criteriaGift1.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	criteriaGift2 := prizeDistributionApprovedGift(20)
	criteriaGift2.Criteria = []*entity.Criteria{prizeDistributionCriteria(2)}
	placeGift := prizeDistributionApprovedGift(30)
	placeGift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1, 2, 3})

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift1, criteriaGift2, placeGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {
			{giftID: 10, ruleType: "none", assignedRank: 1},
			{giftID: 30, ruleType: "places", targetRank: 1, assignedRank: 1},
		},
		2: {
			{giftID: 20, ruleType: "none", assignedRank: 1},
			{giftID: 30, ruleType: "places", targetRank: 2, assignedRank: 2},
		},
		3: {{giftID: 30, ruleType: "places", targetRank: 3, assignedRank: 3}},
	})
}

func TestPrizeDistributionKeepsLegacyPlaceWhenResultHasCriteriaGift(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(2)
	setPrizeResultCriteria(results, 1, 1)
	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	legacyPlace := 1
	placeGift := prizeDistributionApprovedGift(20)
	placeGift.Place = &legacyPlace

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift, placeGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {
			{giftID: 10, ruleType: "none", assignedRank: 1},
			{giftID: 20, ruleType: "places", targetRank: 1, assignedRank: 1},
		},
	})
}

func TestPrizeDistributionKeepsLastNSlotsWhenTargetsHaveCriteriaGifts(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(4)
	setPrizeResultCriteria(results, 3, 1)
	setPrizeResultCriteria(results, 4, 2)
	criteriaGift1 := prizeDistributionApprovedGift(10)
	criteriaGift1.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	criteriaGift2 := prizeDistributionApprovedGift(20)
	criteriaGift2.Criteria = []*entity.Criteria{prizeDistributionCriteria(2)}
	lastNGift := prizeDistributionApprovedGift(30)
	lastNGift.PlaceRule = mustGiftPlaceRuleLastN(t, 2)

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift1, criteriaGift2, lastNGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		3: {
			{giftID: 10, ruleType: "none", assignedRank: 1},
			{giftID: 30, ruleType: "last_n", targetRank: 3, assignedRank: 3},
		},
		4: {
			{giftID: 20, ruleType: "none", assignedRank: 1},
			{giftID: 30, ruleType: "last_n", targetRank: 4, assignedRank: 4},
		},
	})
}

func TestPrizeDistributionPlaceFallbackCanAssignParticipantWithCriteriaGift(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(3)
	setPrizeResultCriteria(results, 3, 1)
	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	placeGift := prizeDistributionApprovedGift(20)
	placeGift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{5})

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift, placeGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		3: {
			{giftID: 10, ruleType: "none", assignedRank: 1},
			{giftID: 20, ruleType: "places", targetRank: 5, assignedRank: 3, fallback: true},
		},
	})
}

func TestPrizeDistributionNoPlaceGiftsStillSkipHigherPriorityParticipants(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(3)
	setPrizeResultCriteria(results, 1, 1)
	setPrizeResultCriteria(results, 2, 2)
	criteriaGift1 := prizeDistributionApprovedGift(10)
	criteriaGift1.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	criteriaGift2 := prizeDistributionApprovedGift(20)
	criteriaGift2.Criteria = []*entity.Criteria{prizeDistributionCriteria(2)}
	genericGift1 := prizeDistributionApprovedGift(30)
	genericGift2 := prizeDistributionApprovedGift(40)

	output := h.distributePrizeSlots(results, []*entity.Gift{criteriaGift1, criteriaGift2, genericGift1, genericGift2}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 10, ruleType: "none", assignedRank: 1}},
		2: {{giftID: 20, ruleType: "none", assignedRank: 1}},
		3: {
			{giftID: 30, ruleType: "none", assignedRank: 3},
			{giftID: 40, ruleType: "none", assignedRank: 3},
		},
	})
}

func TestPrizeDistributionLastNLargerThanGroupAssignsAllEligible(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(2)
	gift := prizeDistributionApprovedGift(100)
	gift.PlaceRule = mustGiftPlaceRuleLastN(t, 5)

	output := h.distributePrizeSlots(results, []*entity.Gift{gift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 100, ruleType: "last_n", targetRank: 1, assignedRank: 1}},
		2: {{giftID: 100, ruleType: "last_n", targetRank: 2, assignedRank: 2}},
	})
}

func TestPrizeDistributionPlaceSlotKeepsTargetWhenParticipantHasCriteriaGift(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(5)
	setPrizeResultCriteria(results, 3, 1)
	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	placeGift := prizeDistributionApprovedGift(20)
	placeGift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{3})

	output := h.distributePrizeSlots(results, []*entity.Gift{placeGift, criteriaGift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		3: {
			{giftID: 10, ruleType: "none", assignedRank: 1},
			{giftID: 20, ruleType: "places", targetRank: 3, assignedRank: 3},
		},
	})
}

func TestPrizeDistributionReportsUnassignedSlotWhenNoFreeEligibleCandidate(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	results, participants := prizeDistributionRankedScenario(1)
	gift := prizeDistributionApprovedGift(100)
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{1, 2})

	output := h.distributePrizeSlots(results, []*entity.Gift{gift}, participants)

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 100, ruleType: "places", targetRank: 1, assignedRank: 1}},
	})
	assertPrizeUnassignedSlots(t, output.UnassignedSlots, []prizeUnassignedExpectation{
		{giftID: 100, ruleType: "places", targetRank: 2, reason: "target_out_of_range"},
	})
}

func TestPrizeDistributionNormalizesResultAndSlotOrdering(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	elapsedFast := 90
	elapsedTie := 100
	participant1 := prizeDistributionParticipantWithID(1)
	participant2 := prizeDistributionParticipantWithID(2)
	participant3 := prizeDistributionParticipantWithID(3)
	results := []*repository.ResultWithPlace{
		{
			Result: &entity.Result{
				ID:             30,
				ParticipantID:  3,
				ElapsedTimeSec: &elapsedTie,
			},
			PlaceAbsolute:     3,
			PlaceByGender:     3,
			PlaceByGenderBike: 3,
		},
		{
			Result: &entity.Result{
				ID:             10,
				ParticipantID:  1,
				ElapsedTimeSec: &elapsedFast,
			},
			PlaceAbsolute:     1,
			PlaceByGender:     1,
			PlaceByGenderBike: 1,
		},
		{
			Result: &entity.Result{
				ID:             20,
				ParticipantID:  2,
				ElapsedTimeSec: &elapsedTie,
			},
			PlaceAbsolute:     2,
			PlaceByGender:     2,
			PlaceByGenderBike: 2,
		},
	}
	gift := prizeDistributionApprovedGift(100)
	gift.PlaceRule = mustGiftPlaceRulePlaces(t, []int{3, 1, 2})

	output := h.distributePrizeSlots(results, []*entity.Gift{gift}, map[uint]*entity.Participant{
		1: participant1,
		2: participant2,
		3: participant3,
	})

	assertOnlyPrizeAssignments(t, output.Results, map[uint][]prizeAssignmentExpectation{
		1: {{giftID: 100, ruleType: "places", targetRank: 1, assignedRank: 1}},
		2: {{giftID: 100, ruleType: "places", targetRank: 2, assignedRank: 2}},
		3: {{giftID: 100, ruleType: "places", targetRank: 3, assignedRank: 3}},
	})
}

type prizeAssignmentExpectation struct {
	giftID       uint
	ruleType     string
	targetRank   int
	assignedRank int
	fallback     bool
}

type prizeUnassignedExpectation struct {
	giftID     uint
	ruleType   string
	targetRank int
	reason     string
}

func assertPrizeAssignments(t *testing.T, distribution []*PrizeDistributionResult, want map[uint][]prizeAssignmentExpectation) {
	t.Helper()

	gotByParticipant := make(map[uint][]*PrizeGiftAssignment, len(distribution))
	for _, result := range distribution {
		gotByParticipant[result.ParticipantID] = result.MatchedGiftAssignments
	}

	for participantID, wantAssignments := range want {
		gotAssignments := gotByParticipant[participantID]
		if len(gotAssignments) != len(wantAssignments) {
			t.Fatalf("participant %d assignment count mismatch: got %s, want %+v", participantID, compactPrizeAssignments(gotAssignments), wantAssignments)
		}
		for i, wantAssignment := range wantAssignments {
			gotAssignment := gotAssignments[i]
			if gotAssignment.Gift.ID != wantAssignment.giftID ||
				gotAssignment.RuleType != wantAssignment.ruleType ||
				gotAssignment.TargetRank != wantAssignment.targetRank ||
				gotAssignment.AssignedRank != wantAssignment.assignedRank ||
				gotAssignment.IsFallback != wantAssignment.fallback {
				t.Fatalf("participant %d assignment %d mismatch: got %s, want %+v", participantID, i, compactPrizeAssignment(gotAssignment), wantAssignment)
			}
		}
	}
}

func assertOnlyPrizeAssignments(t *testing.T, distribution []*PrizeDistributionResult, want map[uint][]prizeAssignmentExpectation) {
	t.Helper()

	gotByParticipant := make(map[uint][]*PrizeGiftAssignment, len(distribution))
	for _, result := range distribution {
		gotByParticipant[result.ParticipantID] = result.MatchedGiftAssignments
	}

	for _, result := range distribution {
		wantAssignments, ok := want[result.ParticipantID]
		if !ok && len(result.MatchedGiftAssignments) > 0 {
			t.Fatalf("participant %d has unexpected assignments: %s", result.ParticipantID, compactPrizeAssignments(result.MatchedGiftAssignments))
		}
		if !ok {
			continue
		}
		assertPrizeAssignmentList(t, result.ParticipantID, result.MatchedGiftAssignments, wantAssignments)
	}

	for participantID, wantAssignments := range want {
		if _, ok := gotByParticipant[participantID]; !ok && len(wantAssignments) > 0 {
			t.Fatalf("participant %d missing from distribution, want %+v", participantID, wantAssignments)
		}
	}
}

func assertPrizeAssignmentList(
	t *testing.T,
	participantID uint,
	gotAssignments []*PrizeGiftAssignment,
	wantAssignments []prizeAssignmentExpectation,
) {
	t.Helper()

	if len(gotAssignments) != len(wantAssignments) {
		t.Fatalf("participant %d assignment count mismatch: got %s, want %+v", participantID, compactPrizeAssignments(gotAssignments), wantAssignments)
	}
	for i, wantAssignment := range wantAssignments {
		gotAssignment := gotAssignments[i]
		if gotAssignment.Gift.ID != wantAssignment.giftID ||
			gotAssignment.RuleType != wantAssignment.ruleType ||
			gotAssignment.TargetRank != wantAssignment.targetRank ||
			gotAssignment.AssignedRank != wantAssignment.assignedRank ||
			gotAssignment.IsFallback != wantAssignment.fallback {
			t.Fatalf("participant %d assignment %d mismatch: got %s, want %+v", participantID, i, compactPrizeAssignment(gotAssignment), wantAssignment)
		}
	}
}

func assertPrizeUnassignedSlots(t *testing.T, got []*UnassignedPrizeSlot, want []prizeUnassignedExpectation) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unassigned slots count mismatch: got %+v, want %+v", got, want)
	}
	for i, wantSlot := range want {
		gotSlot := got[i]
		if gotSlot.GiftID != wantSlot.giftID ||
			gotSlot.RuleType != wantSlot.ruleType ||
			gotSlot.TargetRank != wantSlot.targetRank ||
			gotSlot.Reason != wantSlot.reason {
			t.Fatalf("unassigned slot %d mismatch: got %+v, want %+v", i, gotSlot, wantSlot)
		}
	}
}

func compactPrizeAssignments(assignments []*PrizeGiftAssignment) []string {
	rows := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		rows = append(rows, compactPrizeAssignment(assignment))
	}
	return rows
}

func compactPrizeAssignment(assignment *PrizeGiftAssignment) string {
	if assignment == nil || assignment.Gift == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"gift=%d rule=%s target=%d assigned=%d fallback=%t reason=%s",
		assignment.Gift.ID,
		assignment.RuleType,
		assignment.TargetRank,
		assignment.AssignedRank,
		assignment.IsFallback,
		assignment.MatchReason,
	)
}

func prizeDistributionRankedScenario(count int) ([]*repository.ResultWithPlace, map[uint]*entity.Participant) {
	participants := make([]*entity.Participant, 0, count)
	for i := 1; i <= count; i++ {
		participants = append(participants, prizeDistributionParticipantWithID(uint(i)))
	}
	return prizeDistributionRankedScenarioWithParticipants(participants)
}

func prizeDistributionRankedScenarioWithParticipants(participants []*entity.Participant) ([]*repository.ResultWithPlace, map[uint]*entity.Participant) {
	results := make([]*repository.ResultWithPlace, 0, len(participants))
	participantMap := make(map[uint]*entity.Participant, len(participants))
	for i, participant := range participants {
		rank := i + 1
		elapsed := 1000 + rank
		results = append(results, &repository.ResultWithPlace{
			Result: &entity.Result{
				ID:             uint(rank),
				ParticipantID:  participant.ID,
				ElapsedTimeSec: &elapsed,
			},
			ParticipantGender:   string(participant.Gender),
			ParticipantBikeType: string(participant.BikeType),
			PlaceAbsolute:       rank,
			PlaceByGender:       rank,
			PlaceByGenderBike:   rank,
		})
		participantMap[participant.ID] = participant
	}
	return results, participantMap
}

func prizeDistributionParticipantWithProfile(id uint, gender valueobject.Gender, bikeType valueobject.BikeType) *entity.Participant {
	participant := prizeDistributionParticipantWithID(id)
	participant.Gender = gender
	participant.BikeType = bikeType
	return participant
}

func setPrizeResultCriteria(results []*repository.ResultWithPlace, participantID uint, criteriaIDs ...uint) {
	criteria := make([]*entity.Criteria, 0, len(criteriaIDs))
	for _, criteriaID := range criteriaIDs {
		criteria = append(criteria, prizeDistributionCriteria(criteriaID))
	}
	for _, result := range results {
		if result.Result.ParticipantID == participantID {
			result.Result.Criteria = criteria
			return
		}
	}
}

func mustGiftPlaceRulePlaces(t *testing.T, places []int) valueobject.GiftPlaceRule {
	t.Helper()

	rule, err := valueobject.NewGiftPlaceRulePlaces(places)
	if err != nil {
		t.Fatalf("places rule must be valid for test setup: %v", err)
	}
	return rule
}

func mustGiftPlaceRuleLastN(t *testing.T, count int) valueobject.GiftPlaceRule {
	t.Helper()

	rule, err := valueobject.NewGiftPlaceRuleLastN(count)
	if err != nil {
		t.Fatalf("last_n rule must be valid for test setup: %v", err)
	}
	return rule
}
