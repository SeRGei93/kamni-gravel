package handler

import (
	"bytes"
	"context"
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
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/http/response"
)

// GiftsHandler обрабатывает запросы для подарков
type GiftsHandler struct {
	giftRepo              repository.GiftRepository
	getGiftsHandler       *query.GetGiftsHandler
	getGiftByIDHandler    *query.GetGiftByIDHandler
	getManualGiftsHandler *query.GetManualGiftsHandler
	addGiftHandler        *command.AddGiftHandler
	updateGiftHandler     *command.UpdateGiftHandler
	assignRandomRecipient randomGiftRecipientAssigner
	publicGiftNotifier    GiftPublicationNotifier
	giftsCache            miniappGiftsCacheInvalidator
}

type randomGiftRecipientAssigner interface {
	Handle(ctx context.Context, cmd command.AssignRandomAdminGiftRecipientCommand) (*command.AssignRandomAdminGiftRecipientResult, error)
}

// GiftPublicationNotifier отправляет опубликованный приз в Telegram-чат.
type GiftPublicationNotifier interface {
	NotifyWithRetry(ctx context.Context, gift *entity.Gift) error
}

// miniappGiftsCacheInvalidator сбрасывает кеш каталога подарков мини-приложения для события.
// Реализация может быть nil — тогда инвалидация пропускается.
type miniappGiftsCacheInvalidator interface {
	InvalidateEvent(eventID uint)
}

// NewGiftsHandler создаёт новый handler.
// giftsCache может быть nil — тогда кеш каталога мини-приложения не инвалидируется.
func NewGiftsHandler(
	giftRepo repository.GiftRepository,
	getGiftsHandler *query.GetGiftsHandler,
	getGiftByIDHandler *query.GetGiftByIDHandler,
	getManualGiftsHandler *query.GetManualGiftsHandler,
	addGiftHandler *command.AddGiftHandler,
	updateGiftHandler *command.UpdateGiftHandler,
	assignRandomRecipient randomGiftRecipientAssigner,
	giftsCache miniappGiftsCacheInvalidator,
	publicGiftNotifier ...GiftPublicationNotifier,
) *GiftsHandler {
	var notifier GiftPublicationNotifier
	if len(publicGiftNotifier) > 0 {
		notifier = publicGiftNotifier[0]
	}

	return &GiftsHandler{
		giftRepo:              giftRepo,
		getGiftsHandler:       getGiftsHandler,
		getGiftByIDHandler:    getGiftByIDHandler,
		getManualGiftsHandler: getManualGiftsHandler,
		addGiftHandler:        addGiftHandler,
		updateGiftHandler:     updateGiftHandler,
		assignRandomRecipient: assignRandomRecipient,
		publicGiftNotifier:    notifier,
		giftsCache:            giftsCache,
	}
}

// AssignRandomRecipient handles POST /api/gifts/{id}/random-recipient.
func (h *GiftsHandler) AssignRandomRecipient(w http.ResponseWriter, r *http.Request) {
	if h.assignRandomRecipient == nil {
		log.Printf("ERROR admin random gift recipient assignment unavailable")
		response.InternalServerError(w, "Random gift assignment is unavailable")
		return
	}

	giftID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil || giftID == 0 {
		log.Printf("WARN admin random gift recipient assignment rejected: reason=invalid_gift_id")
		response.BadRequest(w, "Invalid gift ID")
		return
	}

	result, err := h.assignRandomRecipient.Handle(r.Context(), command.AssignRandomAdminGiftRecipientCommand{GiftID: uint(giftID)})
	if err != nil {
		switch {
		case errors.Is(err, command.ErrGiftNotFound), errors.Is(err, command.ErrManualGiftRecipientNotFound):
			log.Printf("WARN admin random gift recipient assignment rejected: gift_id=%d reason=not_found", giftID)
			response.NotFound(w, "Gift or participant not found")
		case errors.Is(err, command.ErrAdminRandomGiftNotApproved),
			errors.Is(err, command.ErrAdminRandomGiftAlreadyAssigned),
			errors.Is(err, command.ErrManualGiftNoUnawardedParticipants),
			errors.Is(err, command.ErrManualGiftRecipientEvent),
			errors.Is(err, command.ErrManualGiftRecipientIneligible):
			log.Printf("WARN admin random gift recipient assignment rejected: gift_id=%d reason=conflict error=%v", giftID, err)
			response.Conflict(w, err.Error())
		default:
			log.Printf("ERROR admin random gift recipient assignment failed: gift_id=%d error=%v", giftID, err)
			response.InternalServerError(w, "Failed to assign random gift recipient")
		}
		return
	}
	if result == nil {
		log.Printf("ERROR admin random gift recipient assignment failed: gift_id=%d reason=empty_result", giftID)
		response.InternalServerError(w, "Failed to assign random gift recipient")
		return
	}

	if result.BecameManual {
		h.invalidateMiniappGiftsCache(result.EventID)
	}
	adminID := uint(0)
	if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
		adminID = claims.UserID
	}
	log.Printf("INFO admin random gift recipient assignment completed: admin_id=%d gift_id=%d event_id=%d recipient_participant_id=%d converted_to_manual=%t", adminID, result.GiftID, result.EventID, result.RecipientParticipantID, result.BecameManual)
	response.NoContent(w)
}

// GetManualByEvent returns protected recipient summaries for manual gifts.
func (h *GiftsHandler) GetManualByEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := parseEventIDParam(r)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}
	if h.getManualGiftsHandler == nil {
		log.Printf("ERROR manual gifts admin query unavailable: event_id=%d", eventID)
		response.InternalServerError(w, "Manual gift management is unavailable")
		return
	}

	manualGifts, err := h.getManualGiftsHandler.Handle(r.Context(), query.GetManualGiftsQuery{EventID: eventID})
	if err != nil {
		log.Printf("ERROR manual gifts admin query failed: event_id=%d error=%v", eventID, err)
		response.InternalServerError(w, "Failed to get manual gifts")
		return
	}

	gifts := make([]*dto.ManualGiftDTO, 0, len(manualGifts))
	for _, gift := range manualGifts {
		gifts = append(gifts, manualGiftDTOFromReadModel(gift))
	}
	log.Printf("DEBUG manual gifts admin query served: event_id=%d returned_count=%d", eventID, len(gifts))
	response.Success(w, dto.ManualGiftListResponse{Gifts: gifts})
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

	var ownerUserID *int64
	if rawOwnerUserID := strings.TrimSpace(r.URL.Query().Get("owner_user_id")); rawOwnerUserID != "" {
		parsedOwnerUserID, err := strconv.ParseInt(rawOwnerUserID, 10, 64)
		if err != nil || parsedOwnerUserID <= 0 {
			response.BadRequest(w, "Invalid owner_user_id")
			return
		}
		ownerUserID = &parsedOwnerUserID
	}

	// Пагинация включается только если переданы page/page_size. Без них возвращаем
	// все подарки (нужно для модалок выбора приза и подсчёта назначений).
	page := ParsePageParams(r)
	paginate := (r.URL.Query().Has("page") || r.URL.Query().Has("page_size")) && !page.All

	// Парсим query параметры для фильтров
	queryParams := query.GetGiftsQuery{
		EventID:      uint(eventID),
		ReviewStatus: reviewStatus,
		OwnerUserID:  ownerUserID,
		SearchQuery:  strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if paginate {
		queryParams.Limit = page.Limit
		queryParams.Offset = page.Offset
	}

	// Вызываем query handler
	gifts, total, err := h.getGiftsHandler.Handle(r.Context(), queryParams)
	if err != nil {
		log.Printf("Error getting gifts: event_id=%d review_status=%s owner_user_id=%v q=%q error=%v", eventID, reviewStatusParam, ownerUserID, queryParams.SearchQuery, err)
		if errors.Is(err, query.ErrInvalidGiftReviewStatusFilter) {
			response.BadRequest(w, "Invalid review_status")
			return
		}
		response.InternalServerError(w, "Failed to get gifts")
		return
	}

	// Считаем количество подарков по статусам для бейджей вкладок (по всему событию).
	statusCounts, err := h.giftRepo.CountsByReviewStatus(r.Context(), uint(eventID))
	if err != nil {
		log.Printf("Error counting gifts by review status: event_id=%d error=%v", eventID, err)
		response.InternalServerError(w, "Failed to get gifts")
		return
	}

	// Конвертируем в DTO
	giftDTOs := make([]*dto.GiftDTO, 0, len(gifts))
	for _, gift := range gifts {
		giftDTOs = append(giftDTOs, dto.FromGift(gift))
	}

	resp := dto.GiftListResponse{
		Gifts:        giftDTOs,
		Total:        total,
		StatusCounts: statusCounts,
	}
	if paginate {
		resp.Page = page.Page
		resp.PageSize = page.PageSize
	}

	log.Printf("DEBUG Gifts list served: event_id=%d review_status=%s owner_user_id=%v q=%q paginated=%t total=%d page=%d page_size=%d returned=%d status_counts=%v",
		eventID, reviewStatusParam, ownerUserID, queryParams.SearchQuery, paginate, total, page.Page, page.PageSize, len(giftDTOs), statusCounts)

	// Возвращаем ответ
	response.Success(w, resp)
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

// CreateGiftRequest представляет запрос на ручное добавление подарка от имени Telegram-пользователя.
type CreateGiftRequest struct {
	UserID         int64  `json:"user_id"`
	Description    string `json:"description"`
	GenderFilter   string `json:"gender_filter"`
	BikeTypeFilter string `json:"bike_type_filter"`
}

// Create обрабатывает POST /api/events/:eventId/gifts - ручное добавление подарка администратором.
func (h *GiftsHandler) Create(w http.ResponseWriter, r *http.Request) {
	eventIDStr := chi.URLParam(r, "eventId")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil || eventID == 0 {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	var req CreateGiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	req.Description = strings.TrimSpace(req.Description)
	if req.UserID <= 0 {
		log.Printf("WARN Manual gift creation rejected: event_id=%d telegram_user_id=%d error_class=missing_user_id", eventID, req.UserID)
		response.BadRequest(w, "user_id is required")
		return
	}
	if req.Description == "" {
		log.Printf("WARN Manual gift creation rejected: event_id=%d telegram_user_id=%d error_class=missing_description", eventID, req.UserID)
		response.BadRequest(w, "description is required")
		return
	}

	gift, err := h.addGiftHandler.Handle(r.Context(), command.AddGiftCommand{
		UserID:         req.UserID,
		EventID:        uint(eventID),
		Description:    req.Description,
		GenderFilter:   req.GenderFilter,
		BikeTypeFilter: req.BikeTypeFilter,
	})
	if err != nil {
		errorClass := giftCreateErrorClass(err)
		switch {
		case errors.Is(err, command.ErrUserBlacklisted):
			log.Printf("WARN Manual gift creation rejected: event_id=%d telegram_user_id=%d error_class=%s", eventID, req.UserID, errorClass)
			response.Forbidden(w, err.Error())
		case errors.Is(err, command.ErrUserNotFound), errors.Is(err, command.ErrEventNotFound):
			log.Printf("WARN Manual gift creation rejected: event_id=%d telegram_user_id=%d error_class=%s", eventID, req.UserID, errorClass)
			response.NotFound(w, err.Error())
		case errors.Is(err, command.ErrEmptyDescription),
			errors.Is(err, command.ErrInvalidGiftGenderFilter),
			errors.Is(err, command.ErrInvalidGiftBikeTypeFilter):
			log.Printf("WARN Manual gift creation rejected: event_id=%d telegram_user_id=%d error_class=%s", eventID, req.UserID, errorClass)
			response.BadRequest(w, err.Error())
		default:
			log.Printf("ERROR Manual gift creation failed: event_id=%d telegram_user_id=%d error_class=%s error=%v", eventID, req.UserID, errorClass, err)
			response.InternalServerError(w, "Failed to create gift")
		}
		return
	}

	response.Created(w, dto.FromGift(gift))
}

func giftCreateErrorClass(err error) string {
	switch {
	case errors.Is(err, command.ErrUserBlacklisted):
		return "user_blacklisted"
	case errors.Is(err, command.ErrUserNotFound):
		return "user_not_found"
	case errors.Is(err, command.ErrEventNotFound):
		return "event_not_found"
	case errors.Is(err, command.ErrEmptyDescription):
		return "missing_description"
	case errors.Is(err, command.ErrInvalidGiftGenderFilter):
		return "invalid_gender_filter"
	case errors.Is(err, command.ErrInvalidGiftBikeTypeFilter):
		return "invalid_bike_type_filter"
	default:
		return "unknown"
	}
}

// UpdateGiftRequest представляет запрос на обновление подарка
type UpdateGiftRequest struct {
	Description                     *string
	GenderFilter                    *string
	BikeTypeFilter                  *string
	ReviewStatus                    *string
	Place                           *int
	PlaceSet                        bool
	PlaceRule                       valueobject.GiftPlaceRule
	PlaceRuleSet                    bool
	CriteriaIDs                     []uint
	CriteriaIDsSet                  bool
	ManualDistribution              *bool
	ManualRecipientParticipantID    *uint
	ManualRecipientParticipantIDSet bool
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

	updateResult, err := h.updateGiftHandler.Handle(r.Context(), command.UpdateGiftCommand{
		GiftID:                          uint(id),
		Description:                     req.Description,
		GenderFilter:                    req.GenderFilter,
		BikeTypeFilter:                  req.BikeTypeFilter,
		ReviewStatus:                    req.ReviewStatus,
		Place:                           req.Place,
		PlaceSet:                        req.PlaceSet,
		PlaceRule:                       req.PlaceRule,
		PlaceRuleSet:                    req.PlaceRuleSet,
		CriteriaIDs:                     req.CriteriaIDs,
		CriteriaIDsSet:                  req.CriteriaIDsSet,
		ManualDistribution:              req.ManualDistribution,
		ManualRecipientParticipantID:    req.ManualRecipientParticipantID,
		ManualRecipientParticipantIDSet: req.ManualRecipientParticipantIDSet,
	})
	if err != nil {
		log.Printf("Error updating gift: gift_id=%d error=%v", id, err)
		switch {
		case errors.Is(err, command.ErrGiftNotFound):
			response.NotFound(w, "Gift not found")
		case errors.Is(err, command.ErrManualGiftRecipientNotFound):
			response.NotFound(w, "Manual gift recipient participant not found")
		case errors.Is(err, command.ErrEmptyDescription),
			errors.Is(err, command.ErrInvalidGiftGenderFilter),
			errors.Is(err, command.ErrInvalidGiftBikeTypeFilter),
			errors.Is(err, command.ErrInvalidGiftReviewStatus),
			errors.Is(err, command.ErrInvalidGiftPlace),
			errors.Is(err, command.ErrInvalidGiftPlaceRule),
			errors.Is(err, command.ErrGiftCriteriaPayloadRequired),
			errors.Is(err, command.ErrManualGiftRecipientConflict):
			response.BadRequest(w, err.Error())
		case errors.Is(err, command.ErrManualGiftNotManual),
			errors.Is(err, command.ErrManualGiftRecipientEvent),
			errors.Is(err, command.ErrManualGiftRecipientIneligible):
			response.Conflict(w, err.Error())
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

	// Сбрасываем кеш каталога мини-приложения, если подарок одобрен на любом конце
	// перехода: одобрение (pending→approved), правка уже одобренного (approved→approved)
	// и снятие одобрения (approved→pending). Иначе кеш отдавал бы устаревший каталог.
	if updatedGift != nil && updateResult != nil && !updateChangesOnlyManualRecipient(req) &&
		(updatedGift.ReviewStatus == entity.GiftReviewStatusApproved ||
			updateResult.PreviousReviewStatus == entity.GiftReviewStatusApproved) {
		h.invalidateMiniappGiftsCache(updatedGift.EventID)
	}

	if updateResult != nil && updateResult.BecameApproved {
		h.notifyPublicGiftApproved(r.Context(), updatedGift)
	}
	if req.ManualDistribution != nil || req.ManualRecipientParticipantIDSet {
		adminID := uint(0)
		if claims, ok := middleware.GetUserFromContext(r.Context()); ok {
			adminID = claims.UserID
		}
		log.Printf("INFO manual gift admin update completed: admin_id=%d gift_id=%d recipient_participant_id=%s", adminID, id, manualGiftRecipientIDLogValue(req.ManualRecipientParticipantID))
	}

	response.Success(w, dto.FromGift(updatedGift))
}

// invalidateMiniappGiftsCache сбрасывает кеш каталога мини-приложения для события, если кеш включён.
func (h *GiftsHandler) invalidateMiniappGiftsCache(eventID uint) {
	if h.giftsCache == nil {
		return
	}
	h.giftsCache.InvalidateEvent(eventID)
}

func (h *GiftsHandler) notifyPublicGiftApproved(ctx context.Context, gift *entity.Gift) {
	if h.publicGiftNotifier == nil || gift == nil {
		return
	}

	notifyCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	if err := h.publicGiftNotifier.NotifyWithRetry(notifyCtx, gift); err != nil {
		log.Printf("WARN Public gift notification failed after approval: gift_id=%d event_id=%d user_id=%d chat=public error=%v", gift.ID, gift.EventID, gift.UserID, err)
	}
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

	if value, ok := raw["manual_distribution"]; ok {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return UpdateGiftRequest{}, errors.New("manual_distribution must be boolean")
		}
		var manualDistribution bool
		if err := json.Unmarshal(value, &manualDistribution); err != nil {
			return UpdateGiftRequest{}, err
		}
		req.ManualDistribution = &manualDistribution
	}

	if value, ok := raw["manual_recipient_participant_id"]; ok {
		req.ManualRecipientParticipantIDSet = true
		if !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			var participantID uint
			if err := json.Unmarshal(value, &participantID); err != nil || participantID == 0 {
				if err != nil {
					return UpdateGiftRequest{}, err
				}
				return UpdateGiftRequest{}, errors.New("manual_recipient_participant_id must be positive")
			}
			req.ManualRecipientParticipantID = &participantID
		}
	}

	return req, nil
}

func parseEventIDParam(r *http.Request) (uint, error) {
	eventID, err := strconv.ParseUint(chi.URLParam(r, "eventId"), 10, 32)
	if err != nil || eventID == 0 {
		return 0, errors.New("invalid event ID")
	}
	return uint(eventID), nil
}

func updateChangesOnlyManualRecipient(req UpdateGiftRequest) bool {
	return req.ManualDistribution == nil &&
		req.ManualRecipientParticipantIDSet &&
		req.Description == nil &&
		req.GenderFilter == nil &&
		req.BikeTypeFilter == nil &&
		req.ReviewStatus == nil &&
		!req.PlaceSet &&
		!req.PlaceRuleSet &&
		!req.CriteriaIDsSet
}

func manualGiftDTOFromReadModel(model *query.ManualGiftReadModel) *dto.ManualGiftDTO {
	if model == nil {
		return nil
	}
	dtoModel := &dto.ManualGiftDTO{
		ID:                 model.ID,
		EventID:            model.EventID,
		Description:        model.Description,
		GenderFilter:       model.GenderFilter,
		BikeTypeFilter:     model.BikeTypeFilter,
		ReviewStatus:       model.ReviewStatus,
		ManualDistribution: model.ManualDistribution,
		Place:              model.Place,
		PlaceRule:          dto.FromGiftPlaceRule(model.PlaceRule),
		CreatedAt:          model.CreatedAt,
	}
	if len(model.Attachments) > 0 {
		dtoModel.Attachments = make([]*dto.GiftAttachmentDTO, len(model.Attachments))
		for index, attachment := range model.Attachments {
			dtoModel.Attachments[index] = &dto.GiftAttachmentDTO{
				ID:             attachment.ID,
				GiftID:         attachment.GiftID,
				TelegramFileID: attachment.TelegramFileID,
				FileType:       attachment.FileType,
			}
		}
	}
	if len(model.Criteria) > 0 {
		dtoModel.Criteria = make([]*dto.CriteriaDTO, len(model.Criteria))
		for index, criteria := range model.Criteria {
			dtoModel.Criteria[index] = dto.FromCriteria(criteria)
		}
	}
	if model.Recipient != nil {
		dtoModel.Recipient = &dto.ManualGiftRecipientDTO{
			ID:          model.Recipient.ID,
			DisplayName: model.Recipient.DisplayName,
			Username:    model.Recipient.Username,
			Status:      model.Recipient.Status,
		}
	}
	return dtoModel
}

func manualGiftRecipientIDLogValue(participantID *uint) string {
	if participantID == nil {
		return "none"
	}
	return strconv.FormatUint(uint64(*participantID), 10)
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
	gift, err := h.giftRepo.FindByID(r.Context(), uint(id))
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

	// Удаление одобренного подарка меняет публичный каталог — сбрасываем кеш события.
	if gift != nil && gift.ReviewStatus == entity.GiftReviewStatusApproved {
		h.invalidateMiniappGiftsCache(gift.EventID)
	}

	// Возвращаем успешный ответ без содержимого
	response.NoContent(w)
}
