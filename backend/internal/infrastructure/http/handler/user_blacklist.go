package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/infrastructure/http/response"
)

// UserBlacklistHandler обрабатывает admin API для blacklist пользователей.
type UserBlacklistHandler struct {
	listHandler         *query.ListUserBlacklistHandler
	addHandler          *command.AddUserBlacklistHandler
	updateReasonHandler *command.UpdateUserBlacklistReasonHandler
	removeHandler       *command.RemoveUserBlacklistHandler
}

// NewUserBlacklistHandler создаёт новый handler.
func NewUserBlacklistHandler(
	listHandler *query.ListUserBlacklistHandler,
	addHandler *command.AddUserBlacklistHandler,
	updateReasonHandler *command.UpdateUserBlacklistReasonHandler,
	removeHandler *command.RemoveUserBlacklistHandler,
) *UserBlacklistHandler {
	return &UserBlacklistHandler{
		listHandler:         listHandler,
		addHandler:          addHandler,
		updateReasonHandler: updateReasonHandler,
		removeHandler:       removeHandler,
	}
}

// UserBlacklistRequest представляет запрос создания/обновления blacklist.
type UserBlacklistRequest struct {
	TelegramUserID int64  `json:"telegram_user_id"`
	Reason         string `json:"reason"`
}

// GetAll обрабатывает GET /api/user-blacklist.
func (h *UserBlacklistHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	entries, err := h.listHandler.Handle(r.Context())
	if err != nil {
		log.Printf("ERROR User blacklist list failed in HTTP: error=%v", err)
		response.InternalServerError(w, "Failed to get user blacklist")
		return
	}

	entryDTOs := make([]*dto.UserBlacklistDTO, 0, len(entries))
	for _, entry := range entries {
		entryDTOs = append(entryDTOs, dto.FromUserBlacklist(entry))
	}

	response.Success(w, dto.UserBlacklistListResponse{
		Entries: entryDTOs,
		Total:   len(entryDTOs),
	})
}

// Create обрабатывает POST /api/user-blacklist.
func (h *UserBlacklistHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req UserBlacklistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("WARN User blacklist create failed: operation=decode_body error=%v", err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	entry, err := h.addHandler.Handle(r.Context(), command.AddUserBlacklistCommand{
		TelegramUserID: req.TelegramUserID,
		Reason:         req.Reason,
	})
	if err != nil {
		log.Printf("ERROR User blacklist create failed: telegram_user_id=%d error=%v", req.TelegramUserID, err)
		if errors.Is(err, command.ErrInvalidTelegramUserID) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalServerError(w, "Failed to add user blacklist entry")
		return
	}

	log.Printf("INFO User blacklist create completed in HTTP: telegram_user_id=%d", req.TelegramUserID)
	response.Created(w, dto.FromUserBlacklist(entry))
}

// Update обрабатывает PUT /api/user-blacklist/{telegramUserId}.
func (h *UserBlacklistHandler) Update(w http.ResponseWriter, r *http.Request) {
	telegramUserID, ok := parseTelegramUserIDParam(w, r)
	if !ok {
		return
	}

	var req UserBlacklistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("WARN User blacklist update failed: operation=decode_body telegram_user_id=%d error=%v", telegramUserID, err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	entry, err := h.updateReasonHandler.Handle(r.Context(), command.UpdateUserBlacklistReasonCommand{
		TelegramUserID: telegramUserID,
		Reason:         req.Reason,
	})
	if err != nil {
		log.Printf("ERROR User blacklist update failed: telegram_user_id=%d error=%v", telegramUserID, err)
		switch {
		case errors.Is(err, command.ErrInvalidTelegramUserID):
			response.BadRequest(w, err.Error())
		case errors.Is(err, command.ErrUserBlacklistNotFound):
			response.NotFound(w, "User blacklist entry not found")
		default:
			response.InternalServerError(w, "Failed to update user blacklist entry")
		}
		return
	}

	log.Printf("INFO User blacklist update completed in HTTP: telegram_user_id=%d", telegramUserID)
	response.Success(w, dto.FromUserBlacklist(entry))
}

// Delete обрабатывает DELETE /api/user-blacklist/{telegramUserId}.
func (h *UserBlacklistHandler) Delete(w http.ResponseWriter, r *http.Request) {
	telegramUserID, ok := parseTelegramUserIDParam(w, r)
	if !ok {
		return
	}

	err := h.removeHandler.Handle(r.Context(), command.RemoveUserBlacklistCommand{
		TelegramUserID: telegramUserID,
	})
	if err != nil {
		log.Printf("ERROR User blacklist delete failed: telegram_user_id=%d error=%v", telegramUserID, err)
		switch {
		case errors.Is(err, command.ErrInvalidTelegramUserID):
			response.BadRequest(w, err.Error())
		case errors.Is(err, command.ErrUserBlacklistNotFound):
			response.NotFound(w, "User blacklist entry not found")
		default:
			response.InternalServerError(w, "Failed to delete user blacklist entry")
		}
		return
	}

	log.Printf("INFO User blacklist delete completed in HTTP: telegram_user_id=%d", telegramUserID)
	response.NoContent(w)
}

func parseTelegramUserIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	idStr := chi.URLParam(r, "telegramUserId")
	telegramUserID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || telegramUserID <= 0 {
		log.Printf("WARN User blacklist request has invalid Telegram ID: telegram_user_id=%q", idStr)
		response.BadRequest(w, "Invalid Telegram user ID")
		return 0, false
	}

	return telegramUserID, true
}
