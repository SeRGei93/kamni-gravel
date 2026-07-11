package handler

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/response"
)

// ParticipantsHandler обрабатывает запросы для участников
type ParticipantsHandler struct {
	participantRepo             repository.ParticipantRepository
	resultRepo                  repository.ResultRepository
	giftRepo                    repository.GiftRepository
	criteriaRepo                repository.CriteriaRepository
	prizeAssignmentRepo         repository.PrizeAssignmentRepository
	getParticipantsHandler      *query.GetParticipantsHandler
	getParticipantByIDHandler   *query.GetParticipantByIDHandler
	getPrizeDistributionHandler *query.GetPrizeDistributionHandler
	registerParticipantHandler  *command.RegisterParticipantHandler
	updateParticipantHandler    *command.UpdateParticipantHandler
	deleteParticipantHandler    *command.DeleteParticipantHandler
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
	deleteParticipantHandler *command.DeleteParticipantHandler,
) *ParticipantsHandler {
	return &ParticipantsHandler{
		participantRepo:             participantRepo,
		resultRepo:                  resultRepo,
		giftRepo:                    giftRepo,
		criteriaRepo:                criteriaRepo,
		prizeAssignmentRepo:         prizeAssignmentRepo,
		getParticipantsHandler:      getParticipantsHandler,
		getParticipantByIDHandler:   getParticipantByIDHandler,
		getPrizeDistributionHandler: getPrizeDistributionHandler,
		registerParticipantHandler:  registerParticipantHandler,
		updateParticipantHandler:    updateParticipantHandler,
		deleteParticipantHandler:    deleteParticipantHandler,
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

	// Пагинация включается только если переданы page/page_size. Без них возвращаем
	// всех участников (нужно для номинаций и глобального поиска).
	page := ParsePageParams(r)
	paginate := (r.URL.Query().Has("page") || r.URL.Query().Has("page_size")) && !page.All

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

	var criteriaID *uint
	if criteriaIDStr := r.URL.Query().Get("criteria_id"); criteriaIDStr != "" {
		parsedCriteriaID, parseErr := strconv.ParseUint(criteriaIDStr, 10, 32)
		if parseErr != nil || parsedCriteriaID == 0 {
			response.BadRequest(w, "Invalid criteria ID")
			return
		}
		parsed := uint(parsedCriteriaID)
		criteriaID = &parsed
	}

	// Доп. фильтры, применяемые на уровне обработчика (server-side, чтобы пагинация
	// была корректной): наличие подарка и текстовый поиск.
	var hasGiftFilter *bool
	if hasGiftStr := r.URL.Query().Get("has_gift"); hasGiftStr != "" {
		hg := hasGiftStr == "true"
		hasGiftFilter = &hg
	}
	rawSearchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	searchQuery := strings.TrimLeft(strings.ToLower(rawSearchQuery), "@")
	if rawSearchQuery != "" {
		log.Printf("DEBUG [FIX:participant-search] normalized query: event_id=%d has_username_prefix=%t", eventID, strings.HasPrefix(rawSearchQuery, "@"))
	}
	sortKey := r.URL.Query().Get("sort")
	sortOrder := r.URL.Query().Get("order")

	// Вызываем query handler (фильтры по велосипеду/полу/финишу + расчёт мест)
	participantsWithPlace, err := h.getParticipantsHandler.Handle(r.Context(), queryParams)
	if err != nil {
		log.Printf("Error getting participants: %v", err)
		response.InternalServerError(w, "Failed to get participants")
		return
	}

	giftUserIDs, err := h.getGiftUserIDsByEvent(r.Context(), uint(eventID))
	if err != nil {
		log.Printf("Error getting participant gift flags: event_id=%d error=%v", eventID, err)
		response.InternalServerError(w, "Failed to get participant gift flags")
		return
	}

	// Получаем результаты с глобальными местами (по всему событию, не зависят от фильтра).
	resultMap := make(map[uint]*repository.ResultWithPlace)
	if resultsWithPlaces, err := h.resultRepo.FindByEventWithPlaces(r.Context(), uint(eventID)); err != nil {
		log.Printf("Error getting results with places (places omitted): event_id=%d error=%v", eventID, err)
	} else {
		for _, rwp := range resultsWithPlaces {
			resultMap[rwp.Result.ParticipantID] = rwp
		}
	}

	// Время участников на предыдущем событии (user_id → секунды). Ошибка не
	// блокирует список — колонка просто останется пустой.
	prevElapsedByUser, err := h.resultRepo.FindPrevEventElapsedByUser(r.Context(), uint(eventID))
	if err != nil {
		log.Printf("Error getting previous event times (column omitted): event_id=%d error=%v", eventID, err)
		prevElapsedByUser = nil
	}

	automaticPrizeCounts, err := h.automaticPrizeCountsByParticipant(r.Context(), uint(eventID))
	if err != nil {
		log.Printf("ERROR participant prize counts failed: event_id=%d stage=automatic_distribution error=%v", eventID, err)
		response.InternalServerError(w, "Failed to calculate participant prize counts")
		return
	}
	manualPrizeCounts, err := h.manualPrizeCountsByParticipant(r.Context(), uint(eventID))
	if err != nil {
		log.Printf("ERROR participant prize counts failed: event_id=%d stage=manual_assignments error=%v", eventID, err)
		response.InternalServerError(w, "Failed to calculate participant prize counts")
		return
	}
	log.Printf("DEBUG participant prize counts calculated: event_id=%d automatic_participants=%d manual_participants=%d", eventID, len(automaticPrizeCounts), len(manualPrizeCounts))

	participantIDsByCriteria, err := h.participantIDsByResultCriteria(
		r.Context(),
		uint(eventID),
		criteriaID,
		participantsWithPlace,
	)
	if err != nil {
		log.Printf("Error getting participants by result criteria: event_id=%d criteria_id=%v error=%v", eventID, criteriaID, err)
		response.InternalServerError(w, "Failed to filter participants by criteria")
		return
	}

	// Конвертируем в DTO с местами и флагом has_gift.
	allDTOs := make([]*dto.ParticipantDTO, 0, len(participantsWithPlace))
	for _, pwp := range participantsWithPlace {
		participantDTO := dto.FromParticipant(pwp.Participant)
		participantDTO.Place = pwp.Place
		_, participantDTO.HasGift = giftUserIDs[pwp.Participant.UserID]
		participantDTO.PrizesCount = automaticPrizeCounts[pwp.Participant.ID] + manualPrizeCounts[pwp.Participant.ID]

		if rwp, ok := resultMap[pwp.Participant.ID]; ok {
			participantDTO.PlaceAbsolute = &rwp.PlaceAbsolute
			participantDTO.PlaceByGender = &rwp.PlaceByGender
			participantDTO.PlaceByGenderBike = &rwp.PlaceByGenderBike
		}

		// Ручное «время прошлого года» уже выставлено в FromParticipant и
		// имеет приоритет над вычисленным по предыдущему событию.
		if participantDTO.PrevElapsedTimeSec == nil {
			if prevSec, ok := prevElapsedByUser[pwp.Participant.UserID]; ok {
				participantDTO.SetPrevElapsed(prevSec)
			}
		}

		allDTOs = append(allDTOs, participantDTO)
	}

	// Применяем фильтры has_gift и поиск (server-side, до пагинации).
	filtered := make([]*dto.ParticipantDTO, 0, len(allDTOs))
	for _, d := range allDTOs {
		if criteriaID != nil {
			if _, ok := participantIDsByCriteria[d.ID]; !ok {
				continue
			}
		}
		if hasGiftFilter != nil && d.HasGift != *hasGiftFilter {
			continue
		}
		if searchQuery != "" && !participantMatchesSearch(d, searchQuery) {
			continue
		}
		filtered = append(filtered, d)
	}

	// Сортировка (server-side, до пагинации) по числовым/временным колонкам,
	// чтобы порядок охватывал все страницы отфильтрованного набора.
	sortParticipantDTOs(filtered, sortKey, sortOrder)

	total := len(filtered)
	if rawSearchQuery != "" {
		log.Printf("DEBUG [FIX:participant-search] filter completed: event_id=%d result_count=%d", eventID, total)
	}

	// Пагинация (срез страницы) поверх отфильтрованного набора.
	pageItems := filtered
	if paginate {
		start := page.Offset
		if start > total {
			start = total
		}
		end := start + page.Limit
		if end > total {
			end = total
		}
		pageItems = filtered[start:end]
	}

	resp := dto.ParticipantListResponse{
		Participants: pageItems,
		Total:        total,
	}
	if paginate {
		resp.Page = page.Page
		resp.PageSize = page.PageSize
	}

	log.Printf("DEBUG Participants list served: event_id=%d paginated=%t total=%d page=%d page_size=%d returned=%d criteria_id=%v has_gift=%v q=%q sort=%q order=%q",
		eventID, paginate, total, page.Page, page.PageSize, len(pageItems), criteriaID, hasGiftFilter, searchQuery, sortKey, sortOrder)

	response.Success(w, resp)
}

func (h *ParticipantsHandler) participantIDsByResultCriteria(
	ctx context.Context,
	eventID uint,
	criteriaID *uint,
	participantsWithPlace []*query.ParticipantWithPlace,
) (map[uint]struct{}, error) {
	if criteriaID == nil {
		return nil, nil
	}
	if h.criteriaRepo == nil {
		return nil, errors.New("criteria repository is not configured")
	}

	if finder, ok := h.criteriaRepo.(repository.ResultCriteriaParticipantRepository); ok {
		return finder.FindParticipantIDsByResultCriteria(ctx, eventID, *criteriaID)
	}

	participantIDs := make(map[uint]struct{})
	for _, participantWithPlace := range participantsWithPlace {
		result := participantWithPlace.Participant.Result
		if result == nil {
			continue
		}

		criteria, err := h.criteriaRepo.FindByResult(ctx, result.ID)
		if err != nil {
			return nil, err
		}
		for _, criterion := range criteria {
			if criterion.ID == *criteriaID {
				participantIDs[participantWithPlace.Participant.ID] = struct{}{}
				break
			}
		}
	}

	return participantIDs, nil
}

func (h *ParticipantsHandler) automaticPrizeCountsByParticipant(ctx context.Context, eventID uint) (map[uint]int, error) {
	counts := make(map[uint]int)
	if h.getPrizeDistributionHandler == nil {
		// This is used only by narrowly constructed legacy handlers in tests.
		return counts, nil
	}

	distribution, err := h.getPrizeDistributionHandler.Handle(ctx, query.GetPrizeDistributionQuery{EventID: eventID})
	if err != nil {
		return nil, err
	}
	for _, participant := range distribution {
		if len(participant.MatchedGiftAssignments) > 0 {
			counts[participant.ParticipantID] += len(participant.MatchedGiftAssignments)
			continue
		}
		// Legacy distribution responses did not carry slot metadata. Use this
		// fallback only when assignments are absent to avoid double counting.
		counts[participant.ParticipantID] += len(participant.MatchedGifts)
	}
	return counts, nil
}

func (h *ParticipantsHandler) manualPrizeCountsByParticipant(ctx context.Context, eventID uint) (map[uint]int, error) {
	if counter, ok := h.giftRepo.(repository.ManualGiftRecipientCountRepository); ok {
		return counter.ManualRecipientCountsByEvent(ctx, eventID)
	}

	// Compatibility fallback for narrowly-scoped test doubles that predate the
	// aggregation contract. Production PostgreSQL repositories implement the
	// aggregation above, so database failures still surface as HTTP errors.
	gifts, err := h.giftRepo.FindByEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	counts := make(map[uint]int)
	for _, gift := range gifts {
		if !gift.ManualDistribution || gift.ManualRecipientParticipantID == nil {
			continue
		}
		counts[*gift.ManualRecipientParticipantID]++
	}
	return counts, nil
}

// participantMatchesSearch проверяет совпадение участника с поисковым запросом
// (без учёта регистра) по нику, имени, фамилии и Telegram ID.
func participantMatchesSearch(d *dto.ParticipantDTO, queryLower string) bool {
	if strings.Contains(strings.ToLower(d.Username), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(d.FirstName), queryLower) {
		return true
	}
	if strings.Contains(strings.ToLower(d.LastName), queryLower) {
		return true
	}
	if strings.Contains(strconv.FormatInt(d.UserID, 10), queryLower) {
		return true
	}
	return false
}

// participantSorter описывает сортировку списка участников по одной колонке.
type participantSorter struct {
	// missing сообщает, что значение отсутствует (уходит в конец при любом порядке).
	missing func(*dto.ParticipantDTO) bool
	// compare сравнивает два значения по возрастанию: <0, 0, >0.
	compare func(a, b *dto.ParticipantDTO) int
}

// participantSortComparators — множество сортируемых колонок. Ключи ДОЛЖНЫ
// совпадать с колонками, помеченными `sortable` на фронтенде (participantColumns).
var participantSortComparators = map[string]participantSorter{
	"place":                    placeSorter(),
	"place_absolute":           intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.PlaceAbsolute }),
	"place_by_gender":          intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.PlaceByGender }),
	"place_by_gender_bike":     intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.PlaceByGenderBike }),
	"prizes_count":             intValSorter(func(d *dto.ParticipantDTO) int { return d.PrizesCount }),
	"user_id":                  int64ValSorter(func(d *dto.ParticipantDTO) int64 { return d.UserID }),
	"distance_km":              intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.DistanceMeters }),
	"calories":                 intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.Calories }),
	"avg_heart_rate":           intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.AvgHeartRate }),
	"max_heart_rate":           intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.MaxHeartRate }),
	"avg_cadence":              intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.AvgCadence }),
	"elapsed_time":             intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.ElapsedTimeSec }),
	"moving_time":              intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.MovingTimeSec }),
	"prev_elapsed_time":        intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.PrevElapsedTimeSec }),
	"prev_elapsed_delta":       intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.PrevElapsedDeltaSec }),
	"idle_time":                intPtrSorter(func(d *dto.ParticipantDTO) *int { return d.IdleTimeSec }),
	"peak_speed_kmh":           floatPtrSorter(func(d *dto.ParticipantDTO) *float64 { return d.PeakSpeedKmh }),
	"avg_speed_kmh":            floatPtrSorter(func(d *dto.ParticipantDTO) *float64 { return d.AvgSpeedKmh }),
	"avg_moving_speed_kmh":     floatPtrSorter(func(d *dto.ParticipantDTO) *float64 { return d.AvgMovingSpeedKmh }),
	"peak_avg_speed_delta_kmh": floatPtrSorter(func(d *dto.ParticipantDTO) *float64 { return d.PeakAvgSpeedDeltaKmh }),
	"started_at":               timePtrSorter(func(d *dto.ParticipantDTO) *time.Time { return d.StartedAt }),
	"ride_finished_at":         timePtrSorter(func(d *dto.ParticipantDTO) *time.Time { return d.RideFinishedAt }),
	"ride_date":                strPtrSorter(func(d *dto.ParticipantDTO) *string { return d.RideDate }),
}

// sortParticipantDTOs стабильно сортирует список по колонке sortKey. Пустой
// ключ оставляет порядок по умолчанию; неизвестный ключ игнорируется (WARN).
// Отсутствующие значения всегда уходят в конец, независимо от направления.
func sortParticipantDTOs(items []*dto.ParticipantDTO, sortKey, order string) {
	if sortKey == "" {
		return
	}
	sorter, ok := participantSortComparators[sortKey]
	if !ok {
		log.Printf("WARN Participants list unknown sort key ignored: sort=%q", sortKey)
		return
	}

	desc := order == "desc"
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		aMissing, bMissing := sorter.missing(a), sorter.missing(b)
		if aMissing || bMissing {
			if aMissing && bMissing {
				return false
			}
			// Непустое значение всегда идёт раньше пустого, при любом направлении.
			return bMissing
		}

		c := sorter.compare(a, b)
		if c == 0 {
			return false
		}
		if desc {
			return c > 0
		}
		return c < 0
	})
}

func intPtrSorter(get func(*dto.ParticipantDTO) *int) participantSorter {
	return participantSorter{
		missing: func(d *dto.ParticipantDTO) bool { return get(d) == nil },
		compare: func(a, b *dto.ParticipantDTO) int { return cmp.Compare(*get(a), *get(b)) },
	}
}

func floatPtrSorter(get func(*dto.ParticipantDTO) *float64) participantSorter {
	return participantSorter{
		missing: func(d *dto.ParticipantDTO) bool { return get(d) == nil },
		compare: func(a, b *dto.ParticipantDTO) int { return cmp.Compare(*get(a), *get(b)) },
	}
}

func timePtrSorter(get func(*dto.ParticipantDTO) *time.Time) participantSorter {
	return participantSorter{
		missing: func(d *dto.ParticipantDTO) bool { return get(d) == nil },
		compare: func(a, b *dto.ParticipantDTO) int { return get(a).Compare(*get(b)) },
	}
}

func strPtrSorter(get func(*dto.ParticipantDTO) *string) participantSorter {
	return participantSorter{
		missing: func(d *dto.ParticipantDTO) bool { s := get(d); return s == nil || *s == "" },
		compare: func(a, b *dto.ParticipantDTO) int { return cmp.Compare(*get(a), *get(b)) },
	}
}

func intValSorter(get func(*dto.ParticipantDTO) int) participantSorter {
	return participantSorter{
		missing: func(*dto.ParticipantDTO) bool { return false },
		compare: func(a, b *dto.ParticipantDTO) int { return cmp.Compare(get(a), get(b)) },
	}
}

func int64ValSorter(get func(*dto.ParticipantDTO) int64) participantSorter {
	return participantSorter{
		missing: func(*dto.ParticipantDTO) bool { return false },
		compare: func(a, b *dto.ParticipantDTO) int { return cmp.Compare(get(a), get(b)) },
	}
}

// placeSorter трактует место 0 («нет места») как отсутствующее — уходит в конец.
func placeSorter() participantSorter {
	return participantSorter{
		missing: func(d *dto.ParticipantDTO) bool { return d.Place == 0 },
		compare: func(a, b *dto.ParticipantDTO) int { return cmp.Compare(a.Place, b.Place) },
	}
}

func (h *ParticipantsHandler) getGiftUserIDsByEvent(ctx context.Context, eventID uint) (map[int64]struct{}, error) {
	gifts, err := h.giftRepo.FindByEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	userIDs := make(map[int64]struct{}, len(gifts))
	for _, gift := range gifts {
		userIDs[gift.UserID] = struct{}{}
	}

	return userIDs, nil
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

	// «Время прошлого года»: ручное значение уже выставлено в FromParticipant;
	// иначе берём вычисленное по предыдущему событию. Ошибка не блокирует ответ.
	if participantDTO.PrevElapsedTimeSec == nil {
		if prevElapsedByUser, err := h.resultRepo.FindPrevEventElapsedByUser(r.Context(), participant.EventID); err != nil {
			log.Printf("Error getting previous event times (field omitted): event_id=%d participant_id=%d error=%v", participant.EventID, participant.ID, err)
		} else if prevSec, ok := prevElapsedByUser[participant.UserID]; ok {
			participantDTO.SetPrevElapsed(prevSec)
		}
	}

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
			if dist.ParticipantID == participant.ID && (len(dist.MatchedGifts) > 0 || len(dist.MatchedGiftAssignments) > 0) {
				// Собираем все подарки
				participantDTO.MatchedGifts = make([]*dto.GiftDTO, 0, len(dist.MatchedGifts))
				for _, gift := range dist.MatchedGifts {
					participantDTO.MatchedGifts = append(participantDTO.MatchedGifts, dto.FromGift(gift))
				}
				participantDTO.MatchedGiftAssignments = make([]*dto.PrizeGiftAssignmentDTO, 0, len(dist.MatchedGiftAssignments))
				for _, assignment := range dist.MatchedGiftAssignments {
					participantDTO.MatchedGiftAssignments = append(participantDTO.MatchedGiftAssignments, dto.FromPrizeGiftAssignment(assignment))
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
		log.Printf("Error registering participant: event_id=%d telegram_user_id=%d error=%v", eventID, req.UserID, err)
		switch {
		case errors.Is(err, command.ErrUserBlacklisted):
			log.Printf("WARN Participant registration blocked in HTTP: event_id=%d telegram_user_id=%d reason=blacklisted", eventID, req.UserID)
			response.Forbidden(w, err.Error())
		case errors.Is(err, command.ErrUserNotFound):
			response.NotFound(w, err.Error())
		case errors.Is(err, command.ErrEventNotFound):
			response.NotFound(w, err.Error())
		case errors.Is(err, command.ErrEventNotActive):
			response.BadRequest(w, err.Error())
		case errors.Is(err, command.ErrAlreadyRegistered):
			response.Conflict(w, err.Error())
		case errors.Is(err, command.ErrInvalidBikeType), errors.Is(err, command.ErrInvalidGender):
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
	Status   *string `json:"status,omitempty"`
	// Ручное «время прошлого года» в секундах: 0 — удалить ручное значение.
	PrevElapsedTimeSec *int `json:"prev_elapsed_time_sec,omitempty"`
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
		ParticipantID:      uint(id),
		BikeType:           req.BikeType,
		Gender:             req.Gender,
		Notes:              req.Notes,
		Status:             req.Status,
		PrevElapsedTimeSec: req.PrevElapsedTimeSec,
	})
	if err != nil {
		log.Printf("Error updating participant: %v", err)
		if err.Error() == "participant not found" {
			response.NotFound(w, err.Error())
		} else if err == command.ErrInvalidBikeType || err == command.ErrInvalidGender || err == command.ErrInvalidStatus || err == command.ErrInvalidPrevElapsedTime {
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

	log.Printf("Participant delete request received: participant_id=%d", id)
	if err := h.deleteParticipantHandler.Handle(r.Context(), command.DeleteParticipantCommand{ParticipantID: uint(id)}); err != nil {
		if errors.Is(err, command.ErrParticipantNotFound) {
			log.Printf("Participant delete not found: participant_id=%d", id)
			response.NotFound(w, "Participant not found")
			return
		}
		log.Printf("Error deleting participant: participant_id=%d error=%v", id, err)
		response.InternalServerError(w, "Failed to delete participant")
		return
	}

	log.Printf("Participant delete completed: participant_id=%d", id)
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
