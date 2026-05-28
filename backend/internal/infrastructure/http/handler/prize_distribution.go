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

	// Вызываем query handler
	distributionOutput, err := h.getPrizeDistributionHandler.HandleDetailed(r.Context(), query.GetPrizeDistributionQuery{
		EventID: uint(eventID),
	})
	if err != nil {
		log.Printf("Error getting prize distribution: %v", err)
		response.InternalServerError(w, "Failed to get prize distribution")
		return
	}

	// Конвертируем в DTO
	distribution := distributionOutput.Results
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
	unassignedSlots := make([]*dto.UnassignedPrizeSlotDTO, 0, len(distributionOutput.UnassignedSlots))
	for _, slot := range distributionOutput.UnassignedSlots {
		unassignedSlots = append(unassignedSlots, dto.FromUnassignedPrizeSlot(slot))
	}

	response.Success(w, dto.PrizeDistributionListResponse{
		Distribution:    distributionDTOs,
		UnassignedSlots: unassignedSlots,
		Total:           len(distributionDTOs),
	})
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
