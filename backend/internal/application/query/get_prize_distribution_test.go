package query

import (
	"fmt"
	"strings"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestPrizeDistributionIgnoresUnapprovedGifts(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	participant := prizeDistributionParticipant()
	rwp := prizeDistributionResultWithPlace([]*entity.Criteria{prizeDistributionCriteria(1)})

	result := h.findAllMatchingGifts(rwp, participant, []*entity.Gift{
		{
			ID:             1,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusPendingReview,
			Criteria:       []*entity.Criteria{prizeDistributionCriteria(1)},
		},
	}, map[uint]bool{})

	if len(result) != 0 {
		t.Fatalf("pending gift should not match, got %d matches", len(result))
	}
}

func TestPrizeDistributionUsesCriteriaBeforePlaceAndGeneric(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	participant := prizeDistributionParticipant()
	rwp := prizeDistributionResultWithPlace([]*entity.Criteria{prizeDistributionCriteria(1)})
	place := 1

	result := h.findAllMatchingGifts(rwp, participant, []*entity.Gift{
		{
			ID:             30,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
		},
		{
			ID:             20,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Place:          &place,
		},
		{
			ID:             10,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Criteria:       []*entity.Criteria{prizeDistributionCriteria(1)},
		},
	}, map[uint]bool{})

	if len(result) != 1 {
		t.Fatalf("criteria priority should return one best-priority gift, got %d", len(result))
	}
	if result[0].ID != 10 {
		t.Fatalf("criteria gift should win over place/generic, got gift %d", result[0].ID)
	}
}

func TestPrizeDistributionUsesPlaceAsCriteriaTieBreaker(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	participant := prizeDistributionParticipant()
	rwp := prizeDistributionResultWithPlace([]*entity.Criteria{prizeDistributionCriteria(1)})
	place := 1

	result := h.findAllMatchingGifts(rwp, participant, []*entity.Gift{
		{
			ID:             20,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Criteria:       []*entity.Criteria{prizeDistributionCriteria(1)},
		},
		{
			ID:             10,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Place:          &place,
			Criteria:       []*entity.Criteria{prizeDistributionCriteria(1)},
		},
	}, map[uint]bool{})

	if len(result) != 1 {
		t.Fatalf("criteria+place tie breaker should return one best-priority gift, got %d", len(result))
	}
	if result[0].ID != 10 {
		t.Fatalf("criteria+place gift should win tie breaker, got gift %d", result[0].ID)
	}
}

func TestPrizeDistributionAllowsMultipleSamePriorityGifts(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	participant := prizeDistributionParticipant()
	rwp := prizeDistributionResultWithPlace([]*entity.Criteria{prizeDistributionCriteria(1)})

	result := h.findAllMatchingGifts(rwp, participant, []*entity.Gift{
		{
			ID:             20,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Criteria:       []*entity.Criteria{prizeDistributionCriteria(1)},
		},
		{
			ID:             10,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Criteria:       []*entity.Criteria{prizeDistributionCriteria(1)},
		},
	}, map[uint]bool{})

	assertPrizeGiftIDs(t, result, []uint{10, 20})
}

func TestPrizeDistributionDoesNotBundleLowerPriorityGifts(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	participant := prizeDistributionParticipant()
	rwp := prizeDistributionResultWithPlace([]*entity.Criteria{prizeDistributionCriteria(1)})
	place := 1

	result := h.findAllMatchingGifts(rwp, participant, []*entity.Gift{
		{
			ID:             10,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Criteria:       []*entity.Criteria{prizeDistributionCriteria(1)},
		},
		{
			ID:             20,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
			Place:          &place,
		},
		{
			ID:             30,
			GenderFilter:   "all",
			BikeTypeFilter: "all",
			ReviewStatus:   entity.GiftReviewStatusApproved,
		},
	}, map[uint]bool{})

	assertPrizeGiftIDs(t, result, []uint{10})
}

func TestPrizeDistributionNoPlaceGiftCascadesWhenFasterParticipantHasHigherPriority(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	participant1 := prizeDistributionParticipantWithID(1)
	participant2 := prizeDistributionParticipantWithID(2)
	criteriaGift := prizeDistributionApprovedGift(10)
	criteriaGift.Criteria = []*entity.Criteria{prizeDistributionCriteria(1)}
	genericGift := prizeDistributionApprovedGift(20)

	distribution := h.distributePrizes(
		[]*repository.ResultWithPlace{
			prizeDistributionResultWithParticipant(1, 1, 1, []*entity.Criteria{prizeDistributionCriteria(1)}),
			prizeDistributionResultWithParticipant(2, 2, 2, nil),
		},
		[]*entity.Gift{criteriaGift, genericGift},
		map[uint]*entity.Participant{
			participant1.ID: participant1,
			participant2.ID: participant2,
		},
	)

	assertDistributionGiftIDs(t, distribution, map[uint][]uint{
		1: {10},
		2: {20},
	})
}

func TestPrizeDistributionUsedGiftIsAssignedOnlyOnce(t *testing.T) {
	h := &GetPrizeDistributionHandler{}
	genericGift := prizeDistributionApprovedGift(10)

	distribution := h.distributePrizes(
		[]*repository.ResultWithPlace{
			prizeDistributionResultWithParticipant(1, 1, 1, nil),
			prizeDistributionResultWithParticipant(2, 2, 2, nil),
		},
		[]*entity.Gift{genericGift},
		map[uint]*entity.Participant{
			1: prizeDistributionParticipantWithID(1),
			2: prizeDistributionParticipantWithID(2),
		},
	)

	assertDistributionGiftIDs(t, distribution, map[uint][]uint{
		1: {10},
		2: nil,
	})
}

func prizeDistributionParticipant() *entity.Participant {
	return prizeDistributionParticipantWithID(1)
}

func prizeDistributionParticipantWithID(id uint) *entity.Participant {
	return &entity.Participant{
		ID:       id,
		UserID:   int64(100 + id),
		EventID:  77,
		Gender:   valueobject.GenderMale,
		BikeType: valueobject.BikeTypeGravel,
		User:     &entity.User{Username: "rider"},
	}
}

func prizeDistributionResultWithPlace(criteria []*entity.Criteria) *repository.ResultWithPlace {
	return prizeDistributionResultWithParticipant(1, 1, 1, criteria)
}

func prizeDistributionResultWithParticipant(participantID uint, resultID uint, place int, criteria []*entity.Criteria) *repository.ResultWithPlace {
	return &repository.ResultWithPlace{
		Result: &entity.Result{
			ID:            resultID,
			ParticipantID: participantID,
			Criteria:      criteria,
		},
		PlaceAbsolute:     place,
		PlaceByGender:     place,
		PlaceByGenderBike: place,
	}
}

func prizeDistributionCriteria(id uint) *entity.Criteria {
	return &entity.Criteria{ID: id, Name: "Speed"}
}

func prizeDistributionApprovedGift(id uint) *entity.Gift {
	return &entity.Gift{
		ID:             id,
		GenderFilter:   "all",
		BikeTypeFilter: "all",
		ReviewStatus:   entity.GiftReviewStatusApproved,
	}
}

func assertDistributionGiftIDs(t *testing.T, distribution []*PrizeDistributionResult, want map[uint][]uint) {
	t.Helper()

	got := make(map[uint][]uint, len(distribution))
	for _, result := range distribution {
		got[result.ParticipantID] = prizeGiftIDs(result.MatchedGifts)
	}

	for participantID, wantIDs := range want {
		gotIDs := got[participantID]
		if len(gotIDs) != len(wantIDs) {
			t.Fatalf("participant %d gifts mismatch: got %v, want %v; distribution=%s", participantID, gotIDs, wantIDs, compactPrizeDistribution(distribution))
		}
		for i := range wantIDs {
			if gotIDs[i] != wantIDs[i] {
				t.Fatalf("participant %d gifts mismatch: got %v, want %v; distribution=%s", participantID, gotIDs, wantIDs, compactPrizeDistribution(distribution))
			}
		}
	}
}

func assertPrizeGiftIDs(t *testing.T, gifts []*entity.Gift, want []uint) {
	t.Helper()

	got := prizeGiftIDs(gifts)
	if len(got) != len(want) {
		t.Fatalf("gift IDs mismatch: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("gift IDs mismatch: got %v, want %v", got, want)
		}
	}
}

func prizeGiftIDs(gifts []*entity.Gift) []uint {
	ids := make([]uint, 0, len(gifts))
	for _, gift := range gifts {
		ids = append(ids, gift.ID)
	}
	return ids
}

func compactPrizeDistribution(distribution []*PrizeDistributionResult) string {
	rows := make([]string, 0, len(distribution))
	for _, result := range distribution {
		rows = append(rows, fmt.Sprintf("participant=%d gifts=%v reason=%s", result.ParticipantID, prizeGiftIDs(result.MatchedGifts), result.MatchReason))
	}
	return strings.Join(rows, "; ")
}
