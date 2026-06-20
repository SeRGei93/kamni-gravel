package handler

import (
	"log"
	"net/http"
	"strconv"

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
	paginate := r.URL.Query().Has("page") || r.URL.Query().Has("page_size")
	page := ParsePageParams(r)
	matchReasonFilter := r.URL.Query().Get("match_reason")

	// Вызываем query handler
	distributionOutput, err := h.getPrizeDistributionHandler.HandleDetailed(r.Context(), query.GetPrizeDistributionQuery{
		EventID: uint(eventID),
	})
	if err != nil {
		log.Printf("Error getting prize distribution: %v", err)
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

	// Фильтр по типу совпадения (server-side, до пагинации).
	if matchReasonFilter != "" {
		filtered := make([]*dto.PrizeDistributionDTO, 0, len(distributionDTOs))
		for _, d := range distributionDTOs {
			if d.MatchReason == matchReasonFilter {
				filtered = append(filtered, d)
			}
		}
		distributionDTOs = filtered
	}

	total := len(distributionDTOs)

	// Пагинация (срез страницы) поверх отфильтрованного набора.
	pageItems := distributionDTOs
	if paginate {
		start := page.Offset
		if start > total {
			start = total
		}
		end := start + page.Limit
		if end > total {
			end = total
		}
		pageItems = distributionDTOs[start:end]
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

	log.Printf("DEBUG Prize distribution served: event_id=%d paginated=%t match_reason=%q total=%d page=%d page_size=%d returned=%d",
		eventID, paginate, matchReasonFilter, total, page.Page, page.PageSize, len(pageItems))

	response.Success(w, resp)
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
