package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/http/response"
)

// GiftsHandler обрабатывает запросы для подарков
type GiftsHandler struct {
	giftRepo           repository.GiftRepository
	getGiftsHandler    *query.GetGiftsHandler
	getGiftByIDHandler *query.GetGiftByIDHandler
	updateGiftHandler  *command.UpdateGiftHandler
}

// NewGiftsHandler создаёт новый handler
func NewGiftsHandler(
	giftRepo repository.GiftRepository,
	getGiftsHandler *query.GetGiftsHandler,
	getGiftByIDHandler *query.GetGiftByIDHandler,
	updateGiftHandler *command.UpdateGiftHandler,
) *GiftsHandler {
	return &GiftsHandler{
		giftRepo:           giftRepo,
		getGiftsHandler:    getGiftsHandler,
		getGiftByIDHandler: getGiftByIDHandler,
		updateGiftHandler:  updateGiftHandler,
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

	var reviewStatus *entity.GiftReviewStatus
	reviewStatusParam := r.URL.Query().Get("review_status")
	if reviewStatusParam != "" {
		status, err := entity.NewGiftReviewStatus(reviewStatusParam)
		if err != nil {
			log.Printf("level=warn msg=\"Invalid gift review status filter\" event_id=%d review_status=%s", eventID, reviewStatusParam)
			response.BadRequest(w, "Invalid review_status")
			return
		}
		reviewStatus = &status
	}

	// Парсим query параметры для фильтров
	queryParams := query.GetGiftsQuery{
		EventID:      uint(eventID),
		ReviewStatus: reviewStatus,
	}

	// Вызываем query handler
	gifts, err := h.getGiftsHandler.Handle(r.Context(), queryParams)
	if err != nil {
		log.Printf("Error getting gifts: event_id=%d review_status=%s error=%v", eventID, reviewStatusParam, err)
		if errors.Is(err, query.ErrInvalidGiftReviewStatusFilter) {
			response.BadRequest(w, "Invalid review_status")
			return
		}
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

// UpdateGiftRequest представляет запрос на обновление подарка
type UpdateGiftRequest struct {
	Description    *string
	GenderFilter   *string
	BikeTypeFilter *string
	ReviewStatus   *string
	Place          *int
	PlaceSet       bool
	PlaceRule      valueobject.GiftPlaceRule
	PlaceRuleSet   bool
	CriteriaIDs    []uint
	CriteriaIDsSet bool
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

	req, err := decodeUpdateGiftRequest(r)
	if err != nil {
		var placeRuleErr invalidPlaceRulePayloadError
		if errors.As(err, &placeRuleErr) {
			log.Printf("level=warn msg=\"Invalid gift place_rule payload\" gift_id=%d rule_type=%s reason=%s", id, placeRuleErr.ruleType, placeRuleErr.reason)
		}
		response.BadRequest(w, "Invalid request body")
		return
	}

	_, err = h.updateGiftHandler.Handle(r.Context(), command.UpdateGiftCommand{
		GiftID:         uint(id),
		Description:    req.Description,
		GenderFilter:   req.GenderFilter,
		BikeTypeFilter: req.BikeTypeFilter,
		ReviewStatus:   req.ReviewStatus,
		Place:          req.Place,
		PlaceSet:       req.PlaceSet,
		PlaceRule:      req.PlaceRule,
		PlaceRuleSet:   req.PlaceRuleSet,
		CriteriaIDs:    req.CriteriaIDs,
		CriteriaIDsSet: req.CriteriaIDsSet,
	})
	if err != nil {
		log.Printf("Error updating gift: gift_id=%d error=%v", id, err)
		switch {
		case errors.Is(err, command.ErrGiftNotFound):
			response.NotFound(w, "Gift not found")
		case errors.Is(err, command.ErrEmptyDescription),
			errors.Is(err, command.ErrInvalidGiftGenderFilter),
			errors.Is(err, command.ErrInvalidGiftBikeTypeFilter),
			errors.Is(err, command.ErrInvalidGiftReviewStatus),
			errors.Is(err, command.ErrInvalidGiftPlace),
			errors.Is(err, command.ErrInvalidGiftPlaceRule),
			errors.Is(err, command.ErrGiftCriteriaPayloadRequired):
			response.BadRequest(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to update gift")
		}
		return
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

func decodeUpdateGiftRequest(r *http.Request) (UpdateGiftRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return UpdateGiftRequest{}, err
	}

	req := UpdateGiftRequest{}
	if value, ok := raw["description"]; ok {
		var description string
		if err := json.Unmarshal(value, &description); err != nil {
			return UpdateGiftRequest{}, err
		}
		req.Description = &description
	}

	if value, ok := raw["gender_filter"]; ok {
		var genderFilter string
		if err := json.Unmarshal(value, &genderFilter); err != nil {
			return UpdateGiftRequest{}, err
		}
		req.GenderFilter = &genderFilter
	}

	if value, ok := raw["bike_type_filter"]; ok {
		var bikeTypeFilter string
		if err := json.Unmarshal(value, &bikeTypeFilter); err != nil {
			return UpdateGiftRequest{}, err
		}
		req.BikeTypeFilter = &bikeTypeFilter
	}

	if value, ok := raw["review_status"]; ok {
		var reviewStatus string
		if err := json.Unmarshal(value, &reviewStatus); err != nil {
			return UpdateGiftRequest{}, err
		}
		req.ReviewStatus = &reviewStatus
	}

	if value, ok := raw["place"]; ok {
		req.PlaceSet = true
		if !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			var place int
			if err := json.Unmarshal(value, &place); err != nil {
				return UpdateGiftRequest{}, err
			}
			req.Place = &place
		}
	}

	if value, ok := raw["place_rule"]; ok {
		req.PlaceRuleSet = true
		placeRule, err := decodeGiftPlaceRule(value)
		if err != nil {
			return UpdateGiftRequest{}, err
		}
		req.PlaceRule = placeRule
	}

	if value, ok := raw["criteria_ids"]; ok {
		var criteriaIDs []uint
		if err := json.Unmarshal(value, &criteriaIDs); err != nil {
			return UpdateGiftRequest{}, err
		}
		req.CriteriaIDs = criteriaIDs
		req.CriteriaIDsSet = true
	}

	return req, nil
}

type giftPlaceRulePayload struct {
	Type      string `json:"type"`
	Places    []int  `json:"places"`
	LastCount *int   `json:"last_count"`
}

type invalidPlaceRulePayloadError struct {
	ruleType string
	reason   string
}

func (e invalidPlaceRulePayloadError) Error() string {
	return "invalid gift place_rule payload: " + e.reason
}

func decodeGiftPlaceRule(raw json.RawMessage) (valueobject.GiftPlaceRule, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return valueobject.NewGiftPlaceRuleNone(), nil
	}

	var payload giftPlaceRulePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return valueobject.GiftPlaceRule{}, invalidPlaceRulePayloadError{reason: "invalid_json"}
	}

	switch payload.Type {
	case string(valueobject.GiftPlaceRuleTypeNone):
		return valueobject.NewGiftPlaceRuleNone(), nil
	case string(valueobject.GiftPlaceRuleTypePlaces):
		rule, err := valueobject.NewGiftPlaceRulePlaces(payload.Places)
		if err != nil {
			return valueobject.GiftPlaceRule{}, invalidPlaceRulePayloadError{ruleType: payload.Type, reason: err.Error()}
		}
		return rule, nil
	case string(valueobject.GiftPlaceRuleTypeLastN):
		if payload.LastCount == nil {
			return valueobject.GiftPlaceRule{}, invalidPlaceRulePayloadError{ruleType: payload.Type, reason: "missing_last_count"}
		}
		rule, err := valueobject.NewGiftPlaceRuleLastN(*payload.LastCount)
		if err != nil {
			return valueobject.GiftPlaceRule{}, invalidPlaceRulePayloadError{ruleType: payload.Type, reason: err.Error()}
		}
		return rule, nil
	default:
		return valueobject.GiftPlaceRule{}, invalidPlaceRulePayloadError{ruleType: payload.Type, reason: "unsupported_rule_type"}
	}
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
