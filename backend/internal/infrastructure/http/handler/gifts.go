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

// GiftsHandler обрабатывает запросы для подарков
type GiftsHandler struct {
	giftRepo           repository.GiftRepository
	giftCriteriaRepo   repository.GiftCriteriaRepository
	criteriaRepo       repository.CriteriaRepository
	getGiftsHandler    *query.GetGiftsHandler
	getGiftByIDHandler *query.GetGiftByIDHandler
	addGiftHandler     *command.AddGiftHandler
}

// NewGiftsHandler создаёт новый handler
func NewGiftsHandler(
	giftRepo repository.GiftRepository,
	giftCriteriaRepo repository.GiftCriteriaRepository,
	criteriaRepo repository.CriteriaRepository,
	getGiftsHandler *query.GetGiftsHandler,
	getGiftByIDHandler *query.GetGiftByIDHandler,
	addGiftHandler *command.AddGiftHandler,
) *GiftsHandler {
	return &GiftsHandler{
		giftRepo:           giftRepo,
		giftCriteriaRepo:   giftCriteriaRepo,
		criteriaRepo:       criteriaRepo,
		getGiftsHandler:    getGiftsHandler,
		getGiftByIDHandler: getGiftByIDHandler,
		addGiftHandler:     addGiftHandler,
	}
}

// GetAll обрабатывает GET /api/events/:eventId/gifts - список подарков события
func (h *GiftsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Извлекаем eventID из URL
	eventIDStr := chi.URLParam(r, "eventId")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Парсим query параметры для фильтров
	queryParams := query.GetGiftsQuery{
		EventID: uint(eventID),
	}

	// Вызываем query handler
	gifts, err := h.getGiftsHandler.Handle(r.Context(), queryParams)
	if err != nil {
		log.Printf("Error getting gifts: %v", err)
		response.InternalServerError(w, "Failed to get gifts")
		return
	}

	// Конвертируем в DTO
	giftDTOs := make([]*dto.GiftDTO, 0, len(gifts))
	for _, gift := range gifts {
		giftDTOs = append(giftDTOs, dto.FromGift(gift))
	}

	// Возвращаем ответ
	response.Success(w, dto.GiftListResponse{
		Gifts: giftDTOs,
		Total: len(giftDTOs),
	})
}

// GetByID обрабатывает GET /api/gifts/:id - детали подарка
func (h *GiftsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid gift ID")
		return
	}

	// Вызываем query handler
	gift, err := h.getGiftByIDHandler.Handle(r.Context(), query.GetGiftByIDQuery{
		GiftID: uint(id),
	})
	if err != nil {
		log.Printf("Error getting gift: %v", err)
		response.InternalServerError(w, "Failed to get gift")
		return
	}

	if gift == nil {
		response.NotFound(w, "Gift not found")
		return
	}

	// Возвращаем DTO
	response.Success(w, dto.FromGift(gift))
}

// CreateRequest представляет запрос на создание подарка
type CreateGiftRequest struct {
	UserID      int64                   `json:"user_id"`
	Description string                  `json:"description"`
	Attachments []GiftAttachmentRequest `json:"attachments,omitempty"`
}

// GiftAttachmentRequest представляет запрос на прикрепление файла
type GiftAttachmentRequest struct {
	TelegramFileID string `json:"telegram_file_id"`
	FileType       string `json:"file_type"` // photo, document
}

// Create обрабатывает POST /api/events/:eventId/gifts - создание подарка
func (h *GiftsHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Извлекаем eventID из URL
	eventIDStr := chi.URLParam(r, "eventId")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	var req CreateGiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Конвертируем attachments
	attachments := make([]command.GiftAttachmentData, 0, len(req.Attachments))
	for _, attachReq := range req.Attachments {
		attachments = append(attachments, command.GiftAttachmentData{
			TelegramFileID: attachReq.TelegramFileID,
			FileType:       attachReq.FileType,
		})
	}

	// Вызываем command handler
	gift, err := h.addGiftHandler.Handle(r.Context(), command.AddGiftCommand{
		UserID:      req.UserID,
		EventID:     uint(eventID),
		Description: req.Description,
		Attachments: attachments,
	})
	if err != nil {
		log.Printf("Error creating gift: %v", err)
		switch err {
		case command.ErrUserNotFound:
			response.NotFound(w, err.Error())
		case command.ErrEventNotFound:
			response.NotFound(w, err.Error())
		case command.ErrEmptyDescription:
			response.BadRequest(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to create gift")
		}
		return
	}

	// Возвращаем созданный подарок
	response.Created(w, dto.FromGift(gift))
}

// UpdateGiftRequest представляет запрос на обновление подарка
type UpdateGiftRequest struct {
	Description    string  `json:"description"`
	GenderFilter   string  `json:"gender_filter"`   // all, male, female
	BikeTypeFilter string  `json:"bike_type_filter"` // all, gravel, mtb, road, single_speed, tandem
	Place          *int    `json:"place"`
	CriteriaIDs    []uint  `json:"criteria_ids"`
}

// Update обрабатывает PUT /api/gifts/:id - обновление подарка
func (h *GiftsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid gift ID")
		return
	}

	var req UpdateGiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Проверяем существование подарка
	gift, err := h.giftRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Gift not found: %v", err)
		response.NotFound(w, "Gift not found")
		return
	}

	// Обновляем поля подарка
	needsUpdate := false
	if req.Description != "" {
		gift.Description = req.Description
		needsUpdate = true
	}
	if req.GenderFilter != "" {
		// Нормализуем пустую строку в "all"
		if req.GenderFilter == "" {
			gift.GenderFilter = "all"
		} else {
			gift.GenderFilter = req.GenderFilter
		}
		needsUpdate = true
	}
	if req.BikeTypeFilter != "" {
		// Нормализуем пустую строку в "all"
		if req.BikeTypeFilter == "" {
			gift.BikeTypeFilter = "all"
		} else {
			gift.BikeTypeFilter = req.BikeTypeFilter
		}
		needsUpdate = true
	}
	if req.Place != nil {
		gift.Place = req.Place
		needsUpdate = true
	}

	if needsUpdate {
		if err := h.giftRepo.Update(r.Context(), gift); err != nil {
			log.Printf("Error updating gift: %v", err)
			response.InternalServerError(w, "Failed to update gift")
			return
		}
	}

	// Обновляем критерии если переданы
	if req.CriteriaIDs != nil {
		// Удаляем все старые критерии
		if err := h.giftCriteriaRepo.RemoveAllCriteriaFromGift(r.Context(), uint(id)); err != nil {
			log.Printf("Error removing criteria from gift: %v", err)
			response.InternalServerError(w, "Failed to update gift criteria")
			return
		}

		// Добавляем новые критерии
		for _, criteriaID := range req.CriteriaIDs {
			if err := h.giftCriteriaRepo.AddCriteriaToGift(r.Context(), uint(id), criteriaID); err != nil {
				log.Printf("Error adding criteria to gift: %v", err)
				response.InternalServerError(w, "Failed to update gift criteria")
				return
			}
		}
	}

	// Загружаем обновлённый подарок с критериями
	updatedGift, err := h.getGiftByIDHandler.Handle(r.Context(), query.GetGiftByIDQuery{
		GiftID: uint(id),
	})
	if err != nil {
		log.Printf("Error getting updated gift: %v", err)
		response.InternalServerError(w, "Failed to get updated gift")
		return
	}

	response.Success(w, dto.FromGift(updatedGift))
}

// Delete обрабатывает DELETE /api/gifts/:id - удаление подарка
func (h *GiftsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid gift ID")
		return
	}

	// Проверяем существование подарка
	_, err = h.giftRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Gift not found: %v", err)
		response.NotFound(w, "Gift not found")
		return
	}

	// Удаляем подарок
	if err := h.giftRepo.Delete(r.Context(), uint(id)); err != nil {
		log.Printf("Error deleting gift: %v", err)
		response.InternalServerError(w, "Failed to delete gift")
		return
	}

	// Возвращаем успешный ответ без содержимого
	response.NoContent(w)
}
