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

// ParticipantsHandler обрабатывает запросы для участников
type ParticipantsHandler struct {
	participantRepo            repository.ParticipantRepository
	resultRepo                 repository.ResultRepository
	giftRepo                   repository.GiftRepository
	criteriaRepo               repository.CriteriaRepository
	prizeAssignmentRepo        repository.PrizeAssignmentRepository
	getParticipantsHandler     *query.GetParticipantsHandler
	getParticipantByIDHandler  *query.GetParticipantByIDHandler
	getPrizeDistributionHandler *query.GetPrizeDistributionHandler
	registerParticipantHandler *command.RegisterParticipantHandler
	updateParticipantHandler   *command.UpdateParticipantHandler
}

// NewParticipantsHandler создаёт новый handler
func NewParticipantsHandler(
	participantRepo repository.ParticipantRepository,
	resultRepo repository.ResultRepository,
	giftRepo repository.GiftRepository,
	criteriaRepo repository.CriteriaRepository,
	prizeAssignmentRepo repository.PrizeAssignmentRepository,
	getParticipantsHandler *query.GetParticipantsHandler,
	getParticipantByIDHandler *query.GetParticipantByIDHandler,
	getPrizeDistributionHandler *query.GetPrizeDistributionHandler,
	registerParticipantHandler *command.RegisterParticipantHandler,
	updateParticipantHandler *command.UpdateParticipantHandler,
) *ParticipantsHandler {
	return &ParticipantsHandler{
		participantRepo:             participantRepo,
		resultRepo:                  resultRepo,
		giftRepo:                     giftRepo,
		criteriaRepo:                 criteriaRepo,
		prizeAssignmentRepo:          prizeAssignmentRepo,
		getParticipantsHandler:      getParticipantsHandler,
		getParticipantByIDHandler:   getParticipantByIDHandler,
		getPrizeDistributionHandler: getPrizeDistributionHandler,
		registerParticipantHandler:  registerParticipantHandler,
		updateParticipantHandler:    updateParticipantHandler,
	}
}

// GetAll обрабатывает GET /api/events/:eventId/participants - список участников события
func (h *ParticipantsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Извлекаем eventID из URL
	eventIDStr := chi.URLParam(r, "eventId")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	// Парсим query параметры для фильтров
	queryParams := query.GetParticipantsQuery{
		EventID: uint(eventID),
	}

	if bikeType := r.URL.Query().Get("bike_type"); bikeType != "" {
		queryParams.BikeType = &bikeType
	}

	if gender := r.URL.Query().Get("gender"); gender != "" {
		queryParams.Gender = &gender
	}

	if isFinishedStr := r.URL.Query().Get("is_finished"); isFinishedStr != "" {
		isFinished := isFinishedStr == "true"
		queryParams.IsFinished = &isFinished
	}

	// Вызываем query handler
	participantsWithPlace, err := h.getParticipantsHandler.Handle(r.Context(), queryParams)
	if err != nil {
		log.Printf("Error getting participants: %v", err)
		response.InternalServerError(w, "Failed to get participants")
		return
	}

	// Получаем результаты с местами
	resultsWithPlaces, err := h.resultRepo.FindByEventWithPlaces(r.Context(), uint(eventID))
	if err != nil {
		log.Printf("Error getting results with places: %v", err)
		// Fallback на старую логику
		participantDTOs := make([]*dto.ParticipantDTO, 0, len(participantsWithPlace))
		for _, pwp := range participantsWithPlace {
			participantDTO := dto.FromParticipant(pwp.Participant)
			participantDTO.Place = pwp.Place
			participantDTOs = append(participantDTOs, participantDTO)
		}
		response.Success(w, dto.ParticipantListResponse{
			Participants: participantDTOs,
			Total:        len(participantDTOs),
		})
		return
	}

	// Создаём мапу результатов по participant_id
	resultMap := make(map[uint]*repository.ResultWithPlace)
	for _, rwp := range resultsWithPlaces {
		resultMap[rwp.Result.ParticipantID] = rwp
	}

	// Конвертируем в DTO с местами
	participantDTOs := make([]*dto.ParticipantDTO, 0, len(participantsWithPlace))
	for _, pwp := range participantsWithPlace {
		participantDTO := dto.FromParticipant(pwp.Participant)
		participantDTO.Place = pwp.Place

		// Добавляем места из результатов
		if rwp, ok := resultMap[pwp.Participant.ID]; ok {
			participantDTO.PlaceAbsolute = &rwp.PlaceAbsolute
			participantDTO.PlaceByGender = &rwp.PlaceByGender
			participantDTO.PlaceByGenderBike = &rwp.PlaceByGenderBike
		}

		participantDTOs = append(participantDTOs, participantDTO)
	}

	// Возвращаем ответ
	response.Success(w, dto.ParticipantListResponse{
		Participants: participantDTOs,
		Total:        len(participantDTOs),
	})
}

// GetByID обрабатывает GET /api/participants/:id - детали участника
func (h *ParticipantsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return
	}

	// Вызываем query handler
	participant, err := h.getParticipantByIDHandler.Handle(r.Context(), query.GetParticipantByIDQuery{
		ParticipantID: uint(id),
	})
	if err != nil {
		log.Printf("Error getting participant: %v", err)
		response.InternalServerError(w, "Failed to get participant")
		return
	}

	if participant == nil {
		response.NotFound(w, "Participant not found")
		return
	}

	// Конвертируем в DTO
	participantDTO := dto.FromParticipant(participant)

	// Получаем места и matched_gift через prize distribution
	resultsWithPlaces, err := h.resultRepo.FindByEventWithPlaces(r.Context(), participant.EventID)
	if err == nil {
		// Находим результат участника
		for _, rwp := range resultsWithPlaces {
			if rwp.Result.ParticipantID == participant.ID {
				participantDTO.PlaceAbsolute = &rwp.PlaceAbsolute
				participantDTO.PlaceByGender = &rwp.PlaceByGender
				participantDTO.PlaceByGenderBike = &rwp.PlaceByGenderBike
				break
			}
		}
	}

	// Получаем matched_gifts через prize distribution
	distribution, err := h.getPrizeDistributionHandler.Handle(r.Context(), query.GetPrizeDistributionQuery{
		EventID: participant.EventID,
	})
	if err == nil {
		// Находим запись для участника
		for _, dist := range distribution {
			if dist.ParticipantID == participant.ID && len(dist.MatchedGifts) > 0 {
				// Собираем все подарки
				participantDTO.MatchedGifts = make([]*dto.GiftDTO, 0, len(dist.MatchedGifts))
				for _, gift := range dist.MatchedGifts {
					participantDTO.MatchedGifts = append(participantDTO.MatchedGifts, dto.FromGift(gift))
				}
				break
			}
		}
	}

	// Возвращаем DTO
	response.Success(w, participantDTO)
}

// CreateRequest представляет запрос на регистрацию участника
type CreateParticipantRequest struct {
	UserID   int64  `json:"user_id"`
	BikeType string `json:"bike_type"`
	Gender   string `json:"gender"`
}

// Create обрабатывает POST /api/events/:eventId/participants - регистрация участника
func (h *ParticipantsHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Извлекаем eventID из URL
	eventIDStr := chi.URLParam(r, "eventId")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid event ID")
		return
	}

	var req CreateParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Вызываем command handler
	participant, err := h.registerParticipantHandler.Handle(r.Context(), command.RegisterParticipantCommand{
		UserID:   req.UserID,
		EventID:  uint(eventID),
		BikeType: req.BikeType,
		Gender:   req.Gender,
	})
	if err != nil {
		log.Printf("Error registering participant: %v", err)
		switch err {
		case command.ErrUserNotFound:
			response.NotFound(w, err.Error())
		case command.ErrEventNotFound:
			response.NotFound(w, err.Error())
		case command.ErrEventNotActive:
			response.BadRequest(w, err.Error())
		case command.ErrAlreadyRegistered:
			response.Conflict(w, err.Error())
		case command.ErrInvalidBikeType, command.ErrInvalidGender:
			response.BadRequest(w, err.Error())
		default:
			response.InternalServerError(w, "Failed to register participant")
		}
		return
	}

	// Возвращаем созданного участника
	response.Created(w, dto.FromParticipant(participant))
}

// UpdateRequest представляет запрос на обновление участника
type UpdateParticipantRequest struct {
	BikeType *string `json:"bike_type,omitempty"`
	Gender   *string `json:"gender,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

// Update обрабатывает PUT /api/participants/:id - обновление участника
func (h *ParticipantsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return
	}

	var req UpdateParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// Вызываем command handler
	participant, err := h.updateParticipantHandler.Handle(r.Context(), command.UpdateParticipantCommand{
		ParticipantID: uint(id),
		BikeType:      req.BikeType,
		Gender:        req.Gender,
		Notes:         req.Notes,
	})
	if err != nil {
		log.Printf("Error updating participant: %v", err)
		if err.Error() == "participant not found" {
			response.NotFound(w, err.Error())
		} else if err == command.ErrInvalidBikeType || err == command.ErrInvalidGender {
			response.BadRequest(w, err.Error())
		} else {
			response.InternalServerError(w, "Failed to update participant")
		}
		return
	}

	// Возвращаем обновлённого участника
	response.Success(w, dto.FromParticipant(participant))
}

// Delete обрабатывает DELETE /api/participants/:id - удаление участника
func (h *ParticipantsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return
	}

	// Проверяем существование участника
	_, err = h.participantRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Participant not found: %v", err)
		response.NotFound(w, "Participant not found")
		return
	}

	// Удаляем участника
	if err := h.participantRepo.Delete(r.Context(), uint(id)); err != nil {
		log.Printf("Error deleting participant: %v", err)
		response.InternalServerError(w, "Failed to delete participant")
		return
	}

	// Возвращаем успешный ответ без содержимого
	response.NoContent(w)
}

// GetGifts обрабатывает GET /api/participants/:id/gifts - подарки от участника
func (h *ParticipantsHandler) GetGifts(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return
	}

	// Получаем участника для получения user_id
	participant, err := h.participantRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Participant not found: %v", err)
		response.NotFound(w, "Participant not found")
		return
	}

	// Получаем подарки пользователя
	gifts, err := h.giftRepo.FindByUser(r.Context(), participant.UserID)
	if err != nil {
		log.Printf("Error getting gifts: %v", err)
		response.InternalServerError(w, "Failed to get gifts")
		return
	}

	// Загружаем критерии для каждого подарка
	for _, gift := range gifts {
		criteria, err := h.criteriaRepo.FindByGift(r.Context(), gift.ID)
		if err != nil {
			log.Printf("Error getting criteria for gift %d: %v", gift.ID, err)
			continue
		}
		gift.Criteria = criteria
	}

	// Конвертируем в DTO
	giftDTOs := make([]*dto.GiftDTO, 0, len(gifts))
	for _, gift := range gifts {
		giftDTOs = append(giftDTOs, dto.FromGift(gift))
	}

	response.Success(w, dto.GiftListResponse{
		Gifts: giftDTOs,
		Total: len(giftDTOs),
	})
}

// GetPrizes обрабатывает GET /api/participants/:id/prizes - призы участника
func (h *ParticipantsHandler) GetPrizes(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(w, "Invalid participant ID")
		return
	}

	// Проверяем существование участника
	_, err = h.participantRepo.FindByID(r.Context(), uint(id))
	if err != nil {
		log.Printf("Participant not found: %v", err)
		response.NotFound(w, "Participant not found")
		return
	}

	// Получаем призы участника
	prizes, err := h.prizeAssignmentRepo.FindByParticipant(r.Context(), uint(id))
	if err != nil {
		log.Printf("Error getting prizes: %v", err)
		response.InternalServerError(w, "Failed to get prizes")
		return
	}

	// Конвертируем в DTO
	prizeDTOs := make([]*dto.PrizeAssignmentDTO, 0, len(prizes))
	for _, prize := range prizes {
		prizeDTOs = append(prizeDTOs, dto.FromPrizeAssignment(prize))
	}

	response.Success(w, dto.PrizeAssignmentListResponse{
		PrizeAssignments: prizeDTOs,
		Total:            len(prizeDTOs),
	})
}
