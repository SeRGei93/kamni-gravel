package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestPrizeDistributionHandlerFiltersAndRanksSelectedCohort(t *testing.T) {
	h := newPrizeDistributionHTTPTestHandler()

	tests := []struct {
		name              string
		target            string
		wantIDs           []uint
		wantDisplayPlaces []*int
		wantTotal         int
		wantPage          int
		wantPageSize      int
	}{
		{
			name:              "all filters include every automatic distribution row",
			target:            "/api/events/77/prize-distribution?gender=all&bike_type=all",
			wantIDs:           []uint{2, 1, 3, 4},
			wantDisplayPlaces: intPointers(1, 2, 3, 0),
			wantTotal:         4,
		},
		{
			name:              "empty filters keep the same rows",
			target:            "/api/events/77/prize-distribution",
			wantIDs:           []uint{2, 1, 3, 4},
			wantDisplayPlaces: intPointers(1, 2, 3, 0),
			wantTotal:         4,
		},
		{
			name:              "gender and bike type are intersected",
			target:            "/api/events/77/prize-distribution?gender=male&bike_type=gravel",
			wantIDs:           []uint{2, 1},
			wantDisplayPlaces: intPointers(1, 2),
			wantTotal:         2,
		},
		{
			name:              "match reason is applied after cohort rank",
			target:            "/api/events/77/prize-distribution?gender=male&bike_type=gravel&match_reason=no_match",
			wantIDs:           []uint{1},
			wantDisplayPlaces: intPointers(2),
			wantTotal:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPrizeDistribution(t, h, tt.target)
			if got.Total != tt.wantTotal {
				t.Fatalf("total = %d, want %d", got.Total, tt.wantTotal)
			}
			if got.Page != tt.wantPage || got.PageSize != tt.wantPageSize {
				t.Fatalf("page = %d/%d, want %d/%d", got.Page, got.PageSize, tt.wantPage, tt.wantPageSize)
			}
			if len(got.Distribution) != len(tt.wantIDs) {
				t.Fatalf("distribution length = %d, want %d", len(got.Distribution), len(tt.wantIDs))
			}

			for index, row := range got.Distribution {
				if row.ParticipantID != tt.wantIDs[index] {
					t.Fatalf("row %d participant_id = %d, want %d", index, row.ParticipantID, tt.wantIDs[index])
				}
				wantPlace := tt.wantDisplayPlaces[index]
				if wantPlace == nil {
					if row.DisplayPlace != nil {
						t.Fatalf("row %d display_place = %d, want nil", index, *row.DisplayPlace)
					}
					continue
				}
				if row.DisplayPlace == nil || *row.DisplayPlace != *wantPlace {
					t.Fatalf("row %d display_place = %v, want %d", index, row.DisplayPlace, *wantPlace)
				}
			}
		})
	}
}

func TestPrizeDistributionHandlerFiltersDoNotChangeAssignments(t *testing.T) {
	h := newPrizeDistributionHTTPTestHandler()

	unfiltered := getPrizeDistribution(t, h, "/api/events/77/prize-distribution")
	filtered := getPrizeDistribution(t, h, "/api/events/77/prize-distribution?gender=male&bike_type=gravel&match_reason=place")

	if len(filtered.Distribution) != 1 {
		t.Fatalf("filtered distribution length = %d, want 1", len(filtered.Distribution))
	}
	if filtered.Distribution[0].ParticipantID != 2 {
		t.Fatalf("filtered participant_id = %d, want 2", filtered.Distribution[0].ParticipantID)
	}
	if len(filtered.Distribution[0].MatchedGiftAssignments) != 1 {
		t.Fatalf("filtered assignments = %d, want 1", len(filtered.Distribution[0].MatchedGiftAssignments))
	}

	unfilteredGiftID := prizeGiftAssignmentID(t, unfiltered, 2)
	filteredGiftID := filtered.Distribution[0].MatchedGiftAssignments[0].GiftID
	if filteredGiftID != unfilteredGiftID {
		t.Fatalf("filtered gift assignment = %d, want unchanged %d", filteredGiftID, unfilteredGiftID)
	}
}

func TestPrizeDistributionHandlerPaginatesAfterFilters(t *testing.T) {
	h := newPrizeDistributionHTTPTestHandler(49)

	got := getPrizeDistribution(t, h, "/api/events/77/prize-distribution?gender=male&bike_type=gravel&page=2&page_size=50")
	if got.Total != 51 {
		t.Fatalf("total = %d, want 51", got.Total)
	}
	if got.Page != 2 || got.PageSize != 50 {
		t.Fatalf("page = %d/%d, want 2/50", got.Page, got.PageSize)
	}
	if len(got.Distribution) != 1 {
		t.Fatalf("distribution length = %d, want 1", len(got.Distribution))
	}
	if got.Distribution[0].ParticipantID != 53 {
		t.Fatalf("participant_id = %d, want 53", got.Distribution[0].ParticipantID)
	}
	if got.Distribution[0].DisplayPlace == nil || *got.Distribution[0].DisplayPlace != 51 {
		t.Fatalf("display_place = %v, want 51", got.Distribution[0].DisplayPlace)
	}
}

func newPrizeDistributionHTTPTestHandler(extraMaleGravelParticipants ...int) *PrizeDistributionHandler {
	firstElapsed, firstMoving := 3600, 2100
	secondElapsed, secondMoving := 3600, 2000
	thirdElapsed, thirdMoving := 4200, 2400
	dnfElapsed, dnfMoving := 3000, 1800
	place := 1

	participants := []*entity.Participant{
		{
			ID:       1,
			UserID:   101,
			EventID:  77,
			Gender:   valueobject.GenderMale,
			BikeType: valueobject.BikeTypeGravel,
			Status:   valueobject.ParticipantStatusActive,
			User:     &entity.User{ID: 101, Username: "first"},
		},
		{
			ID:       2,
			UserID:   102,
			EventID:  77,
			Gender:   valueobject.GenderMale,
			BikeType: valueobject.BikeTypeGravel,
			Status:   valueobject.ParticipantStatusActive,
			User:     &entity.User{ID: 102, Username: "second"},
		},
		{
			ID:       3,
			UserID:   103,
			EventID:  77,
			Gender:   valueobject.GenderFemale,
			BikeType: valueobject.BikeTypeMTB,
			Status:   valueobject.ParticipantStatusActive,
			User:     &entity.User{ID: 103, Username: "third"},
		},
		{
			ID:       4,
			UserID:   104,
			EventID:  77,
			Gender:   valueobject.GenderFemale,
			BikeType: valueobject.BikeTypeMTB,
			Status:   valueobject.ParticipantStatusDNF,
			User:     &entity.User{ID: 104, Username: "dnf"},
			Result: &entity.Result{
				ID:             14,
				ParticipantID:  4,
				ElapsedTimeSec: &dnfElapsed,
				MovingTimeSec:  &dnfMoving,
			},
		},
	}

	rows := []*repository.ResultWithPlace{
		{
			Result:              &entity.Result{ID: 11, ParticipantID: 1, ElapsedTimeSec: &firstElapsed, MovingTimeSec: &firstMoving},
			ParticipantGender:   valueobject.GenderMale.String(),
			ParticipantBikeType: valueobject.BikeTypeGravel.String(),
			PlaceAbsolute:       1,
			PlaceByGender:       1,
			PlaceByGenderBike:   1,
		},
		{
			Result:              &entity.Result{ID: 12, ParticipantID: 2, ElapsedTimeSec: &secondElapsed, MovingTimeSec: &secondMoving},
			ParticipantGender:   valueobject.GenderMale.String(),
			ParticipantBikeType: valueobject.BikeTypeGravel.String(),
			PlaceAbsolute:       2,
			PlaceByGender:       2,
			PlaceByGenderBike:   2,
		},
		{
			Result:              &entity.Result{ID: 13, ParticipantID: 3, ElapsedTimeSec: &thirdElapsed, MovingTimeSec: &thirdMoving},
			ParticipantGender:   valueobject.GenderFemale.String(),
			ParticipantBikeType: valueobject.BikeTypeMTB.String(),
			PlaceAbsolute:       3,
			PlaceByGender:       1,
			PlaceByGenderBike:   1,
		},
	}
	if len(extraMaleGravelParticipants) > 0 {
		for index := 0; index < extraMaleGravelParticipants[0]; index++ {
			participantID := uint(5 + index)
			userID := int64(100 + participantID)
			resultID := uint(15 + index)
			elapsed, moving := 5000+index, 3000+index
			participants = append(participants, &entity.Participant{
				ID:       participantID,
				UserID:   userID,
				EventID:  77,
				Gender:   valueobject.GenderMale,
				BikeType: valueobject.BikeTypeGravel,
				Status:   valueobject.ParticipantStatusActive,
				User:     &entity.User{ID: userID, Username: "extra"},
			})
			rows = append(rows, &repository.ResultWithPlace{
				Result:              &entity.Result{ID: resultID, ParticipantID: participantID, ElapsedTimeSec: &elapsed, MovingTimeSec: &moving},
				ParticipantGender:   valueobject.GenderMale.String(),
				ParticipantBikeType: valueobject.BikeTypeGravel.String(),
				PlaceAbsolute:       4 + index,
				PlaceByGender:       3 + index,
				PlaceByGenderBike:   3 + index,
			})
		}
	}
	resultRepo := &prizeDistributionResultRepoFake{rows: rows}
	giftRepo := &prizeDistributionGiftRepoFake{gifts: []*entity.Gift{{
		ID:           101,
		UserID:       501,
		EventID:      77,
		Description:  "Приз за первое место",
		ReviewStatus: entity.GiftReviewStatusApproved,
		Place:        &place,
	}}}
	participantRepo := &prizeDistributionParticipantRepoFake{participants: participants}
	criteriaRepo := &prizeDistributionCriteriaRepoFake{}

	return NewPrizeDistributionHandler(
		query.NewGetPrizeDistributionHandler(resultRepo, giftRepo, participantRepo, criteriaRepo),
		nil,
	)
}

func getPrizeDistribution(t *testing.T, h *PrizeDistributionHandler, target string) dto.PrizeDistributionListResponse {
	t.Helper()
	router := chi.NewRouter()
	router.Get("/api/events/{id}/prize-distribution", h.GetPrizeDistribution)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var got dto.PrizeDistributionListResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return got
}

func prizeGiftAssignmentID(t *testing.T, distribution dto.PrizeDistributionListResponse, participantID uint) uint {
	t.Helper()
	for _, row := range distribution.Distribution {
		if row.ParticipantID != participantID {
			continue
		}
		if len(row.MatchedGiftAssignments) != 1 {
			t.Fatalf("participant %d assignments = %d, want 1", participantID, len(row.MatchedGiftAssignments))
		}
		return row.MatchedGiftAssignments[0].GiftID
	}
	t.Fatalf("participant %d not found", participantID)
	return 0
}

func intPointers(values ...int) []*int {
	result := make([]*int, len(values))
	for index, value := range values {
		if value == 0 {
			continue
		}
		place := value
		result[index] = &place
	}
	return result
}

type prizeDistributionResultRepoFake struct {
	repository.ResultRepository
	rows []*repository.ResultWithPlace
}

func (r *prizeDistributionResultRepoFake) FindByEventWithPlaces(context.Context, uint) ([]*repository.ResultWithPlace, error) {
	return r.rows, nil
}

type prizeDistributionGiftRepoFake struct {
	repository.GiftRepository
	gifts []*entity.Gift
}

func (r *prizeDistributionGiftRepoFake) FindByEventAndReviewStatus(
	context.Context,
	uint,
	entity.GiftReviewStatus,
) ([]*entity.Gift, error) {
	return r.gifts, nil
}

type prizeDistributionParticipantRepoFake struct {
	repository.ParticipantRepository
	participants []*entity.Participant
}

func (r *prizeDistributionParticipantRepoFake) FindByEvent(context.Context, uint) ([]*entity.Participant, error) {
	return r.participants, nil
}

type prizeDistributionCriteriaRepoFake struct {
	repository.CriteriaRepository
}

func (r *prizeDistributionCriteriaRepoFake) FindByGift(context.Context, uint) ([]*entity.Criteria, error) {
	return nil, nil
}

func (r *prizeDistributionCriteriaRepoFake) FindByResult(context.Context, uint) ([]*entity.Criteria, error) {
	return nil, nil
}
