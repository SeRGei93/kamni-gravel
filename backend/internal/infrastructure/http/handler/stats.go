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

// StatsHandler обрабатывает запросы для статистики
type StatsHandler struct {
	getStatsHandler      *query.GetStatsHandler
	getDailyStatsHandler *query.GetDailyStatsHandler
}

// NewStatsHandler создаёт новый handler
func NewStatsHandler(
	getStatsHandler *query.GetStatsHandler,
	getDailyStatsHandler *query.GetDailyStatsHandler,
) *StatsHandler {
	return &StatsHandler{
		getStatsHandler:      getStatsHandler,
		getDailyStatsHandler: getDailyStatsHandler,
	}
}

// GetAll обрабатывает GET /api/stats - статистика по всем событиям
func (h *StatsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Вызываем query handler без фильтра по событию
	stats, err := h.getStatsHandler.Handle(r.Context(), query.GetStatsQuery{
		EventID: nil,
	})
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		response.InternalServerError(w, "Failed to get stats")
		return
	}

	// Конвертируем в DTO
	statsDTOs := make([]*dto.StatsDTO, 0, len(stats))
	for _, stat := range stats {
		statsDTOs = append(statsDTOs, dto.FromEventStats(stat))
	}

	// Возвращаем ответ
	response.Success(w, dto.StatsListResponse{
		Stats: statsDTOs,
		Total: len(statsDTOs),
	})
}

// GetByEventID обрабатывает GET /api/events/:eventId/stats - статистика конкретного события
func (h *StatsHandler) GetByEventID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем eventID из URL
	eventIDStr := chi.URLParam(r, "eventId")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Вызываем query handler с фильтром по событию
	stats, err := h.getStatsHandler.Handle(r.Context(), query.GetStatsQuery{
		EventID: func() *uint { id := uint(eventID); return &id }(),
	})
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		if err.Error() == "event not found" {
			response.NotFound(w, err.Error())
		} else {
			response.InternalServerError(w, "Failed to get stats")
		}
		return
	}

	if len(stats) == 0 {
		response.NotFound(w, "Stats not found")
		return
	}

	// Возвращаем статистику первого (и единственного) события
	response.Success(w, dto.FromEventStats(stats[0]))
}

// GetDailyByEventID обрабатывает GET /api/events/:eventId/stats/daily —
// посуточная статистика события (новые участники и финиши по дням).
func (h *StatsHandler) GetDailyByEventID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем eventID из URL
	eventIDStr := chi.URLParam(r, "eventId")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Вызываем query handler
	stats, err := h.getDailyStatsHandler.Handle(r.Context(), query.GetDailyStatsQuery{
		EventID: uint(eventID),
	})
	if err != nil {
		log.Printf("Error getting daily stats: %v", err)
		if err.Error() == "event not found" {
			response.NotFound(w, err.Error())
		} else {
			response.InternalServerError(w, "Failed to get daily stats")
		}
		return
	}

	response.Success(w, dto.FromEventDailyStats(stats))
}
