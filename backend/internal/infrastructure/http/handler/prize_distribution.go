package handler

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/infrastructure/http/response"
)

// PrizeDistributionHandler обрабатывает запросы для распределения призов
type PrizeDistributionHandler struct {
	getPrizeDistributionHandler *query.GetPrizeDistributionHandler
	getResultsWithPlacesHandler *query.GetResultsWithPlacesHandler
}

// NewPrizeDistributionHandler создаёт новый handler
func NewPrizeDistributionHandler(
	getPrizeDistributionHandler *query.GetPrizeDistributionHandler,
	getResultsWithPlacesHandler *query.GetResultsWithPlacesHandler,
) *PrizeDistributionHandler {
	return &PrizeDistributionHandler{
		getPrizeDistributionHandler: getPrizeDistributionHandler,
		getResultsWithPlacesHandler: getResultsWithPlacesHandler,
	}
}

// GetPrizeDistribution обрабатывает GET /api/events/:id/prize-distribution
func (h *PrizeDistributionHandler) GetPrizeDistribution(w http.ResponseWriter, r *http.Request) {
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Пагинация включается только если переданы page/page_size. Без них возвращаем
	// всё распределение (нужно для страницы призов: подсветка назначенных подарков).
	page := ParsePageParams(r)
	paginate := (r.URL.Query().Has("page") || r.URL.Query().Has("page_size")) && !page.All
	genderFilter := normalizePrizeDistributionFilter(r.URL.Query().Get("gender"))
	bikeTypeFilter := normalizePrizeDistributionFilter(r.URL.Query().Get("bike_type"))
	matchReasonFilter := normalizePrizeDistributionFilter(r.URL.Query().Get("match_reason"))

	// Вызываем query handler
	distributionOutput, err := h.getPrizeDistributionHandler.HandleDetailed(r.Context(), query.GetPrizeDistributionQuery{
		EventID: uint(eventID),
	})
	if err != nil {
		log.Printf("ERROR prize distribution failed: event_id=%d stage=calculate error=%v", eventID, err)
		response.InternalServerError(w, "Failed to get prize distribution")
		return
	}

	// Считаем агрегаты по всему распределению (для карточек статистики).
	distribution := distributionOutput.Results
	stats := &dto.PrizeDistributionStatsDTO{TotalParticipants: len(distribution)}
	for _, dist := range distribution {
		slots := len(dist.MatchedGiftAssignments)
		if slots == 0 {
			slots = len(dist.MatchedGifts)
		}
		if slots > 0 {
			stats.WithPrizes++
			stats.PrizeSlots += slots
		}
	}
	stats.WithoutPrizes = stats.TotalParticipants - stats.WithPrizes

	// Конвертируем в DTO
	distributionDTOs := make([]*dto.PrizeDistributionDTO, 0, len(distribution))
	for _, dist := range distribution {
		dtoObj := &dto.PrizeDistributionDTO{
			ParticipantID:     dist.ParticipantID,
			ParticipantName:   dist.ParticipantName,
			Gender:            dist.Gender,
			BikeType:          dist.BikeType,
			Status:            dist.Status,
			PlaceAbsolute:     dist.PlaceAbsolute,
			PlaceByGender:     dist.PlaceByGender,
			PlaceByGenderBike: dist.PlaceByGenderBike,
			MatchReason:       dist.MatchReason,
		}

		// Конвертируем критерии результата
		if len(dist.ResultCriteria) > 0 {
			dtoObj.ResultCriteria = make([]*dto.CriteriaDTO, len(dist.ResultCriteria))
			for i, c := range dist.ResultCriteria {
				dtoObj.ResultCriteria[i] = dto.FromCriteria(c)
			}
		}

		// Конвертируем подарки
		if len(dist.MatchedGifts) > 0 {
			dtoObj.MatchedGifts = make([]*dto.GiftDTO, len(dist.MatchedGifts))
			for i, gift := range dist.MatchedGifts {
				dtoObj.MatchedGifts[i] = dto.FromGift(gift)
			}
		}
		if len(dist.MatchedGiftAssignments) > 0 {
			dtoObj.MatchedGiftAssignments = make([]*dto.PrizeGiftAssignmentDTO, len(dist.MatchedGiftAssignments))
			for i, assignment := range dist.MatchedGiftAssignments {
				dtoObj.MatchedGiftAssignments[i] = dto.FromPrizeGiftAssignment(assignment)
			}
		}

		distributionDTOs = append(distributionDTOs, dtoObj)
	}

	cohortRows := filterAndRankPrizeDistribution(distributionDTOs, genderFilter, bikeTypeFilter)
	filteredRows := filterPrizeDistributionByMatchReason(cohortRows, matchReasonFilter)
	total := len(filteredRows)

	// Пагинация (срез страницы) поверх отфильтрованного набора.
	pageItems := filteredRows
	if paginate {
		start := page.Offset
		if start > total {
			start = total
		}
		end := start + page.Limit
		if end > total {
			end = total
		}
		pageItems = filteredRows[start:end]
	}

	unassignedSlots := make([]*dto.UnassignedPrizeSlotDTO, 0, len(distributionOutput.UnassignedSlots))
	for _, slot := range distributionOutput.UnassignedSlots {
		unassignedSlots = append(unassignedSlots, dto.FromUnassignedPrizeSlot(slot))
	}

	resp := dto.PrizeDistributionListResponse{
		Distribution:    pageItems,
		UnassignedSlots: unassignedSlots,
		Total:           total,
		Stats:           stats,
	}
	if paginate {
		resp.Page = page.Page
		resp.PageSize = page.PageSize
	}

	log.Printf("DEBUG Prize distribution served: event_id=%d paginated=%t gender=%q bike_type=%q match_reason=%q source_rows=%d cohort_rows=%d filtered_rows=%d total=%d page=%d page_size=%d returned=%d",
		eventID,
		paginate,
		genderFilter,
		bikeTypeFilter,
		matchReasonFilter,
		len(distributionDTOs),
		len(cohortRows),
		len(filteredRows),
		total,
		page.Page,
		page.PageSize,
		len(pageItems),
	)

	response.Success(w, resp)
}

func normalizePrizeDistributionFilter(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "all" {
		return ""
	}
	return normalized
}

func filterAndRankPrizeDistribution(
	rows []*dto.PrizeDistributionDTO,
	genderFilter, bikeTypeFilter string,
) []*dto.PrizeDistributionDTO {
	cohort := make([]*dto.PrizeDistributionDTO, 0, len(rows))
	for _, row := range rows {
		if genderFilter != "" && row.Gender != genderFilter {
			continue
		}
		if bikeTypeFilter != "" && row.BikeType != bikeTypeFilter {
			continue
		}
		row.DisplayPlace = nil
		cohort = append(cohort, row)
	}

	sort.SliceStable(cohort, func(i, j int) bool {
		leftRanked := isRankedPrizeDistributionRow(cohort[i])
		rightRanked := isRankedPrizeDistributionRow(cohort[j])
		return leftRanked && !rightRanked
	})

	displayPlace := 0
	for _, row := range cohort {
		if !isRankedPrizeDistributionRow(row) {
			continue
		}
		displayPlace++
		place := displayPlace
		row.DisplayPlace = &place
	}

	return cohort
}

func isRankedPrizeDistributionRow(row *dto.PrizeDistributionDTO) bool {
	return row.Status == "active" && row.PlaceAbsolute > 0
}

func filterPrizeDistributionByMatchReason(
	rows []*dto.PrizeDistributionDTO,
	matchReasonFilter string,
) []*dto.PrizeDistributionDTO {
	if matchReasonFilter == "" {
		return rows
	}

	filtered := make([]*dto.PrizeDistributionDTO, 0, len(rows))
	for _, row := range rows {
		if row.MatchReason == matchReasonFilter {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

// GetResultsWithPlaces обрабатывает GET /api/events/:id/results
func (h *PrizeDistributionHandler) GetResultsWithPlaces(w http.ResponseWriter, r *http.Request) {
	eventIDStr := chi.URLParam(r, "id")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Вызываем query handler
	results, err := h.getResultsWithPlacesHandler.Handle(r.Context(), query.GetResultsWithPlacesQuery{
		EventID: uint(eventID),
	})
	if err != nil {
		log.Printf("Error getting results with places: %v", err)
		response.InternalServerError(w, "Failed to get results with places")
		return
	}

	// Конвертируем в DTO
	resultDTOs := make([]*dto.ResultDTO, 0, len(results))
	for _, rwp := range results {
		resultDTO := dto.FromResult(rwp.Result)
		resultDTOs = append(resultDTOs, resultDTO)
	}

	response.Success(w, dto.ResultListResponse{
		Results: resultDTOs,
		Total:   len(resultDTOs),
	})
}
