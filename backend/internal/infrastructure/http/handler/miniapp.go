package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	telegrambot "github.com/go-telegram/bot"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/http/response"
)

// MiniappHandler обрабатывает защищённые запросы Telegram Mini App.
type MiniappHandler struct {
	eventRepo                         repository.EventRepository
	getMiniappGiftsHandler            *query.GetMiniappGiftsHandler
	getMiniappParticipantCountHandler *query.GetMiniappParticipantCountHandler
	getParticipantsHandler            *query.GetParticipantsHandler
	getParticipantByUserHandler       *query.GetParticipantByUserAndEventHandler
	resultRepo                        repository.ResultRepository
	getOwnerManualGiftsHandler        *query.GetOwnerManualGiftsHandler
	getMiniappParticipantsHandler     *query.GetMiniappParticipantsHandler
	setManualGiftRecipientHandler     *command.SetManualGiftRecipientHandler
	fileFetcher                       miniappFileFetcher
	giftsCache                        miniappGiftsCache
}

// ConfigureManualGiftManagement wires protected owner/manual flows after the
// base Mini App handler is constructed. All handlers remain server-owned.
func (h *MiniappHandler) ConfigureManualGiftManagement(
	getOwnerManualGiftsHandler *query.GetOwnerManualGiftsHandler,
	getMiniappParticipantsHandler *query.GetMiniappParticipantsHandler,
	setManualGiftRecipientHandler *command.SetManualGiftRecipientHandler,
) {
	h.getOwnerManualGiftsHandler = getOwnerManualGiftsHandler
	h.getMiniappParticipantsHandler = getMiniappParticipantsHandler
	h.setManualGiftRecipientHandler = setManualGiftRecipientHandler
}

type miniappFileFetcher interface {
	Fetch(ctx context.Context, fileID string) (*http.Response, error)
}

// miniappGiftsCache кеширует каталог одобренных подарков первого экрана мини-приложения.
// Реализация может быть nil — тогда кеширование выключено и каталог всегда читается из БД.
type miniappGiftsCache interface {
	Get(eventID uint, gender, bikeType string) ([]*dto.GiftDTO, bool)
	Set(eventID uint, gender, bikeType string, gifts []*dto.GiftDTO)
}

type telegramFileFetcher struct {
	botToken   string
	httpClient httpDoer
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewMiniappHandler создаёт handler для Telegram Mini App.
// giftsCache может быть nil — тогда каталог подарков не кешируется.
func NewMiniappHandler(
	eventRepo repository.EventRepository,
	getMiniappGiftsHandler *query.GetMiniappGiftsHandler,
	getMiniappParticipantCountHandler *query.GetMiniappParticipantCountHandler,
	getParticipantsHandler *query.GetParticipantsHandler,
	getParticipantByUserHandler *query.GetParticipantByUserAndEventHandler,
	resultRepo repository.ResultRepository,
	botToken string,
	giftsCache miniappGiftsCache,
) *MiniappHandler {
	return newMiniappHandlerWithFileFetcher(
		eventRepo,
		getMiniappGiftsHandler,
		getMiniappParticipantCountHandler,
		getParticipantsHandler,
		getParticipantByUserHandler,
		resultRepo,
		&telegramFileFetcher{
			botToken:   botToken,
			httpClient: http.DefaultClient,
		},
		giftsCache,
	)
}

func newMiniappHandlerWithFileFetcher(
	eventRepo repository.EventRepository,
	getMiniappGiftsHandler *query.GetMiniappGiftsHandler,
	getMiniappParticipantCountHandler *query.GetMiniappParticipantCountHandler,
	getParticipantsHandler *query.GetParticipantsHandler,
	getParticipantByUserHandler *query.GetParticipantByUserAndEventHandler,
	resultRepo repository.ResultRepository,
	fileFetcher miniappFileFetcher,
	giftsCache miniappGiftsCache,
) *MiniappHandler {
	return &MiniappHandler{
		eventRepo:                         eventRepo,
		getMiniappGiftsHandler:            getMiniappGiftsHandler,
		getMiniappParticipantCountHandler: getMiniappParticipantCountHandler,
		getParticipantsHandler:            getParticipantsHandler,
		getParticipantByUserHandler:       getParticipantByUserHandler,
		resultRepo:                        resultRepo,
		fileFetcher:                       fileFetcher,
		giftsCache:                        giftsCache,
	}
}

type MiniappSessionResponse struct {
	User                  MiniappTelegramUserDTO `json:"user"`
	Event                 MiniappEventDTO        `json:"event"`
	MyResultParticipantID *uint                  `json:"my_result_participant_id,omitempty"`
}

type MiniappTelegramUserDTO struct {
	ID           int64  `json:"id"`
	Username     string `json:"username,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	PhotoURL     string `json:"photo_url,omitempty"`
	IsPremium    bool   `json:"is_premium"`
}

type MiniappEventDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Session обрабатывает GET /api/miniapp/session.
func (h *MiniappHandler) Session(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetTelegramWebAppUserFromContext(r.Context())
	if !ok {
		log.Printf("WARN Miniapp session failed: reason=missing_telegram_user path=%s", r.URL.Path)
		response.Unauthorized(w, "Telegram user not found")
		return
	}

	event, ok := h.activeEvent(w, r, user.ID)
	if !ok {
		return
	}

	participant, err := h.getParticipantByUserHandler.Handle(r.Context(), query.GetParticipantByUserAndEventQuery{
		UserID:  user.ID,
		EventID: event.ID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf("DEBUG Miniapp session has no participant: telegram_user_id=%d event_id=%d", user.ID, event.ID)
			participant = nil
		} else {
			log.Printf("ERROR [FIX] Miniapp session participant lookup failed: telegram_user_id=%d event_id=%d error=%v", user.ID, event.ID, err)
			response.InternalServerError(w, "Failed to find current participant")
			return
		}
	}

	myResultParticipantID := miniappMyResultParticipantID(participant)
	log.Printf("INFO [FIX] Miniapp session resolved: telegram_user_id=%d event_id=%d has_my_result=%t", user.ID, event.ID, myResultParticipantID != nil)
	response.Success(w, MiniappSessionResponse{
		User:                  miniappTelegramUserDTO(user),
		Event:                 miniappEventDTO(event),
		MyResultParticipantID: myResultParticipantID,
	})
}

// MyGifts returns all active-event gifts added by the verified Telegram user.
func (h *MiniappHandler) MyGifts(w http.ResponseWriter, r *http.Request) {
	user, ok := h.telegramUser(w, r, "my gifts")
	if !ok {
		return
	}
	event, ok := h.activeEvent(w, r, user.ID)
	if !ok {
		return
	}
	if h.getOwnerManualGiftsHandler == nil {
		log.Printf("ERROR Miniapp my gifts unavailable: telegram_user_id=%d event_id=%d", user.ID, event.ID)
		response.InternalServerError(w, "My gifts are unavailable")
		return
	}

	models, err := h.getOwnerManualGiftsHandler.Handle(r.Context(), query.GetOwnerManualGiftsQuery{
		OwnerTelegramUserID: user.ID,
		EventID:             event.ID,
	})
	if err != nil {
		log.Printf("ERROR Miniapp my gifts failed: telegram_user_id=%d event_id=%d error=%v", user.ID, event.ID, err)
		response.InternalServerError(w, "Failed to get my gifts")
		return
	}

	gifts := make([]*dto.ManualGiftDTO, 0, len(models))
	for _, model := range models {
		gifts = append(gifts, manualGiftDTOFromReadModel(model))
	}
	log.Printf("INFO Miniapp my gifts served: telegram_user_id=%d event_id=%d gift_count=%d", user.ID, event.ID, len(gifts))
	response.Success(w, dto.ManualGiftListResponse{Gifts: gifts})
}

// Participants returns minimal same-event recipient options for the verified user.
func (h *MiniappHandler) Participants(w http.ResponseWriter, r *http.Request) {
	user, ok := h.telegramUser(w, r, "participants")
	if !ok {
		return
	}
	event, ok := h.activeEvent(w, r, user.ID)
	if !ok {
		return
	}
	if h.getMiniappParticipantsHandler == nil {
		log.Printf("ERROR Miniapp participant options unavailable: telegram_user_id=%d event_id=%d", user.ID, event.ID)
		response.InternalServerError(w, "Participant options are unavailable")
		return
	}

	models, err := h.getMiniappParticipantsHandler.Handle(r.Context(), query.GetMiniappParticipantsQuery{EventID: event.ID})
	if err != nil {
		log.Printf("ERROR Miniapp participant options failed: telegram_user_id=%d event_id=%d error=%v", user.ID, event.ID, err)
		response.InternalServerError(w, "Failed to get participants")
		return
	}

	participants := make([]*dto.MiniappParticipantOptionDTO, 0, len(models))
	for _, model := range models {
		participants = append(participants, &dto.MiniappParticipantOptionDTO{
			ID:          model.ID,
			DisplayName: model.DisplayName,
			Username:    model.Username,
			Status:      model.Status,
		})
	}
	log.Printf("INFO Miniapp participant options served: telegram_user_id=%d event_id=%d participant_count=%d", user.ID, event.ID, len(participants))
	response.Success(w, struct {
		Participants []*dto.MiniappParticipantOptionDTO `json:"participants"`
		Total        int                                `json:"total"`
	}{Participants: participants, Total: len(participants)})
}

// UpdateMyGiftRecipient replaces or clears a recipient on an owner manual gift.
func (h *MiniappHandler) UpdateMyGiftRecipient(w http.ResponseWriter, r *http.Request) {
	user, ok := h.telegramUser(w, r, "my gift recipient")
	if !ok {
		return
	}
	event, ok := h.activeEvent(w, r, user.ID)
	if !ok {
		return
	}
	if h.setManualGiftRecipientHandler == nil {
		log.Printf("ERROR Miniapp manual recipient update unavailable: telegram_user_id=%d event_id=%d", user.ID, event.ID)
		response.InternalServerError(w, "Manual gift assignment is unavailable")
		return
	}

	giftID, err := strconv.ParseUint(chi.URLParam(r, "giftId"), 10, 32)
	if err != nil || giftID == 0 {
		response.BadRequest(w, "Invalid gift ID")
		return
	}
	recipientID, err := decodeMiniappGiftRecipientRequest(r)
	if err != nil {
		log.Printf("WARN Miniapp manual recipient update rejected: telegram_user_id=%d event_id=%d gift_id=%d reason=malformed_payload", user.ID, event.ID, giftID)
		response.BadRequest(w, "Invalid request body")
		return
	}

	err = h.setManualGiftRecipientHandler.Handle(r.Context(), command.SetManualGiftRecipientCommand{
		GiftID:  uint(giftID),
		EventID: event.ID,
		Actor: command.ManualGiftRecipientActor{
			TelegramUserID: user.ID,
		},
		RecipientParticipantID: recipientID,
	})
	if err != nil {
		switch {
		case errors.Is(err, command.ErrGiftNotFound),
			errors.Is(err, command.ErrManualGiftOwnerForbidden),
			errors.Is(err, command.ErrManualGiftRecipientNotFound):
			log.Printf("WARN Miniapp manual recipient update rejected: telegram_user_id=%d event_id=%d gift_id=%d reason=not_found", user.ID, event.ID, giftID)
			response.NotFound(w, "Gift or participant not found")
		case errors.Is(err, command.ErrManualGiftNotManual),
			errors.Is(err, command.ErrManualGiftRecipientEvent):
			log.Printf("WARN Miniapp manual recipient update rejected: telegram_user_id=%d event_id=%d gift_id=%d reason=conflict", user.ID, event.ID, giftID)
			response.Conflict(w, err.Error())
		default:
			log.Printf("ERROR Miniapp manual recipient update failed: telegram_user_id=%d event_id=%d gift_id=%d error=%v", user.ID, event.ID, giftID, err)
			response.InternalServerError(w, "Failed to update manual gift recipient")
		}
		return
	}

	log.Printf("INFO Miniapp manual recipient updated: telegram_user_id=%d event_id=%d gift_id=%d recipient_participant_id=%s", user.ID, event.ID, giftID, miniappRecipientIDLogValue(recipientID))
	response.NoContent(w)
}

// miniappMyResultParticipantID возвращает только ID текущего пользователя с
// опубликованным результатом. Так Mini App показывает «Мой результат» лишь
// когда карточка уже доступна в публичном лидерборде, не раскрывая user_id
// других участников.
func miniappMyResultParticipantID(participant *entity.Participant) *uint {
	if participant == nil || !participant.IsFinished() {
		return nil
	}

	participantID := participant.ID
	return &participantID
}

// Gifts обрабатывает GET /api/miniapp/gifts.
func (h *MiniappHandler) Gifts(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetTelegramWebAppUserFromContext(r.Context())
	if !ok {
		log.Printf("WARN Miniapp gifts failed: reason=missing_telegram_user path=%s", r.URL.Path)
		response.Unauthorized(w, "Telegram user not found")
		return
	}

	event, ok := h.activeEvent(w, r, user.ID)
	if !ok {
		return
	}

	gender := r.URL.Query().Get("gender")
	bikeType := r.URL.Query().Get("bike_type")
	participantCount, err := h.getMiniappParticipantCountHandler.Handle(r.Context(), query.GetMiniappParticipantCountQuery{
		EventID:  event.ID,
		Gender:   gender,
		BikeType: bikeType,
	})
	if err != nil {
		if errors.Is(err, query.ErrInvalidMiniappGiftGenderFilter) ||
			errors.Is(err, query.ErrInvalidMiniappGiftBikeTypeFilter) {
			log.Printf("WARN Miniapp gifts rejected invalid participant filters: telegram_user_id=%d event_id=%d gender=%q bike_type=%q error=%v", user.ID, event.ID, gender, bikeType, err)
			response.BadRequest(w, err.Error())
			return
		}

		log.Printf("ERROR Miniapp participant count failed: telegram_user_id=%d event_id=%d gender=%q bike_type=%q error=%v", user.ID, event.ID, gender, bikeType, err)
		response.InternalServerError(w, "Failed to get participant count")
		return
	}

	// Каталог одобренных подарков — тяжёлая часть первого экрана, поэтому читаем его
	// через файловый кеш. Фильтры уже провалидированы вызовом participant count выше,
	// поэтому попадание в кеш безопасно отдаёт результат без повторной валидации.
	giftDTOs, cached := h.cachedMiniappGifts(event.ID, gender, bikeType)
	if !cached {
		gifts, err := h.getMiniappGiftsHandler.Handle(r.Context(), query.GetMiniappGiftsQuery{
			EventID:  event.ID,
			Gender:   gender,
			BikeType: bikeType,
		})
		if err != nil {
			if errors.Is(err, query.ErrInvalidMiniappGiftGenderFilter) ||
				errors.Is(err, query.ErrInvalidMiniappGiftBikeTypeFilter) {
				log.Printf("WARN Miniapp gifts rejected invalid filters: telegram_user_id=%d event_id=%d gender=%q bike_type=%q error=%v", user.ID, event.ID, gender, bikeType, err)
				response.BadRequest(w, err.Error())
				return
			}

			log.Printf("ERROR Miniapp gifts failed: telegram_user_id=%d event_id=%d gender=%q bike_type=%q error=%v", user.ID, event.ID, gender, bikeType, err)
			response.InternalServerError(w, "Failed to get gifts")
			return
		}

		giftDTOs = make([]*dto.GiftDTO, 0, len(gifts))
		for _, gift := range gifts {
			giftDTOs = append(giftDTOs, dto.FromGift(gift))
		}

		if h.giftsCache != nil {
			h.giftsCache.Set(event.ID, gender, bikeType, giftDTOs)
		}
	}

	log.Printf("INFO Miniapp gifts requested: telegram_user_id=%d event_id=%d gender=%q bike_type=%q result_count=%d participant_count=%d", user.ID, event.ID, gender, bikeType, len(giftDTOs), participantCount)
	response.Success(w, dto.GiftListResponse{
		Gifts:            giftDTOs,
		Total:            len(giftDTOs),
		ParticipantCount: &participantCount,
	})
}

// Leaderboard обрабатывает GET /api/miniapp/leaderboard.
// Возвращает всех участников активного события с рассчитанным местом и метриками
// заезда. Фильтрация по полу/типу велосипеда выполняется на клиенте, поэтому
// эндпоинт отдаёт полный список без query-фильтров.
func (h *MiniappHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetTelegramWebAppUserFromContext(r.Context())
	if !ok {
		log.Printf("WARN Miniapp leaderboard failed: reason=missing_telegram_user path=%s", r.URL.Path)
		response.Unauthorized(w, "Telegram user not found")
		return
	}

	event, ok := h.activeEvent(w, r, user.ID)
	if !ok {
		return
	}

	participants, err := h.getParticipantsHandler.Handle(r.Context(), query.GetParticipantsQuery{
		EventID: event.ID,
	})
	if err != nil {
		log.Printf("ERROR Miniapp leaderboard failed: telegram_user_id=%d event_id=%d error=%v", user.ID, event.ID, err)
		response.InternalServerError(w, "Failed to get leaderboard")
		return
	}

	var prevElapsedByUser map[int64]int
	if h.resultRepo != nil {
		prevElapsedByUser, err = h.resultRepo.FindPrevEventElapsedByUser(r.Context(), event.ID)
		if err != nil {
			log.Printf("ERROR Miniapp leaderboard previous event times omitted: telegram_user_id=%d event_id=%d error=%v", user.ID, event.ID, err)
			prevElapsedByUser = nil
		}
	}

	entries := make([]*dto.MiniappLeaderboardEntryDTO, 0, len(participants))
	for _, pwp := range participants {
		if pwp == nil || pwp.Participant == nil {
			continue
		}
		// Показываем только участников, отправивших результат: без результата
		// (IsFinished == false) в лидерборде не отображаем.
		if !pwp.Participant.IsFinished() {
			continue
		}
		entry := dto.NewMiniappLeaderboardEntry(pwp.Participant, pwp.Place)
		// Ручное время уже учтено в DTO и имеет приоритет над результатом
		// предыдущего события.
		if entry.PrevElapsedDeltaSec == nil {
			if prevSec, ok := prevElapsedByUser[pwp.Participant.UserID]; ok {
				entry.SetPrevElapsed(prevSec)
			}
		}
		entries = append(entries, entry)
	}

	log.Printf("INFO Miniapp leaderboard requested: telegram_user_id=%d event_id=%d participant_count=%d", user.ID, event.ID, len(entries))
	response.Success(w, dto.MiniappLeaderboardResponse{
		Participants: entries,
		Total:        len(entries),
	})
}

// TelegramFile обрабатывает GET /api/miniapp/telegram/files/{fileId}.
func (h *MiniappHandler) TelegramFile(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetTelegramWebAppUserFromContext(r.Context())
	if !ok {
		log.Printf("WARN Miniapp file proxy failed: reason=missing_telegram_user path=%s", r.URL.Path)
		response.Unauthorized(w, "Telegram user not found")
		return
	}

	fileID := chi.URLParam(r, "fileId")
	if fileID == "" {
		response.BadRequest(w, "File ID is required")
		return
	}

	fileResponse, err := h.fileFetcher.Fetch(r.Context(), fileID)
	if err != nil {
		log.Printf("WARN Miniapp file proxy failed: telegram_user_id=%d file_id=%s error=%v", user.ID, fileID, err)
		response.NotFound(w, "File not found")
		return
	}
	defer fileResponse.Body.Close()

	if fileResponse.StatusCode < http.StatusOK || fileResponse.StatusCode >= http.StatusMultipleChoices {
		log.Printf("WARN Miniapp file proxy failed: telegram_user_id=%d file_id=%s upstream_status=%d", user.ID, fileID, fileResponse.StatusCode)
		response.NotFound(w, "File not found")
		return
	}

	contentType := fileResponse.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	if contentLength := fileResponse.Header.Get("Content-Length"); contentLength != "" {
		w.Header().Set("Content-Length", contentLength)
	}
	// Telegram file_id неизменен, а байты по нему — тоже. Кешируем в браузере,
	// чтобы повторные открытия каталога и скролл не перекачивали фото заново.
	// private: ответ за Telegram-авторизацией, кеш только в браузере пользователя.
	w.Header().Set("Cache-Control", "private, max-age=2592000, immutable")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, fileResponse.Body); err != nil {
		log.Printf("WARN Miniapp file proxy stream failed: telegram_user_id=%d file_id=%s error=%v", user.ID, fileID, err)
		return
	}
	log.Printf("INFO Miniapp file proxied: telegram_user_id=%d file_id=%s", user.ID, fileID)
}

// cachedMiniappGifts возвращает закешированный каталог DTO для ключа, если кеш включён
// и есть попадание. Второй результат false означает, что нужно сходить в БД.
func (h *MiniappHandler) cachedMiniappGifts(eventID uint, gender, bikeType string) ([]*dto.GiftDTO, bool) {
	if h.giftsCache == nil {
		return nil, false
	}
	return h.giftsCache.Get(eventID, gender, bikeType)
}

func (h *MiniappHandler) activeEvent(w http.ResponseWriter, r *http.Request, telegramUserID int64) (*entity.Event, bool) {
	event, err := h.eventRepo.FindActive(r.Context())
	if err != nil {
		if errors.Is(err, repository.ErrNoActiveEvent) {
			log.Printf("WARN Miniapp request has no active event: telegram_user_id=%d path=%s", telegramUserID, r.URL.Path)
			response.NotFound(w, "No active event")
			return nil, false
		}
		log.Printf("ERROR Miniapp active event lookup failed: telegram_user_id=%d error=%v", telegramUserID, err)
		response.InternalServerError(w, "Failed to get active event")
		return nil, false
	}
	if event == nil {
		log.Printf("WARN Miniapp request has no active event: telegram_user_id=%d path=%s", telegramUserID, r.URL.Path)
		response.NotFound(w, "No active event")
		return nil, false
	}

	return event, true
}

func (h *MiniappHandler) telegramUser(w http.ResponseWriter, r *http.Request, operation string) (*middleware.TelegramWebAppUser, bool) {
	user, ok := middleware.GetTelegramWebAppUserFromContext(r.Context())
	if !ok {
		log.Printf("WARN Miniapp %s failed: reason=missing_telegram_user path=%s", operation, r.URL.Path)
		response.Unauthorized(w, "Telegram user not found")
		return nil, false
	}
	return user, true
}

func decodeMiniappGiftRecipientRequest(r *http.Request) (*uint, error) {
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode recipient request: %w", err)
	}
	if len(raw) != 1 {
		return nil, errors.New("recipient request must contain exactly participant_id")
	}
	payload, ok := raw["participant_id"]
	if !ok {
		return nil, errors.New("participant_id is required")
	}
	if bytes.Equal(bytes.TrimSpace(payload), []byte("null")) {
		return nil, nil
	}

	var participantID uint
	if err := json.Unmarshal(payload, &participantID); err != nil || participantID == 0 {
		return nil, errors.New("participant_id must be a positive integer or null")
	}
	return &participantID, nil
}

func miniappRecipientIDLogValue(participantID *uint) string {
	if participantID == nil {
		return "none"
	}
	return strconv.FormatUint(uint64(*participantID), 10)
}

func miniappTelegramUserDTO(user *middleware.TelegramWebAppUser) MiniappTelegramUserDTO {
	return MiniappTelegramUserDTO{
		ID:           user.ID,
		Username:     user.Username,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		LanguageCode: user.LanguageCode,
		PhotoURL:     user.PhotoURL,
		IsPremium:    user.IsPremium,
	}
}

func miniappEventDTO(event *entity.Event) MiniappEventDTO {
	return MiniappEventDTO{
		ID:          event.ID,
		Name:        event.Name,
		Description: event.Description,
	}
}

func (f *telegramFileFetcher) Fetch(ctx context.Context, fileID string) (*http.Response, error) {
	if f.botToken == "" {
		return nil, errors.New("telegram bot token is empty")
	}

	bot, err := telegrambot.New(f.botToken, telegrambot.WithSkipGetMe())
	if err != nil {
		return nil, fmt.Errorf("create Telegram file client: %w", err)
	}

	file, err := bot.GetFile(ctx, &telegrambot.GetFileParams{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("get Telegram file metadata: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bot.FileDownloadLink(file), nil)
	if err != nil {
		return nil, fmt.Errorf("create Telegram file download request: %w", err)
	}

	client := f.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download Telegram file content: %w", err)
	}

	return resp, nil
}
