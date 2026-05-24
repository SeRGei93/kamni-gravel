package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	telegrambot "github.com/go-telegram/bot"

	"gravel_bot/internal/infrastructure/http/response"
)

// TelegramHandler обрабатывает запросы для работы с Telegram API
type TelegramHandler struct {
	botToken string
}

// NewTelegramHandler создаёт новый handler
func NewTelegramHandler(botToken string) *TelegramHandler {
	return &TelegramHandler{
		botToken: botToken,
	}
}

// GetFileURL обрабатывает GET /api/telegram/files/:fileId - получение URL файла из Telegram
func (h *TelegramHandler) GetFileURL(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileId")
	if fileID == "" {
		response.BadRequest(w, "File ID is required")
		return
	}

	// Создаём временный bot API клиент для получения file_path без getMe на каждый запрос.
	bot, err := telegrambot.New(h.botToken, telegrambot.WithSkipGetMe())
	if err != nil {
		log.Printf("Error creating Telegram file client for file_id=%s: %v", fileID, err)
		response.InternalServerError(w, "Failed to connect to Telegram")
		return
	}

	// Получаем информацию о файле
	file, err := bot.GetFile(r.Context(), &telegrambot.GetFileParams{FileID: fileID})
	if err != nil {
		log.Printf("Error getting file from Telegram: file_id=%s error=%v", fileID, err)
		response.NotFound(w, "File not found")
		return
	}

	// Формируем URL файла
	fileURL := bot.FileDownloadLink(file)

	// Возвращаем URL
	response.Success(w, map[string]string{
		"url": fileURL,
	})
}

// GetFileInfo обрабатывает GET /api/telegram/files/:fileId/info - получение информации о файле
func (h *TelegramHandler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	fileID := chi.URLParam(r, "fileId")
	if fileID == "" {
		response.BadRequest(w, "File ID is required")
		return
	}

	// Создаём временный bot API клиент без getMe на каждый запрос.
	bot, err := telegrambot.New(h.botToken, telegrambot.WithSkipGetMe())
	if err != nil {
		log.Printf("Error creating Telegram file client for file_id=%s: %v", fileID, err)
		response.InternalServerError(w, "Failed to connect to Telegram")
		return
	}

	// Получаем информацию о файле
	file, err := bot.GetFile(r.Context(), &telegrambot.GetFileParams{FileID: fileID})
	if err != nil {
		log.Printf("Error getting file from Telegram: file_id=%s error=%v", fileID, err)
		response.NotFound(w, "File not found")
		return
	}

	// Возвращаем информацию о файле
	fileInfo := map[string]interface{}{
		"file_id":   file.FileID,
		"file_path": file.FilePath,
		"file_size": file.FileSize,
		"url":       bot.FileDownloadLink(file),
	}

	response.Success(w, fileInfo)
}
