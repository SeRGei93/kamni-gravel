package query

import (
	"fmt"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

type prizeDistributionE2EGiftVariant string

const (
	prizeDistributionE2EVariantGeneric        prizeDistributionE2EGiftVariant = "generic"
	prizeDistributionE2EVariantLegacyPlace    prizeDistributionE2EGiftVariant = "legacy_place"
	prizeDistributionE2EVariantPlaces         prizeDistributionE2EGiftVariant = "places"
	prizeDistributionE2EVariantLastN          prizeDistributionE2EGiftVariant = "last_n"
	prizeDistributionE2EVariantCriteria       prizeDistributionE2EGiftVariant = "criteria"
	prizeDistributionE2EVariantMultiCriteria  prizeDistributionE2EGiftVariant = "multi_criteria"
	prizeDistributionE2EVariantCriteriaPlaces prizeDistributionE2EGiftVariant = "criteria_places"
	prizeDistributionE2EVariantCriteriaLastN  prizeDistributionE2EGiftVariant = "criteria_last_n"
	prizeDistributionE2EVariantPendingGeneric prizeDistributionE2EGiftVariant = "pending_generic"
	prizeDistributionE2EVariantPendingPlaces  prizeDistributionE2EGiftVariant = "pending_places"
	prizeDistributionE2EVariantFallback       prizeDistributionE2EGiftVariant = "fallback"
	prizeDistributionE2EVariantUnassigned     prizeDistributionE2EGiftVariant = "unassigned"
)

func TestPrizeDistributionE2EWithSyntheticGiftMatrix(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	criteria := prizeDistributionE2ECriteria()
	results, participants := prizeDistributionE2EParticipants(t, 200, criteria)
	gifts, giftVariants := prizeDistributionE2EGiftMatrix(t, criteria)

	output := h.distributePrizeSlots(results, gifts, participants)

	if len(output.Results) != 200 {
		t.Fatalf("distribution rows = %d, want 200", len(output.Results))
	}

	assignmentsByVariant := prizeDistributionE2EAssertAssignmentInvariants(t, output, participants, results, giftVariants)
	prizeDistributionE2EAssertVariantCoverage(t, output, assignmentsByVariant, giftVariants)
}

func prizeDistributionE2ECriteria() map[string]*entity.Criteria {
	return map[string]*entity.Criteria{
		"speed": {
			ID:           1,
			Name:         "E2E скорость",
			CriteriaType: valueobject.CriteriaTypeSpeed,
		},
		"photo": {
			ID:           2,
			Name:         "E2E фото",
			CriteriaType: valueobject.CriteriaTypePhoto,
		},
		"beer": {
			ID:           3,
			Name:         "E2E пиво",
			CriteriaType: valueobject.CriteriaTypeBeer,
		},
		"style": {
			ID:           4,
			Name:         "E2E стиль",
			CriteriaType: valueobject.CriteriaTypeCustom,
		},
		"fallback": {
			ID:           5,
			Name:         "E2E fallback",
			CriteriaType: valueobject.CriteriaTypeCustom,
		},
		"never": {
			ID:           6,
			Name:         "E2E недостижимый критерий",
			CriteriaType: valueobject.CriteriaTypeCustom,
		},
	}
}

func prizeDistributionE2EParticipants(
	t *testing.T,
	count int,
	criteria map[string]*entity.Criteria,
) ([]*repository.ResultWithPlace, map[uint]*entity.Participant) {
	t.Helper()

	if count != 200 {
		t.Fatalf("this e2e fixture expects 200 participants, got %d", count)
	}

	bikeTypes := []valueobject.BikeType{
		valueobject.BikeTypeGravel,
		valueobject.BikeTypeMTB,
		valueobject.BikeTypeRoad,
		valueobject.BikeTypeFixedGear,
		valueobject.BikeTypeTandem,
	}
	genderRanks := map[valueobject.Gender]int{}
	genderBikeRanks := map[string]int{}
	results := make([]*repository.ResultWithPlace, 0, count)
	participants := make(map[uint]*entity.Participant, count)
	baseTime := time.Date(2026, time.May, 28, 12, 0, 0, 0, time.UTC)

	for i := 1; i <= count; i++ {
		id := uint(i)
		gender := valueobject.GenderMale
		if i%2 == 0 {
			gender = valueobject.GenderFemale
		}
		bikeType := bikeTypes[(i-1)%len(bikeTypes)]
		genderRanks[gender]++
		genderBikeKey := prizeDistributionE2EGenderBikeKey(gender, bikeType)
		genderBikeRanks[genderBikeKey]++
		groupRank := genderBikeRanks[genderBikeKey]

		participant := &entity.Participant{
			ID:           id,
			UserID:       int64(900000 + i),
			EventID:      77,
			BikeType:     bikeType,
			Gender:       gender,
			RegisteredAt: baseTime.Add(time.Duration(i) * time.Second),
			User: &entity.User{
				ID:        int64(900000 + i),
				Username:  fmt.Sprintf("e2e_rider_%03d", i),
				FirstName: fmt.Sprintf("Rider %03d", i),
			},
		}
		participants[id] = participant

		elapsed := 3600 + i
		results = append(results, &repository.ResultWithPlace{
			Result: &entity.Result{
				ID:             id,
				ParticipantID:  id,
				ElapsedTimeSec: &elapsed,
				MovingTimeSec:  &elapsed,
				IsCurrent:      true,
				SubmittedAt:    baseTime.Add(time.Duration(i) * time.Minute),
				Criteria:       prizeDistributionE2EResultCriteria(id, groupRank, criteria),
			},
			ParticipantGender:   string(gender),
			ParticipantBikeType: string(bikeType),
			PlaceAbsolute:       i,
			PlaceByGender:       genderRanks[gender],
			PlaceByGenderBike:   groupRank,
		})
	}

	return results, participants
}

func prizeDistributionE2EResultCriteria(
	participantID uint,
	groupRank int,
	criteria map[string]*entity.Criteria,
) []*entity.Criteria {
	resultCriteria := make([]*entity.Criteria, 0, 5)
	if groupRank%2 == 1 {
		resultCriteria = append(resultCriteria, criteria["speed"])
	}
	if groupRank%3 == 1 {
		resultCriteria = append(resultCriteria, criteria["photo"])
	}
	if groupRank%4 == 1 {
		resultCriteria = append(resultCriteria, criteria["beer"])
	}
	if groupRank%5 == 1 {
		resultCriteria = append(resultCriteria, criteria["style"])
	}
	if participantID == 200 {
		resultCriteria = append(resultCriteria, criteria["fallback"])
	}
	return resultCriteria
}

func prizeDistributionE2EGiftMatrix(
	t *testing.T,
	criteria map[string]*entity.Criteria,
) ([]*entity.Gift, map[uint]prizeDistributionE2EGiftVariant) {
	t.Helper()

	genderFilters := []string{"all", "male", "female"}
	bikeTypeFilters := []string{"all", "gravel", "mtb", "road", "single_speed", "tandem"}
	gifts := make([]*entity.Gift, 0, len(genderFilters)*len(bikeTypeFilters)*10+2)
	variants := make(map[uint]prizeDistributionE2EGiftVariant)

	for _, genderFilter := range genderFilters {
		for _, bikeTypeFilter := range bikeTypeFilters {
			gifts = append(gifts,
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantCriteriaPlaces, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, []*entity.Criteria{criteria["photo"]}, nil, []int{1}, 0),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantCriteriaLastN, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, []*entity.Criteria{criteria["beer"]}, nil, nil, 1),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantCriteria, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, []*entity.Criteria{criteria["speed"]}, nil, nil, 0),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantMultiCriteria, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, []*entity.Criteria{criteria["speed"], criteria["photo"]}, nil, nil, 0),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantLegacyPlace, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, nil, prizeDistributionE2EIntPtr(2), nil, 0),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantPlaces, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, nil, nil, []int{3}, 0),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantLastN, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, nil, nil, nil, 2),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantGeneric, genderFilter, bikeTypeFilter, entity.GiftReviewStatusApproved, nil, nil, nil, 0),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantPendingGeneric, genderFilter, bikeTypeFilter, entity.GiftReviewStatusPendingReview, nil, nil, nil, 0),
				prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantPendingPlaces, genderFilter, bikeTypeFilter, entity.GiftReviewStatusPendingReview, []*entity.Criteria{criteria["photo"]}, nil, []int{1}, 0),
			)
		}
	}

	gifts = append(gifts,
		prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantFallback, "all", "all", entity.GiftReviewStatusApproved, []*entity.Criteria{criteria["fallback"]}, nil, []int{999}, 0),
		prizeDistributionE2EGift(t, variants, prizeDistributionE2EVariantUnassigned, "all", "all", entity.GiftReviewStatusApproved, []*entity.Criteria{criteria["never"]}, nil, []int{1}, 0),
	)

	return gifts, variants
}

func prizeDistributionE2EGift(
	t *testing.T,
	variants map[uint]prizeDistributionE2EGiftVariant,
	variant prizeDistributionE2EGiftVariant,
	genderFilter string,
	bikeTypeFilter string,
	reviewStatus entity.GiftReviewStatus,
	criteria []*entity.Criteria,
	legacyPlace *int,
	places []int,
	lastCount int,
) *entity.Gift {
	t.Helper()

	id := uint(len(variants) + 1)
	gift := &entity.Gift{
		ID:             id,
		UserID:         int64(800000 + id),
		EventID:        77,
		Description:    fmt.Sprintf("E2E %s %s/%s", variant, genderFilter, bikeTypeFilter),
		GenderFilter:   genderFilter,
		BikeTypeFilter: bikeTypeFilter,
		ReviewStatus:   reviewStatus,
		Place:          legacyPlace,
		Criteria:       criteria,
		CreatedAt:      time.Date(2026, time.May, 28, 13, 0, 0, 0, time.UTC).Add(time.Duration(id) * time.Second),
	}
	switch {
	case len(places) > 0:
		gift.PlaceRule = mustGiftPlaceRulePlaces(t, places)
	case lastCount > 0:
		gift.PlaceRule = mustGiftPlaceRuleLastN(t, lastCount)
	}
	variants[id] = variant
	return gift
}

func prizeDistributionE2EAssertAssignmentInvariants(
	t *testing.T,
	output *PrizeDistributionOutput,
	participants map[uint]*entity.Participant,
	results []*repository.ResultWithPlace,
	giftVariants map[uint]prizeDistributionE2EGiftVariant,
) map[prizeDistributionE2EGiftVariant]int {
	t.Helper()

	resultCriteriaByParticipant := make(map[uint][]*entity.Criteria, len(results))
	for _, rwp := range results {
		resultCriteriaByParticipant[rwp.Result.ParticipantID] = rwp.Result.Criteria
	}

	assignmentsByVariant := make(map[prizeDistributionE2EGiftVariant]int)
	assignmentsByGift := make(map[uint]int)
	seenGiftParticipant := make(map[string]bool)

	for _, row := range output.Results {
		participant := participants[row.ParticipantID]
		if participant == nil {
			t.Fatalf("participant %d is missing from fixture", row.ParticipantID)
		}

		var rowPriority *giftMatchPriority
		for _, assignment := range row.MatchedGiftAssignments {
			if assignment.ParticipantID != row.ParticipantID {
				t.Fatalf("assignment participant_id = %d, row participant_id = %d", assignment.ParticipantID, row.ParticipantID)
			}
			if assignment.Gift == nil {
				t.Fatalf("participant %d has assignment without gift", row.ParticipantID)
			}

			gift := assignment.Gift
			variant, ok := giftVariants[gift.ID]
			if !ok {
				t.Fatalf("gift %d is missing variant metadata", gift.ID)
			}
			if gift.ReviewStatus != entity.GiftReviewStatusApproved {
				t.Fatalf("pending gift was assigned: participant=%d gift=%d variant=%s", row.ParticipantID, gift.ID, variant)
			}
			if !giftMatchesParticipantFilters(gift, participant) {
				t.Fatalf("gift filters do not match participant: participant=%d gift=%d gender_filter=%s bike_filter=%s gender=%s bike=%s", row.ParticipantID, gift.ID, gift.GenderFilter, gift.BikeTypeFilter, participant.Gender, participant.BikeType)
			}
			if !allPrizeCriteriaMatch(gift.Criteria, resultCriteriaByParticipant[row.ParticipantID]) {
				t.Fatalf("gift criteria do not match participant result: participant=%d gift=%d variant=%s", row.ParticipantID, gift.ID, variant)
			}
			key := fmt.Sprintf("%d:%d", gift.ID, row.ParticipantID)
			if seenGiftParticipant[key] {
				t.Fatalf("gift %d was assigned to participant %d more than once", gift.ID, row.ParticipantID)
			}
			seenGiftParticipant[key] = true

			priority, ok := classifyGiftForSlotEngine(gift)
			if !ok {
				t.Fatalf("gift %d could not be classified", gift.ID)
			}
			if rowPriority == nil {
				rowPriority = &priority
			} else if *rowPriority != priority {
				t.Fatalf("participant %d received mixed-priority assignments", row.ParticipantID)
			}

			assignmentsByGift[gift.ID]++
			assignmentsByVariant[variant]++
		}
	}

	for giftID, variant := range giftVariants {
		switch variant {
		case prizeDistributionE2EVariantPendingGeneric, prizeDistributionE2EVariantPendingPlaces, prizeDistributionE2EVariantUnassigned:
			if assignmentsByGift[giftID] != 0 {
				t.Fatalf("gift %d variant %s should not be assigned", giftID, variant)
			}
		default:
			if assignmentsByGift[giftID] == 0 {
				t.Fatalf("approved gift %d variant %s was not assigned", giftID, variant)
			}
		}
	}

	return assignmentsByVariant
}

func prizeDistributionE2EAssertVariantCoverage(
	t *testing.T,
	output *PrizeDistributionOutput,
	assignmentsByVariant map[prizeDistributionE2EGiftVariant]int,
	giftVariants map[uint]prizeDistributionE2EGiftVariant,
) {
	t.Helper()

	for _, variant := range []prizeDistributionE2EGiftVariant{
		prizeDistributionE2EVariantGeneric,
		prizeDistributionE2EVariantLegacyPlace,
		prizeDistributionE2EVariantPlaces,
		prizeDistributionE2EVariantLastN,
		prizeDistributionE2EVariantCriteria,
		prizeDistributionE2EVariantMultiCriteria,
		prizeDistributionE2EVariantCriteriaPlaces,
		prizeDistributionE2EVariantCriteriaLastN,
		prizeDistributionE2EVariantFallback,
	} {
		if assignmentsByVariant[variant] == 0 {
			t.Fatalf("variant %s has no assignments", variant)
		}
	}

	if assignmentsByVariant[prizeDistributionE2EVariantPendingGeneric] != 0 ||
		assignmentsByVariant[prizeDistributionE2EVariantPendingPlaces] != 0 {
		t.Fatalf("pending variants should not be assigned: %+v", assignmentsByVariant)
	}

	hasFallback := false
	for _, row := range output.Results {
		for _, assignment := range row.MatchedGiftAssignments {
			if giftVariants[assignment.Gift.ID] == prizeDistributionE2EVariantFallback && assignment.IsFallback && assignment.FallbackReason == "target_out_of_range" {
				hasFallback = true
			}
		}
	}
	if !hasFallback {
		t.Fatalf("fallback gift was not assigned through target_out_of_range fallback")
	}

	hasUnassigned := false
	for _, slot := range output.UnassignedSlots {
		if giftVariants[slot.GiftID] == prizeDistributionE2EVariantUnassigned && slot.Reason == "empty_eligible_group" {
			hasUnassigned = true
		}
	}
	if !hasUnassigned {
		t.Fatalf("unassigned gift did not report empty_eligible_group, got %+v", output.UnassignedSlots)
	}
}

func prizeDistributionE2EGenderBikeKey(gender valueobject.Gender, bikeType valueobject.BikeType) string {
	return string(gender) + ":" + string(bikeType)
}

func prizeDistributionE2EIntPtr(value int) *int {
	return &value
}
