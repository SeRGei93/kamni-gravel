package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/http/response"
)

// ResultsHandler обрабатывает запросы для результатов
type ResultsHandler struct {
	resultRepo          repository.ResultRepository
	participantRepo     repository.ParticipantRepository
	criteriaRepo        repository.CriteriaRepository
	submitResultHandler *command.SubmitResultHandler
	manualResultHandler *command.CreateManualResultHandler
	updateResultHandler *command.UpdateResultHandler
}

// NewResultsHandler создаёт новый handler
func NewResultsHandler(
	resultRepo repository.ResultRepository,
	participantRepo repository.ParticipantRepository,
	criteriaRepo repository.CriteriaRepository,
	submitResultHandler *command.SubmitResultHandler,
	manualResultHandler *command.CreateManualResultHandler,
	updateResultHandler *command.UpdateResultHandler,
) *ResultsHandler {
	return &ResultsHandler{
		resultRepo:          resultRepo,
		participantRepo:     participantRepo,
		criteriaRepo:        criteriaRepo,
		submitResultHandler: submitResultHandler,
		manualResultHandler: manualResultHandler,
		updateResultHandler: updateResultHandler,
	}
}

// parseOptionalMinskTime разбирает опциональную метку времени (RFC3339 или
// Минск wall time). Возвращает (значение, задано_ли, ошибка_парсинга).
func parseOptionalMinskTime(raw *string) (*time.Time, bool, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, false, nil
	}
	parsed, err := valueobject.ParseMinskDateTime(*raw)
	if err != nil {
		return nil, false, err
	}
	return &parsed, true, nil
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

// CreateResultRequest представляет запрос на создание результата.
// Общее время задаётся через elapsed_time_sec ИЛИ парой started_at+finished_at.
// distance_meters — дистанция в метрах (конвертация км↔м на стороне фронтенда).
type CreateResultRequest struct {
	ResultLink     string   `json:"result_link,omitempty"`
	ElapsedTimeSec *int     `json:"elapsed_time_sec,omitempty"`
	MovingTimeSec  *int     `json:"moving_time_sec,omitempty"`
	StartedAt      *string  `json:"started_at,omitempty"`
	FinishedAt     *string  `json:"finished_at,omitempty"`
	DistanceMeters *int     `json:"distance_meters,omitempty"`
	AvgHeartRate   *int     `json:"avg_heart_rate,omitempty"`
	MaxHeartRate   *int     `json:"max_heart_rate,omitempty"`
	PeakSpeedKmh   *float64 `json:"peak_speed_kmh,omitempty"`
	AvgCadence     *int     `json:"avg_cadence,omitempty"`
	Calories       *int     `json:"calories,omitempty"`
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

	startedAt, startPresent, err := parseOptionalMinskTime(req.StartedAt)
	if err != nil {
		log.Printf("WARN Manual result creation rejected: participant_id=%d error_class=invalid_started_at", participantID)
		response.BadRequest(w, "Invalid started_at format. Use ISO 8601 (RFC3339)")
		return
	}
	finishedAt, finishPresent, err := parseOptionalMinskTime(req.FinishedAt)
	if err != nil {
		log.Printf("WARN Manual result creation rejected: participant_id=%d error_class=invalid_finished_at", participantID)
		response.BadRequest(w, "Invalid finished_at format. Use ISO 8601 (RFC3339)")
		return
	}

	// Общее время обязательно: либо elapsed_time_sec, либо пара старт+финиш.
	if req.ElapsedTimeSec == nil && !(startPresent && finishPresent) {
		log.Printf("WARN Manual result creation rejected: participant_id=%d error_class=missing_total", participantID)
		response.BadRequest(w, "Provide elapsed_time_sec or both started_at and finished_at")
		return
	}

	result, err := h.manualResultHandler.Handle(r.Context(), command.CreateManualResultCommand{
		ParticipantID:  uint(participantID),
		ResultLink:     req.ResultLink,
		ElapsedTimeSec: req.ElapsedTimeSec,
		MovingTimeSec:  req.MovingTimeSec,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		DistanceMeters: req.DistanceMeters,
		AvgHeartRate:   req.AvgHeartRate,
		MaxHeartRate:   req.MaxHeartRate,
		PeakSpeedKmh:   req.PeakSpeedKmh,
		AvgCadence:     req.AvgCadence,
		Calories:       req.Calories,
	})
	if err != nil {
		errorClass := resultCreateErrorClass(err)
		if errors.Is(err, command.ErrInvalidResultLink) || errors.Is(err, command.ErrInvalidResultTime) ||
			errors.Is(err, command.ErrInvalidResultMetric) ||
			errors.Is(err, command.ErrParticipantNotFound) || errors.Is(err, command.ErrResultAlreadyExists) {
			log.Printf("WARN Result creation rejected: participant_id=%d error_class=%s", participantID, errorClass)
		} else {
			log.Printf("ERROR Result creation failed: participant_id=%d error_class=%s error=%v", participantID, errorClass, err)
		}

		switch {
		case errors.Is(err, command.ErrInvalidResultLink):
			response.BadRequest(w, "Only Strava result links are accepted")
		case errors.Is(err, command.ErrInvalidResultTime):
			response.BadRequest(w, "Total time must be positive (from elapsed_time_sec or finish-start), and moving_time_sec must be between 0 and total")
		case errors.Is(err, command.ErrInvalidResultMetric):
			response.BadRequest(w, "Metric values (distance, heart rate, cadence, calories, peak speed) must not be negative")
		case errors.Is(err, command.ErrParticipantNotFound):
			response.NotFound(w, "Participant not found")
		case errors.Is(err, command.ErrResultAlreadyExists):
			response.Conflict(w, "Participant already has a current result")
		default:
			response.InternalServerError(w, "Failed to create result")
		}
		return
	}

	response.Created(w, dto.FromResult(result))
}

func resultCreateErrorClass(err error) string {
	switch {
	case errors.Is(err, command.ErrInvalidResultLink):
		return "invalid_result_link"
	case errors.Is(err, command.ErrInvalidResultTime):
		return "invalid_result_time"
	case errors.Is(err, command.ErrInvalidResultMetric):
		return "invalid_result_metric"
	case errors.Is(err, command.ErrResultNotFound):
		return "result_not_found"
	case errors.Is(err, command.ErrResultAlreadyExists):
		return "result_already_exists"
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

// UpdateResultRequest представляет запрос на обновление результата.
// Семантика — полная замена метрик: отсутствующее поле очищается.
// Общее время — через elapsed_time_sec ИЛИ пару started_at+finished_at.
type UpdateResultRequest struct {
	ElapsedTimeSec *int     `json:"elapsed_time_sec,omitempty"`
	MovingTimeSec  *int     `json:"moving_time_sec,omitempty"`
	StartedAt      *string  `json:"started_at,omitempty"`
	FinishedAt     *string  `json:"finished_at,omitempty"`
	DistanceMeters *int     `json:"distance_meters,omitempty"`
	AvgHeartRate   *int     `json:"avg_heart_rate,omitempty"`
	MaxHeartRate   *int     `json:"max_heart_rate,omitempty"`
	PeakSpeedKmh   *float64 `json:"peak_speed_kmh,omitempty"`
	AvgCadence     *int     `json:"avg_cadence,omitempty"`
	Calories       *int     `json:"calories,omitempty"`
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

	startedAt, startPresent, err := parseOptionalMinskTime(req.StartedAt)
	if err != nil {
		log.Printf("WARN Result update rejected: result_id=%d error_class=invalid_started_at", id)
		response.BadRequest(w, "Invalid started_at format. Use ISO 8601 (RFC3339)")
		return
	}
	finishedAt, finishPresent, err := parseOptionalMinskTime(req.FinishedAt)
	if err != nil {
		log.Printf("WARN Result update rejected: result_id=%d error_class=invalid_finished_at", id)
		response.BadRequest(w, "Invalid finished_at format. Use ISO 8601 (RFC3339)")
		return
	}

	if req.ElapsedTimeSec == nil && !(startPresent && finishPresent) {
		log.Printf("WARN Result update rejected: result_id=%d error_class=missing_total", id)
		response.BadRequest(w, "Provide elapsed_time_sec or both started_at and finished_at")
		return
	}

	result, err := h.updateResultHandler.Handle(r.Context(), command.UpdateResultCommand{
		ResultID:       uint(id),
		ElapsedTimeSec: req.ElapsedTimeSec,
		MovingTimeSec:  req.MovingTimeSec,
		StartedAt:      startedAt,
		FinishedAt:     finishedAt,
		DistanceMeters: req.DistanceMeters,
		AvgHeartRate:   req.AvgHeartRate,
		MaxHeartRate:   req.MaxHeartRate,
		PeakSpeedKmh:   req.PeakSpeedKmh,
		AvgCadence:     req.AvgCadence,
		Calories:       req.Calories,
	})
	if err != nil {
		errorClass := resultCreateErrorClass(err)
		switch {
		case errors.Is(err, command.ErrResultNotFound):
			log.Printf("WARN Result update rejected: result_id=%d error_class=%s", id, errorClass)
			response.NotFound(w, "Result not found")
		case errors.Is(err, command.ErrInvalidResultTime):
			log.Printf("WARN Result update rejected: result_id=%d error_class=%s", id, errorClass)
			response.BadRequest(w, "Total time must be positive (from elapsed_time_sec or finish-start), and moving_time_sec must be between 0 and total")
		case errors.Is(err, command.ErrInvalidResultMetric):
			log.Printf("WARN Result update rejected: result_id=%d error_class=%s", id, errorClass)
			response.BadRequest(w, "Metric values (distance, heart rate, cadence, calories, peak speed) must not be negative")
		default:
			log.Printf("ERROR Result update failed: result_id=%d error_class=%s error=%v", id, errorClass, err)
			response.InternalServerError(w, "Failed to update result")
		}
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
