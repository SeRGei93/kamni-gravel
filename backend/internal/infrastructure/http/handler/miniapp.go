package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	telegrambot "github.com/go-telegram/bot"

	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/http/response"
)

// MiniappHandler обрабатывает защищённые запросы Telegram Mini App.
type MiniappHandler struct {
	eventRepo              repository.EventRepository
	getMiniappGiftsHandler *query.GetMiniappGiftsHandler
	fileFetcher            miniappFileFetcher
}

type miniappFileFetcher interface {
	Fetch(ctx context.Context, fileID string) (*http.Response, error)
}

type telegramFileFetcher struct {
	botToken   string
	httpClient httpDoer
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewMiniappHandler создаёт handler для Telegram Mini App.
func NewMiniappHandler(
	eventRepo repository.EventRepository,
	getMiniappGiftsHandler *query.GetMiniappGiftsHandler,
	botToken string,
) *MiniappHandler {
	return newMiniappHandlerWithFileFetcher(
		eventRepo,
		getMiniappGiftsHandler,
		&telegramFileFetcher{
			botToken:   botToken,
			httpClient: http.DefaultClient,
		},
	)
}

func newMiniappHandlerWithFileFetcher(
	eventRepo repository.EventRepository,
	getMiniappGiftsHandler *query.GetMiniappGiftsHandler,
	fileFetcher miniappFileFetcher,
) *MiniappHandler {
	return &MiniappHandler{
		eventRepo:              eventRepo,
		getMiniappGiftsHandler: getMiniappGiftsHandler,
		fileFetcher:            fileFetcher,
	}
}

type MiniappSessionResponse struct {
	User  MiniappTelegramUserDTO `json:"user"`
	Event MiniappEventDTO        `json:"event"`
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

	log.Printf("INFO Miniapp session requested: telegram_user_id=%d event_id=%d", user.ID, event.ID)
	response.Success(w, MiniappSessionResponse{
		User:  miniappTelegramUserDTO(user),
		Event: miniappEventDTO(event),
	})
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

	giftDTOs := make([]*dto.GiftDTO, 0, len(gifts))
	for _, gift := range gifts {
		giftDTOs = append(giftDTOs, dto.FromGift(gift))
	}

	log.Printf("INFO Miniapp gifts requested: telegram_user_id=%d event_id=%d gender=%q bike_type=%q result_count=%d", user.ID, event.ID, gender, bikeType, len(giftDTOs))
	response.Success(w, dto.GiftListResponse{
		Gifts: giftDTOs,
		Total: len(giftDTOs),
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
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, fileResponse.Body); err != nil {
		log.Printf("WARN Miniapp file proxy stream failed: telegram_user_id=%d file_id=%s error=%v", user.ID, fileID, err)
		return
	}
	log.Printf("INFO Miniapp file proxied: telegram_user_id=%d file_id=%s", user.ID, fileID)
}

func (h *MiniappHandler) activeEvent(w http.ResponseWriter, r *http.Request, telegramUserID int64) (*entity.Event, bool) {
	event, err := h.eventRepo.FindActive(r.Context())
	if err != nil {
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
