package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
)

const maxEventGPXFileSize = 20 << 20 // 20 MiB

type eventFileStorage interface {
	SaveEventFile(ctx context.Context, eventID uint, originalName string, src io.Reader) (string, error)
}

// parseDate парсит дату в формате RFC3339 или YYYY-MM-DD
// Если передан формат YYYY-MM-DD, преобразует в начало дня в UTC
func parseDate(dateStr string) (time.Time, error) {
	// Пробуем RFC3339 формат (полный с временем)
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t, nil
	}

	// Пробуем RFC3339Nano формат
	if t, err := time.Parse(time.RFC3339Nano, dateStr); err == nil {
		return t, nil
	}

	// Пробуем формат только даты YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		// Преобразуем в начало дня в UTC
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}

	// Пробуем формат даты с временем без часового пояса YYYY-MM-DDTHH:MM:SS
	if t, err := time.Parse("2006-01-02T15:04:05", dateStr); err == nil {
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// EventsHandler обрабатывает запросы для событий
type EventsHandler struct {
	eventRepo           repository.EventRepository
	getEventsHandler    *query.GetEventsHandler
	getEventByIDHandler *query.GetEventByIDHandler
	createEventHandler  *command.CreateEventHandler
	updateEventHandler  *command.UpdateEventHandler
	eventFileStorage    eventFileStorage
}

// NewEventsHandler создаёт новый handler
func NewEventsHandler(
	eventRepo repository.EventRepository,
	getEventsHandler *query.GetEventsHandler,
	getEventByIDHandler *query.GetEventByIDHandler,
	createEventHandler *command.CreateEventHandler,
	updateEventHandler *command.UpdateEventHandler,
	eventFileStorage eventFileStorage,
) *EventsHandler {
	return &EventsHandler{
		eventRepo:           eventRepo,
		getEventsHandler:    getEventsHandler,
		getEventByIDHandler: getEventByIDHandler,
		createEventHandler:  createEventHandler,
		updateEventHandler:  updateEventHandler,
		eventFileStorage:    eventFileStorage,
	}
}

// GetAll обрабатывает GET /api/events - список всех событий
func (h *EventsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Парсим query параметр activeOnly
	activeOnly := false
	if r.URL.Query().Get("activeOnly") == "true" {
		activeOnly = true
	}

	// Вызываем query handler
	events, err := h.getEventsHandler.Handle(r.Context(), query.GetEventsQuery{
		ActiveOnly: activeOnly,
	})
	if err != nil {
		log.Printf("Error getting events: %v", err)
		response.InternalServerError(w, "Failed to get events")
		return
	}

	// Конвертируем в DTO
	eventDTOs := make([]*dto.EventDTO, 0, len(events))
	for _, event := range events {
		eventDTOs = append(eventDTOs, dto.FromEvent(event))
	}

	// Возвращаем ответ
	response.Success(w, dto.EventListResponse{
		Events: eventDTOs,
		Total:  len(eventDTOs),
	})
}

// GetByID обрабатывает GET /api/events/:id - детали события
func (h *EventsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Вызываем query handler
	event, err := h.getEventByIDHandler.Handle(r.Context(), query.GetEventByIDQuery{
		EventID: uint(id),
	})
	if err != nil {
		log.Printf("Error getting event: %v", err)
		response.InternalServerError(w, "Failed to get event")
		return
	}

	if event == nil {
		response.NotFound(w, "Event not found")
		return
	}

	// Возвращаем DTO
	response.Success(w, dto.FromEvent(event))
}

// CreateRequest представляет запрос на создание события
type CreateEventRequest struct {
	Name          string                    `json:"name"`
	Description   string                    `json:"description"`
	Active        bool                      `json:"active"`
	StartDate     *string                   `json:"start_date,omitempty"` // ISO 8601 format
	EndDate       *string                   `json:"end_date,omitempty"`   // ISO 8601 format
	GPXFilePath   string                    `json:"gpx_file_path,omitempty"`
	TelegramTexts entity.EventTelegramTexts `json:"telegram_texts,omitempty"`
}

// Create обрабатывает POST /api/events - создание события
func (h *EventsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Парсим даты (поддерживаем RFC3339 и формат даты YYYY-MM-DD)
	var startDate, endDate *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		parsed, err := parseDate(*req.StartDate)
		if err != nil {
			response.BadRequest(w, "Invalid start_date format. Use ISO 8601 (RFC3339) or YYYY-MM-DD")
			return
		}
		startDate = &parsed
	}

	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := parseDate(*req.EndDate)
		if err != nil {
			response.BadRequest(w, "Invalid end_date format. Use ISO 8601 (RFC3339) or YYYY-MM-DD")
			return
		}
		endDate = &parsed
	}

	// Вызываем command handler
	event, err := h.createEventHandler.Handle(r.Context(), command.CreateEventCommand{
		Name:          req.Name,
		Description:   req.Description,
		Active:        req.Active,
		StartDate:     startDate,
		EndDate:       endDate,
		GPXFilePath:   req.GPXFilePath,
		TelegramTexts: req.TelegramTexts,
	})
	if err != nil {
		log.Printf("Error creating event: %v", err)
		switch err {
		case command.ErrEventNameRequired:
			response.BadRequest(w, err.Error())
		case command.ErrEventNameExists:
			response.Conflict(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to create event")
		}
		return
	}

	// Возвращаем созданное событие
	response.Created(w, dto.FromEvent(event))
}

// UpdateRequest представляет запрос на обновление события
type UpdateEventRequest struct {
	Name          *string                    `json:"name,omitempty"`
	Description   *string                    `json:"description,omitempty"`
	Active        *bool                      `json:"active,omitempty"`
	StartDate     *string                    `json:"start_date,omitempty"` // ISO 8601 format
	EndDate       *string                    `json:"end_date,omitempty"`   // ISO 8601 format
	GPXFilePath   *string                    `json:"gpx_file_path,omitempty"`
	TelegramTexts *entity.EventTelegramTexts `json:"telegram_texts,omitempty"`
}

// Update обрабатывает PUT /api/events/:id - обновление события
func (h *EventsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Парсим даты (поддерживаем RFC3339 и формат даты YYYY-MM-DD)
	var startDate, endDate *time.Time
	if req.StartDate != nil && *req.StartDate != "" {
		parsed, err := parseDate(*req.StartDate)
		if err != nil {
			response.BadRequest(w, "Invalid start_date format. Use ISO 8601 (RFC3339) or YYYY-MM-DD")
			return
		}
		startDate = &parsed
	}

	if req.EndDate != nil && *req.EndDate != "" {
		parsed, err := parseDate(*req.EndDate)
		if err != nil {
			response.BadRequest(w, "Invalid end_date format. Use ISO 8601 (RFC3339) or YYYY-MM-DD")
			return
		}
		endDate = &parsed
	}

	// Вызываем command handler
	event, err := h.updateEventHandler.Handle(r.Context(), command.UpdateEventCommand{
		EventID:       uint(id),
		Name:          req.Name,
		Description:   req.Description,
		Active:        req.Active,
		StartDate:     startDate,
		EndDate:       endDate,
		GPXFilePath:   req.GPXFilePath,
		TelegramTexts: req.TelegramTexts,
	})
	if err != nil {
		log.Printf("Error updating event: %v", err)
		switch err {
		case command.ErrEventNotFound:
			response.NotFound(w, err.Error())
		case command.ErrEventNameExists:
			response.Conflict(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to update event")
		}
		return
	}

	// Возвращаем обновлённое событие
	response.Success(w, dto.FromEvent(event))
}

// UploadGPXFile обрабатывает POST /api/events/:id/gpx-file - загрузка GPX файла события
func (h *EventsHandler) UploadGPXFile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}
	eventID := uint(id)

	if h.eventFileStorage == nil {
		log.Printf("Event file storage is not configured")
		response.InternalServerError(w, "File storage is not configured")
		return
	}

	if _, err := h.eventRepo.FindByID(r.Context(), eventID); err != nil {
		log.Printf("Event not found for GPX upload: event_id=%d error=%v", eventID, err)
		response.NotFound(w, "Event not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxEventGPXFileSize)
	if err := r.ParseMultipartForm(maxEventGPXFileSize); err != nil {
		response.BadRequest(w, "Invalid multipart form or file is too large")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "GPX file is required")
		return
	}
	defer file.Close()

	fileName := strings.TrimSpace(header.Filename)
	if fileName == "" {
		response.BadRequest(w, "File name is required")
		return
	}
	if strings.ToLower(filepath.Ext(fileName)) != ".gpx" {
		response.BadRequest(w, "Only .gpx files are allowed")
		return
	}

	filePath, err := h.eventFileStorage.SaveEventFile(r.Context(), eventID, fileName, file)
	if err != nil {
		log.Printf("Error saving event GPX file: event_id=%d file=%q error=%v", eventID, fileName, err)
		response.InternalServerError(w, "Failed to save GPX file")
		return
	}

	event, err := h.updateEventHandler.Handle(r.Context(), command.UpdateEventCommand{
		EventID:     eventID,
		GPXFilePath: &filePath,
	})
	if err != nil {
		log.Printf("Error updating event GPX path: event_id=%d path=%q error=%v", eventID, filePath, err)
		switch err {
		case command.ErrEventNotFound:
			response.NotFound(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to update event GPX path")
		}
		return
	}

	response.Success(w, dto.FromEvent(event))
}

// Delete обрабатывает DELETE /api/events/:id - удаление события
func (h *EventsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Проверяем существование события
	_, err = h.eventRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Event not found: %v", err)
		response.NotFound(w, "Event not found")
		return
	}

	// Удаляем событие
	if err := h.eventRepo.Delete(r.Context(), uint(id)); err != nil {
		log.Printf("Error deleting event: %v", err)
		response.InternalServerError(w, "Failed to delete event")
		return
	}

	// Возвращаем успешный ответ без содержимого
	response.NoContent(w)
}
