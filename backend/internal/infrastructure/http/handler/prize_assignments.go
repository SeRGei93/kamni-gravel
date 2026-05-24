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

// PrizeAssignmentsHandler обрабатывает запросы для назначений призов
type PrizeAssignmentsHandler struct {
	prizeAssignmentRepo           repository.PrizeAssignmentRepository
	getPrizeAssignmentsHandler   *query.GetPrizeAssignmentsHandler
	getPrizeAssignmentByIDHandler *query.GetPrizeAssignmentByIDHandler
	assignPrizeHandler           *command.AssignPrizeHandler
}

// NewPrizeAssignmentsHandler создаёт новый handler
func NewPrizeAssignmentsHandler(
	prizeAssignmentRepo repository.PrizeAssignmentRepository,
	getPrizeAssignmentsHandler *query.GetPrizeAssignmentsHandler,
	getPrizeAssignmentByIDHandler *query.GetPrizeAssignmentByIDHandler,
	assignPrizeHandler *command.AssignPrizeHandler,
) *PrizeAssignmentsHandler {
	return &PrizeAssignmentsHandler{
		prizeAssignmentRepo:          prizeAssignmentRepo,
		getPrizeAssignmentsHandler:   getPrizeAssignmentsHandler,
		getPrizeAssignmentByIDHandler: getPrizeAssignmentByIDHandler,
		assignPrizeHandler:           assignPrizeHandler,
	}
}

// GetAll обрабатывает GET /api/events/:eventId/prize-assignments или GET /api/participants/:participantId/prize-assignments
func (h *PrizeAssignmentsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	queryParams := query.GetPrizeAssignmentsQuery{}

	// Проверяем, есть ли eventId в URL
	if eventIDStr := chi.URLParam(r, "eventId"); eventIDStr != "" {
		eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
		if err != nil {
			response.BadRequest(w, "Invalid event ID")
			return
		}
		eventIDUint := uint(eventID)
		queryParams.EventID = &eventIDUint
	}

	// Проверяем, есть ли participantId в URL
	if participantIDStr := chi.URLParam(r, "participantId"); participantIDStr != "" {
		participantID, err := strconv.ParseUint(participantIDStr, 10, 32)
		if err != nil {
			response.BadRequest(w, "Invalid participant ID")
			return
		}
		participantIDUint := uint(participantID)
		queryParams.ParticipantID = &participantIDUint
	}

	// Вызываем query handler
	assignments, err := h.getPrizeAssignmentsHandler.Handle(r.Context(), queryParams)
	if err != nil {
		log.Printf("Error getting prize assignments: %v", err)
		if err.Error() == "event_id or participant_id must be specified" {
			response.BadRequest(w, err.Error())
		} else {
			response.InternalServerError(w, "Failed to get prize assignments")
		}
		return
	}

	// Конвертируем в DTO
	assignmentDTOs := make([]*dto.PrizeAssignmentDTO, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentDTOs = append(assignmentDTOs, dto.FromPrizeAssignment(assignment))
	}

	// Возвращаем ответ
	response.Success(w, dto.PrizeAssignmentListResponse{
		PrizeAssignments: assignmentDTOs,
		Total:            len(assignmentDTOs),
	})
}

// GetByID обрабатывает GET /api/prize-assignments/:id - детали назначения приза
func (h *PrizeAssignmentsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid prize assignment ID")
		return
	}

	// Вызываем query handler
	assignment, err := h.getPrizeAssignmentByIDHandler.Handle(r.Context(), query.GetPrizeAssignmentByIDQuery{
		AssignmentID: uint(id),
	})
	if err != nil {
		log.Printf("Error getting prize assignment: %v", err)
		response.InternalServerError(w, "Failed to get prize assignment")
		return
	}

	if assignment == nil {
		response.NotFound(w, "Prize assignment not found")
		return
	}

	// Возвращаем DTO
	response.Success(w, dto.FromPrizeAssignment(assignment))
}

// CreateRequest представляет запрос на назначение приза
type CreatePrizeAssignmentRequest struct {
	ParticipantID uint   `json:"participant_id"`
	GiftID        uint   `json:"gift_id"`
	Comment       string `json:"comment,omitempty"`
}

// Create обрабатывает POST /api/prize-assignments - назначение приза
func (h *PrizeAssignmentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePrizeAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Вызываем command handler
	assignment, err := h.assignPrizeHandler.Handle(r.Context(), command.AssignPrizeCommand{
		ParticipantID: req.ParticipantID,
		GiftID:        req.GiftID,
		Comment:       req.Comment,
	})
	if err != nil {
		log.Printf("Error assigning prize: %v", err)
		switch err {
		case command.ErrParticipantNotFound:
			response.NotFound(w, err.Error())
		case command.ErrGiftNotFound:
			response.NotFound(w, err.Error())
		case command.ErrGiftAlreadyAssigned:
			response.Conflict(w, err.Error())
		default:
			if err.Error() == "participant is not registered for this event" {
				response.BadRequest(w, err.Error())
			} else {
				response.InternalServerError(w, "Failed to assign prize")
			}
		}
		return
	}

	// Возвращаем созданное назначение
	response.Created(w, dto.FromPrizeAssignment(assignment))
}

// Delete обрабатывает DELETE /api/prize-assignments/:id - удаление назначения приза
func (h *PrizeAssignmentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid prize assignment ID")
		return
	}

	// Проверяем существование назначения
	_, err = h.prizeAssignmentRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Prize assignment not found: %v", err)
		response.NotFound(w, "Prize assignment not found")
		return
	}

	// Удаляем назначение
	if err := h.prizeAssignmentRepo.Remove(r.Context(), uint(id)); err != nil {
		log.Printf("Error deleting prize assignment: %v", err)
		response.InternalServerError(w, "Failed to delete prize assignment")
		return
	}

	// Возвращаем успешный ответ без содержимого
	response.NoContent(w)
}
