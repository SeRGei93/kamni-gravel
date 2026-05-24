package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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

	// Создаём временный bot API клиент для получения file_path
	bot, err := tgbotapi.NewBotAPI(h.botToken)
	if err != nil {
		log.Printf("Error creating bot API: %v", err)
		response.InternalServerError(w, "Failed to connect to Telegram")
		return
	}

	// Получаем информацию о файле
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		log.Printf("Error getting file from Telegram: %v", err)
		response.NotFound(w, "File not found")
		return
	}

	// Формируем URL файла
	fileURL := file.Link(h.botToken)

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

	// Создаём временный bot API клиент
	bot, err := tgbotapi.NewBotAPI(h.botToken)
	if err != nil {
		log.Printf("Error creating bot API: %v", err)
		response.InternalServerError(w, "Failed to connect to Telegram")
		return
	}

	// Получаем информацию о файле
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		log.Printf("Error getting file from Telegram: %v", err)
		response.NotFound(w, "File not found")
		return
	}

	// Возвращаем информацию о файле
	fileInfo := map[string]interface{}{
		"file_id":   file.FileID,
		"file_path": file.FilePath,
		"file_size": file.FileSize,
		"url":       file.Link(h.botToken),
	}

	response.Success(w, fileInfo)
}
