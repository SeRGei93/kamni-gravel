package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
)

// CriteriaHandler обрабатывает запросы для критериев
type CriteriaHandler struct {
	criteriaRepo           repository.CriteriaRepository
	getCriteriaHandler     *query.GetCriteriaHandler
	getCriteriaByIDHandler *query.GetCriteriaByIDHandler
	createCriteriaHandler  *command.CreateCriteriaHandler
	updateCriteriaHandler  *command.UpdateCriteriaHandler
}

// NewCriteriaHandler создаёт новый handler
func NewCriteriaHandler(
	criteriaRepo repository.CriteriaRepository,
	getCriteriaHandler *query.GetCriteriaHandler,
	getCriteriaByIDHandler *query.GetCriteriaByIDHandler,
	createCriteriaHandler *command.CreateCriteriaHandler,
	updateCriteriaHandler *command.UpdateCriteriaHandler,
) *CriteriaHandler {
	return &CriteriaHandler{
		criteriaRepo:           criteriaRepo,
		getCriteriaHandler:     getCriteriaHandler,
		getCriteriaByIDHandler: getCriteriaByIDHandler,
		createCriteriaHandler:  createCriteriaHandler,
		updateCriteriaHandler:  updateCriteriaHandler,
	}
}

// GetAll обрабатывает GET /api/criteria - список критериев
func (h *CriteriaHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// Пагинация включается только если переданы page/page_size. Без них возвращаем
	// все критерии (нужно для селекторов критериев на других страницах).
	page := ParsePageParams(r)
	paginate := (q.Has("page") || q.Has("page_size")) && !page.All

	queryParams := query.GetCriteriaQuery{}
	if paginate {
		queryParams.Limit = page.Limit
		queryParams.Offset = page.Offset
	}

	// Парсим query параметры
	if criteriaType := q.Get("type"); criteriaType != "" {
		queryParams.CriteriaType = &criteriaType
	}

	// Вызываем query handler
	criteria, total, err := h.getCriteriaHandler.Handle(r.Context(), queryParams)
	if err != nil {
		log.Printf("Error getting criteria: %v", err)
		response.InternalServerError(w, "Failed to get criteria")
		return
	}

	// Конвертируем в DTO
	criteriaDTOs := make([]*dto.CriteriaDTO, 0, len(criteria))
	for _, c := range criteria {
		criteriaDTOs = append(criteriaDTOs, dto.FromCriteria(c))
	}

	resp := dto.CriteriaListResponse{
		Criteria: criteriaDTOs,
		Total:    total,
	}
	if paginate {
		resp.Page = page.Page
		resp.PageSize = page.PageSize
		log.Printf("DEBUG Criteria list served (paged): total=%d page=%d page_size=%d returned=%d", total, page.Page, page.PageSize, len(criteriaDTOs))
	} else {
		log.Printf("DEBUG Criteria list served (all): total=%d returned=%d", total, len(criteriaDTOs))
	}

	response.Success(w, resp)
}

// GetByID обрабатывает GET /api/criteria/:id - детали критерия
func (h *CriteriaHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid criteria ID")
		return
	}

	criteria, err := h.getCriteriaByIDHandler.Handle(r.Context(), query.GetCriteriaByIDQuery{
		CriteriaID: uint(id),
	})
	if err != nil {
		log.Printf("Error getting criteria: %v", err)
		response.InternalServerError(w, "Failed to get criteria")
		return
	}

	if criteria == nil {
		response.NotFound(w, "Criteria not found")
		return
	}

	response.Success(w, dto.FromCriteria(criteria))
}

// CreateRequest представляет запрос на создание критерия
type CreateCriteriaRequest struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	CriteriaType string `json:"criteria_type"`
}

// Create обрабатывает POST /api/criteria - создание критерия
func (h *CriteriaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCriteriaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	criteria, err := h.createCriteriaHandler.Handle(r.Context(), command.CreateCriteriaCommand{
		Name:         req.Name,
		Description:  req.Description,
		CriteriaType: req.CriteriaType,
	})
	if err != nil {
		log.Printf("Error creating criteria: %v", err)
		switch err {
		case command.ErrCriteriaNameRequired:
			response.BadRequest(w, err.Error())
		case command.ErrInvalidCriteriaType:
			response.BadRequest(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to create criteria")
		}
		return
	}

	response.Created(w, dto.FromCriteria(criteria))
}

// UpdateRequest представляет запрос на обновление критерия
type UpdateCriteriaRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	CriteriaType *string `json:"criteria_type,omitempty"`
}

// Update обрабатывает PUT /api/criteria/:id - обновление критерия
func (h *CriteriaHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid criteria ID")
		return
	}

	var req UpdateCriteriaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	criteria, err := h.updateCriteriaHandler.Handle(r.Context(), command.UpdateCriteriaCommand{
		CriteriaID:   uint(id),
		Name:         req.Name,
		Description:  req.Description,
		CriteriaType: req.CriteriaType,
	})
	if err != nil {
		log.Printf("Error updating criteria: %v", err)
		switch err {
		case command.ErrCriteriaNotFound:
			response.NotFound(w, err.Error())
		case command.ErrCriteriaNameRequired, command.ErrInvalidCriteriaType:
			response.BadRequest(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to update criteria")
		}
		return
	}

	response.Success(w, dto.FromCriteria(criteria))
}

// Delete обрабатывает DELETE /api/criteria/:id - удаление критерия
func (h *CriteriaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid criteria ID")
		return
	}

	// Проверяем существование критерия
	_, err = h.criteriaRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Criteria not found: %v", err)
		response.NotFound(w, "Criteria not found")
		return
	}

	// Удаляем критерий
	if err := h.criteriaRepo.Delete(r.Context(), uint(id)); err != nil {
		log.Printf("Error deleting criteria: %v", err)
		response.InternalServerError(w, "Failed to delete criteria")
		return
	}

	response.NoContent(w)
}
