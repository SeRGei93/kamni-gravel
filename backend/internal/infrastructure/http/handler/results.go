package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
)

// ResultsHandler обрабатывает запросы для результатов
type ResultsHandler struct {
	resultRepo          repository.ResultRepository
	participantRepo     repository.ParticipantRepository
	criteriaRepo        repository.CriteriaRepository
	submitResultHandler *command.SubmitResultHandler
}

// NewResultsHandler создаёт новый handler
func NewResultsHandler(
	resultRepo repository.ResultRepository,
	participantRepo repository.ParticipantRepository,
	criteriaRepo repository.CriteriaRepository,
	submitResultHandler *command.SubmitResultHandler,
) *ResultsHandler {
	return &ResultsHandler{
		resultRepo:          resultRepo,
		participantRepo:     participantRepo,
		criteriaRepo:        criteriaRepo,
		submitResultHandler: submitResultHandler,
	}
}

// GetByParticipant обрабатывает GET /api/participants/:participantId/results
func (h *ResultsHandler) GetByParticipant(w http.ResponseWriter, r *http.Request) {
	participantIDStr := chi.URLParam(r, "participantId")
	participantID, err := strconv.ParseUint(participantIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return
	}

	results, err := h.resultRepo.FindByParticipant(r.Context(), uint(participantID))
	if err != nil {
		log.Printf("Error getting results: %v", err)
		response.InternalServerError(w, "Failed to get results")
		return
	}

	resultDTOs := make([]*dto.ResultDTO, 0, len(results))
	for _, result := range results {
		// Загружаем критерии для каждого результата
		criteria, err := h.criteriaRepo.FindByResult(r.Context(), result.ID)
		if err != nil {
			log.Printf("Error getting criteria for result %d: %v", result.ID, err)
			// Продолжаем без критериев
		} else {
			result.Criteria = criteria
		}
		resultDTOs = append(resultDTOs, dto.FromResult(result))
	}

	response.Success(w, dto.ResultListResponse{
		Results: resultDTOs,
		Total:   len(resultDTOs),
	})
}

// GetByID обрабатывает GET /api/results/:id
func (h *ResultsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid result ID")
		return
	}

	result, err := h.resultRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Error getting result: %v", err)
		response.InternalServerError(w, "Failed to get result")
		return
	}

	if result == nil {
		response.NotFound(w, "Result not found")
		return
	}

	response.Success(w, dto.FromResult(result))
}

// CreateResultRequest представляет запрос на создание результата
type CreateResultRequest struct {
	ResultLink string `json:"result_link"`
}

// Create обрабатывает POST /api/participants/:participantId/results
func (h *ResultsHandler) Create(w http.ResponseWriter, r *http.Request) {
	participantIDStr := chi.URLParam(r, "participantId")
	participantID, err := strconv.ParseUint(participantIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return
	}

	var req CreateResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if req.ResultLink == "" {
		response.BadRequest(w, "result_link is required")
		return
	}

	participant, err := h.submitResultHandler.Handle(r.Context(), command.SubmitResultCommand{
		ParticipantID: uint(participantID),
		ResultLink:    req.ResultLink,
	})
	if err != nil {
		errorClass := resultCreateErrorClass(err)
		if errors.Is(err, command.ErrInvalidResultLink) || errors.Is(err, command.ErrParticipantNotFound) ||
			errors.Is(err, command.ErrResultSubmissionNotOpen) || errors.Is(err, command.ErrEventStartNotConfigured) ||
			errors.Is(err, command.ErrEventNotStarted) || errors.Is(err, command.ErrEventNotFound) {
			log.Printf("WARN Result creation rejected: participant_id=%d error_class=%s", participantID, errorClass)
		} else {
			log.Printf("ERROR Result creation failed: participant_id=%d error_class=%s error=%v", participantID, errorClass, err)
		}

		switch {
		case errors.Is(err, command.ErrInvalidResultLink):
			response.BadRequest(w, "Only Strava result links are accepted")
		case errors.Is(err, command.ErrParticipantNotFound):
			response.NotFound(w, "Participant not found")
		case errors.Is(err, command.ErrEventNotFound):
			response.Conflict(w, "Participant event was not found")
		case errors.Is(err, command.ErrEventStartNotConfigured):
			response.Conflict(w, "Event start time is not configured")
		case errors.Is(err, command.ErrEventNotStarted):
			response.Conflict(w, "Result submission is available only after the event start time in Minsk UTC+3")
		case errors.Is(err, command.ErrResultSubmissionNotOpen):
			response.Conflict(w, "Result submission is not open")
		default:
			response.InternalServerError(w, "Failed to create result")
		}
		return
	}

	response.Created(w, dto.FromResult(participant.Result))
}

func resultCreateErrorClass(err error) string {
	switch {
	case errors.Is(err, command.ErrInvalidResultLink):
		return "invalid_result_link"
	case errors.Is(err, command.ErrParticipantNotFound):
		return "participant_not_found"
	case errors.Is(err, command.ErrEventNotFound):
		return "event_not_found"
	case errors.Is(err, command.ErrEventStartNotConfigured):
		return "event_start_not_configured"
	case errors.Is(err, command.ErrEventNotStarted):
		return "event_not_started"
	case errors.Is(err, command.ErrResultSubmissionNotOpen):
		return "result_submission_not_open"
	default:
		return "unknown"
	}
}

// UpdateResultRequest представляет запрос на обновление результата
type UpdateResultRequest struct {
	ElapsedTimeSec *int `json:"elapsed_time_sec,omitempty"`
	MovingTimeSec  *int `json:"moving_time_sec,omitempty"`
}

// Update обрабатывает PUT /api/results/:id
func (h *ResultsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid result ID")
		return
	}

	var req UpdateResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Проверяем существование результата
	result, err := h.resultRepo.FindByID(r.Context(), uint(id))
	if err != nil || result == nil {
		response.NotFound(w, "Result not found")
		return
	}

	// Обновляем время
	if err := h.resultRepo.UpdateTime(r.Context(), uint(id), req.ElapsedTimeSec, req.MovingTimeSec); err != nil {
		log.Printf("Error updating result: %v", err)
		response.InternalServerError(w, "Failed to update result")
		return
	}

	// Получаем обновлённый результат
	result, err = h.resultRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Error getting updated result: %v", err)
		response.InternalServerError(w, "Failed to get updated result")
		return
	}

	response.Success(w, dto.FromResult(result))
}

// Delete обрабатывает DELETE /api/results/:id
func (h *ResultsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid result ID")
		return
	}

	// Проверяем существование результата
	result, err := h.resultRepo.FindByID(r.Context(), uint(id))
	if err != nil || result == nil {
		response.NotFound(w, "Result not found")
		return
	}

	if err := h.resultRepo.Delete(r.Context(), uint(id)); err != nil {
		log.Printf("Error deleting result: %v", err)
		response.InternalServerError(w, "Failed to delete result")
		return
	}

	response.NoContent(w)
}

// AddCriteriaRequest представляет запрос на добавление критерия к результату
type AddCriteriaRequest struct {
	CriteriaID uint `json:"criteria_id"`
}

// AddCriteria обрабатывает POST /api/results/:id/criteria
func (h *ResultsHandler) AddCriteria(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid result ID")
		return
	}

	var req AddCriteriaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Проверяем существование результата
	_, err = h.resultRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		response.NotFound(w, "Result not found")
		return
	}

	// Добавляем критерий
	if err := h.resultRepo.AddCriteria(r.Context(), uint(id), req.CriteriaID); err != nil {
		log.Printf("Error adding criteria to result: %v", err)
		response.InternalServerError(w, "Failed to add criteria")
		return
	}

	// Получаем результат с критериями
	result, err := h.resultRepo.FindWithCriteria(r.Context(), uint(id))
	if err != nil {
		log.Printf("Error getting result with criteria: %v", err)
		response.InternalServerError(w, "Failed to get result")
		return
	}

	response.Success(w, dto.FromResult(result))
}

// RemoveCriteria обрабатывает DELETE /api/results/:id/criteria/:criteriaId
func (h *ResultsHandler) RemoveCriteria(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid result ID")
		return
	}

	criteriaIDStr := chi.URLParam(r, "criteriaId")
	criteriaID, err := strconv.ParseUint(criteriaIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid criteria ID")
		return
	}

	// Проверяем существование результата
	_, err = h.resultRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		response.NotFound(w, "Result not found")
		return
	}

	// Удаляем критерий
	if err := h.resultRepo.RemoveCriteria(r.Context(), uint(id), uint(criteriaID)); err != nil {
		log.Printf("Error removing criteria from result: %v", err)
		response.InternalServerError(w, "Failed to remove criteria")
		return
	}

	// Получаем результат с критериями
	result, err := h.resultRepo.FindWithCriteria(r.Context(), uint(id))
	if err != nil {
		log.Printf("Error getting result with criteria: %v", err)
		response.InternalServerError(w, "Failed to get result")
		return
	}

	response.Success(w, dto.FromResult(result))
}
