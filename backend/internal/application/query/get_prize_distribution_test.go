package query

import (
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

func prizeDistributionParticipant() *entity.Participant {
	return &entity.Participant{
		ID:       1,
		UserID:   123,
		EventID:  77,
		Gender:   valueobject.GenderMale,
		BikeType: valueobject.BikeTypeGravel,
		User:     &entity.User{Username: "rider"},
	}
}

func prizeDistributionResultWithPlace(criteria []*entity.Criteria) *repository.ResultWithPlace {
	return &repository.ResultWithPlace{
		Result: &entity.Result{
			ID:            1,
			ParticipantID: 1,
			Criteria:      criteria,
		},
		PlaceAbsolute: 1,
		PlaceByGender: 1,
	}
}

func prizeDistributionCriteria(id uint) *entity.Criteria {
	return &entity.Criteria{ID: id, Name: "Speed"}
}
