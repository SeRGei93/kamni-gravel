package handler

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// GiftHandler обрабатывает добавление подарков
type GiftHandler struct {
	sessionManager *session.Manager
	eventRepo      repository.EventRepository
	addGiftHandler *command.AddGiftHandler
}

// NewGiftHandler создаёт новый handler
func NewGiftHandler(
	sessionManager *session.Manager,
	eventRepo repository.EventRepository,
	addGiftHandler *command.AddGiftHandler,
) *GiftHandler {
	return &GiftHandler{
		sessionManager: sessionManager,
		eventRepo:      eventRepo,
		addGiftHandler: addGiftHandler,
	}
}

// StartAddGift начинает процесс добавления подарка
func (h *GiftHandler) StartAddGift(ctx context.Context, userID int64) (string, *models.InlineKeyboardMarkup) {
	// Получаем активное событие
	event, err := h.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("Error finding active event: %v", err)
		return "Произошла ошибка. Попробуйте позже.", nil
	}

	if event == nil {
		return "В данный момент нет активных событий.", nil
	}

	// Сохраняем ID события в сессии
	h.sessionManager.SetData(userID, "event_id", event.ID)
	h.sessionManager.SetData(userID, "event_telegram_texts", entity.NormalizeEventTelegramTexts(event.TelegramTexts))
	h.sessionManager.SetState(userID, session.StateAwaitingGiftGender)

	texts := h.giftTexts(userID)
	text := texts.GiftGenderStep

	keyboard := keyboard.NewBuilder().
		AddRow(
			keyboard.Button("👨 Мужской", "gift_gender_male"),
			keyboard.Button("👩 Женский", "gift_gender_female"),
		).
		AddRow(
			keyboard.Button("👥 Любой", "gift_gender_all"),
		).
		AddRow(
			keyboard.Button("❌ Отмена", "cancel"),
		).
		Build()

	return text, &keyboard
}

// HandleGiftGenderSelection обрабатывает выбор гендера для подарка
func (h *GiftHandler) HandleGiftGenderSelection(ctx context.Context, userID int64, gender string) (string, *models.InlineKeyboardMarkup) {
	// Сохраняем гендер
	h.sessionManager.SetData(userID, "gift_gender", gender)
	h.sessionManager.SetState(userID, session.StateAwaitingGiftBikeType)

	texts := h.giftTexts(userID)
	text := texts.GiftBikeStep

	keyboard := keyboard.NewBuilder().
		AddRow(
			keyboard.Button("🚵 Гравийник", "gift_bike_gravel"),
			keyboard.Button("🏔 МТБ", "gift_bike_mtb"),
		).
		AddRow(
			keyboard.Button("🚴 Шоссе", "gift_bike_road"),
			keyboard.Button("⚡️ Фикс", "gift_bike_single_speed"),
		).
		AddRow(
			keyboard.Button("👥 Тандем", "gift_bike_tandem"),
		).
		AddRow(
			keyboard.Button("🚲 Любой", "gift_bike_all"),
		).
		AddRow(
			keyboard.Button("❌ Отмена", "cancel"),
		).
		Build()

	return text, &keyboard
}

// HandleGiftBikeTypeSelection обрабатывает выбор типа велосипеда для подарка
func (h *GiftHandler) HandleGiftBikeTypeSelection(ctx context.Context, userID int64, bikeType string) (string, *models.InlineKeyboardMarkup) {
	// Сохраняем тип велосипеда
	h.sessionManager.SetData(userID, "gift_bike_type", bikeType)
	h.sessionManager.SetState(userID, session.StateAwaitingGiftDesc)

	return h.GiftDescriptionPrompt(userID)
}

// HandleGiftDescription обрабатывает описание подарка
func (h *GiftHandler) HandleGiftDescription(ctx context.Context, userID int64, description string) (string, *models.InlineKeyboardMarkup) {
	description = strings.TrimSpace(description)
	if description == "" {
		log.Printf("Gift description input missing text: user_id=%d state=%s", userID, h.sessionManager.GetState(userID))
		return h.GiftDescriptionPrompt(userID)
	}

	// Сохраняем описание
	h.sessionManager.SetData(userID, "gift_description", description)
	h.sessionManager.SetState(userID, session.StateAwaitingGiftPhoto)

	return h.GiftPhotoPrompt(userID)
}

// HandleGiftPhoto обрабатывает фото подарка
func (h *GiftHandler) HandleGiftPhoto(userID int64, fileID string) string {
	count := h.AppendGiftPhoto(userID, fileID)
	return h.GiftPhotoAddedText(userID, count)
}

// AppendGiftPhoto добавляет фото к текущей сессии подарка и возвращает новое количество фото.
func (h *GiftHandler) AppendGiftPhoto(userID int64, fileID string) int {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		log.Printf("Gift photo input missing file id: user_id=%d state=%s", userID, h.sessionManager.GetState(userID))
		return h.giftPhotoCount(userID)
	}

	// Получаем существующие фото
	attachmentsRaw, ok := h.sessionManager.GetData(userID, "gift_attachments")
	var attachments []command.GiftAttachmentData
	if ok {
		existingAttachments, ok := attachmentsRaw.([]command.GiftAttachmentData)
		if !ok {
			log.Printf("Invalid gift session data: user_id=%d state=%s key=gift_attachments type=%T", userID, h.sessionManager.GetState(userID), attachmentsRaw)
		} else {
			attachments = existingAttachments
		}
	}

	// Добавляем новое фото
	attachments = append(attachments, command.GiftAttachmentData{
		TelegramFileID: fileID,
		FileType:       "photo",
	})

	h.sessionManager.SetData(userID, "gift_attachments", attachments)

	return len(attachments)
}

func (h *GiftHandler) giftPhotoCount(userID int64) int {
	attachmentsRaw, ok := h.sessionManager.GetData(userID, "gift_attachments")
	if !ok {
		return 0
	}

	attachments, ok := attachmentsRaw.([]command.GiftAttachmentData)
	if !ok {
		log.Printf("Invalid gift session data: user_id=%d state=%s key=gift_attachments type=%T", userID, h.sessionManager.GetState(userID), attachmentsRaw)
		return 0
	}

	return len(attachments)
}

// GiftPhotoAddedText возвращает текст подтверждения добавления фото.
func (h *GiftHandler) GiftPhotoAddedText(userID int64, photoCount int) string {
	texts := h.giftTexts(userID)
	return renderTelegramText(texts.GiftPhotoAdded, map[string]string{
		"photo_count": fmt.Sprintf("%d", photoCount),
	})
}

// GiftGenderPrompt возвращает текущую подсказку выбора пола подарка без изменения сессии.
func (h *GiftHandler) GiftGenderPrompt(userID int64) (string, *models.InlineKeyboardMarkup) {
	texts := h.giftTexts(userID)
	text := texts.GiftGenderStep

	keyboard := keyboard.NewBuilder().
		AddRow(
			keyboard.Button("👨 Мужской", "gift_gender_male"),
			keyboard.Button("👩 Женский", "gift_gender_female"),
		).
		AddRow(
			keyboard.Button("👥 Любой", "gift_gender_all"),
		).
		AddRow(
			keyboard.Button("❌ Отмена", "cancel"),
		).
		Build()

	return text, &keyboard
}

// GiftBikeTypePrompt возвращает текущую подсказку выбора велосипеда подарка без изменения сессии.
func (h *GiftHandler) GiftBikeTypePrompt(userID int64) (string, *models.InlineKeyboardMarkup) {
	texts := h.giftTexts(userID)
	text := texts.GiftBikeStep

	keyboard := keyboard.NewBuilder().
		AddRow(
			keyboard.Button("🚵 Гравийник", "gift_bike_gravel"),
			keyboard.Button("🏔 МТБ", "gift_bike_mtb"),
		).
		AddRow(
			keyboard.Button("🚴 Шоссе", "gift_bike_road"),
			keyboard.Button("⚡️ Фикс", "gift_bike_single_speed"),
		).
		AddRow(
			keyboard.Button("👥 Тандем", "gift_bike_tandem"),
		).
		AddRow(
			keyboard.Button("🚲 Любой", "gift_bike_all"),
		).
		AddRow(
			keyboard.Button("❌ Отмена", "cancel"),
		).
		Build()

	return text, &keyboard
}

// GiftDescriptionPrompt возвращает текущую подсказку ввода описания подарка без изменения сессии.
func (h *GiftHandler) GiftDescriptionPrompt(userID int64) (string, *models.InlineKeyboardMarkup) {
	texts := h.giftTexts(userID)
	text := texts.GiftDescriptionStep
	keyboard := keyboard.CancelMenu()
	return text, &keyboard
}

// GiftPhotoPrompt возвращает текущую подсказку добавления фото подарка без изменения сессии.
func (h *GiftHandler) GiftPhotoPrompt(userID int64) (string, *models.InlineKeyboardMarkup) {
	texts := h.giftTexts(userID)
	text := texts.GiftPhotoStep
	keyboard := keyboard.GiftPhotoMenu()
	return text, &keyboard
}

// GiftConfirmationPrompt возвращает подсказку подтверждения без изменения сессии.
func (h *GiftHandler) GiftConfirmationPrompt(userID int64) (string, *models.InlineKeyboardMarkup) {
	markup := keyboard.GiftConfirmationMenu()
	return "Приз уже заполнен. Подтвердите отправку кнопками ниже или отмените добавление.", &markup
}

// PreviewGift показывает сводку подарка и переводит сессию в ожидание подтверждения.
func (h *GiftHandler) PreviewGift(userID int64) (string, *models.InlineKeyboardMarkup) {
	data, message, ok := h.readGiftSessionData(userID)
	if !ok {
		h.sessionManager.ResetState(userID)
		return message, nil
	}

	h.sessionManager.SetState(userID, session.StateAwaitingGiftConfirmation)
	markup := keyboard.GiftConfirmationMenu()
	texts := h.giftTexts(userID)
	return buildGiftPreviewText(data, texts), &markup
}

// ConfirmAddGift сохраняет подарок после явного подтверждения пользователя.
func (h *GiftHandler) ConfirmAddGift(ctx context.Context, userID int64) (*entity.Gift, string, error) {
	data, message, ok := h.readGiftSessionData(userID)
	if !ok {
		h.sessionManager.ResetState(userID)
		return nil, message, nil
	}

	cmd := command.AddGiftCommand{
		UserID:         userID,
		EventID:        data.eventID,
		Description:    data.description,
		GenderFilter:   data.gender,
		BikeTypeFilter: data.bikeType,
		Attachments:    data.attachments,
	}

	gift, err := h.addGiftHandler.Handle(ctx, cmd)
	if err != nil {
		log.Printf("Gift confirmation save failed: user_id=%d event_id=%d error=%v", userID, data.eventID, err)
		return nil, fmt.Sprintf("Ошибка при добавлении приза: %v", err), err
	}

	texts := h.giftTexts(userID)

	photoText := ""
	if len(gift.Attachments) > 0 {
		photoText = fmt.Sprintf("\n• Фото: %d", len(gift.Attachments))
	}

	// Очищаем сессию
	h.sessionManager.ResetState(userID)

	return gift, renderTelegramText(texts.GiftSuccess, map[string]string{
		"gender":      giftGenderLabel(data.gender),
		"bike_type":   giftBikeTypeLabel(data.bikeType),
		"description": gift.Description,
		"photo_count": fmt.Sprintf("%d", len(gift.Attachments)),
		"photo_line":  photoText,
	}), nil
}

// RestartAddGift сбрасывает текущий ввод подарка и начинает процесс заново.
func (h *GiftHandler) RestartAddGift(ctx context.Context, userID int64) (string, *models.InlineKeyboardMarkup) {
	h.sessionManager.ResetState(userID)
	return h.StartAddGift(ctx, userID)
}

// CancelAddGift отменяет добавление подарка
func (h *GiftHandler) CancelAddGift(userID int64) string {
	texts := h.giftTexts(userID)
	h.sessionManager.ResetState(userID)
	return texts.GiftCancelled
}

type giftSessionData struct {
	eventID     uint
	gender      string
	bikeType    string
	description string
	attachments []command.GiftAttachmentData
}

func (h *GiftHandler) readGiftSessionData(userID int64) (giftSessionData, string, bool) {
	state := h.sessionManager.GetState(userID)
	restartMessage := h.giftTexts(userID).GiftSessionError

	eventIDRaw, ok := h.sessionManager.GetData(userID, "event_id")
	if !ok {
		log.Printf("Gift session data missing: user_id=%d state=%s key=event_id", userID, state)
		return giftSessionData{}, restartMessage, false
	}
	eventID, ok := giftEventIDFromSession(eventIDRaw)
	if !ok {
		log.Printf("Invalid gift session data: user_id=%d state=%s key=event_id type=%T", userID, state, eventIDRaw)
		return giftSessionData{}, restartMessage, false
	}

	genderRaw, ok := h.sessionManager.GetData(userID, "gift_gender")
	if !ok {
		log.Printf("Gift session data missing: user_id=%d state=%s key=gift_gender", userID, state)
		return giftSessionData{}, restartMessage, false
	}
	gender, ok := genderRaw.(string)
	if !ok || !isKnownGiftGender(gender) {
		log.Printf("Invalid gift session data: user_id=%d state=%s key=gift_gender type=%T", userID, state, genderRaw)
		return giftSessionData{}, restartMessage, false
	}

	bikeTypeRaw, ok := h.sessionManager.GetData(userID, "gift_bike_type")
	if !ok {
		log.Printf("Gift session data missing: user_id=%d state=%s key=gift_bike_type", userID, state)
		return giftSessionData{}, restartMessage, false
	}
	bikeType, ok := bikeTypeRaw.(string)
	if !ok || !isKnownGiftBikeType(bikeType) {
		log.Printf("Invalid gift session data: user_id=%d state=%s key=gift_bike_type type=%T", userID, state, bikeTypeRaw)
		return giftSessionData{}, restartMessage, false
	}

	descriptionRaw, ok := h.sessionManager.GetData(userID, "gift_description")
	if !ok {
		log.Printf("Gift session data missing: user_id=%d state=%s key=gift_description", userID, state)
		return giftSessionData{}, restartMessage, false
	}
	description, ok := descriptionRaw.(string)
	if !ok || description == "" {
		log.Printf("Invalid gift session data: user_id=%d state=%s key=gift_description type=%T", userID, state, descriptionRaw)
		return giftSessionData{}, restartMessage, false
	}

	var attachments []command.GiftAttachmentData
	attachmentsRaw, ok := h.sessionManager.GetData(userID, "gift_attachments")
	if ok {
		attachments, ok = attachmentsRaw.([]command.GiftAttachmentData)
		if !ok {
			log.Printf("Invalid gift session data: user_id=%d state=%s key=gift_attachments type=%T", userID, state, attachmentsRaw)
			return giftSessionData{}, restartMessage, false
		}
		for index, attachment := range attachments {
			if !isKnownGiftAttachmentFileType(attachment.FileType) {
				log.Printf("Invalid gift attachment file type in session: user_id=%d state=%s attachment_index=%d file_type=%s", userID, state, index, attachment.FileType)
				return giftSessionData{}, restartMessage, false
			}
		}
	}

	return giftSessionData{
		eventID:     eventID,
		gender:      gender,
		bikeType:    bikeType,
		description: description,
		attachments: attachments,
	}, "", true
}

func buildGiftPreviewText(data giftSessionData, texts entity.EventTelegramTexts) string {
	photoText := "нет"
	if len(data.attachments) > 0 {
		photoText = fmt.Sprintf("%d", len(data.attachments))
	}

	return renderTelegramText(texts.GiftPreview, map[string]string{
		"gender":      giftGenderLabel(data.gender),
		"bike_type":   giftBikeTypeLabel(data.bikeType),
		"description": data.description,
		"photo_count": photoText,
		"photo_line":  photoText,
	})
}

func (h *GiftHandler) giftTexts(userID int64) entity.EventTelegramTexts {
	textsRaw, ok := h.sessionManager.GetData(userID, "event_telegram_texts")
	if !ok {
		return entity.DefaultEventTelegramTexts()
	}

	texts, ok := textsRaw.(entity.EventTelegramTexts)
	if !ok {
		log.Printf("Invalid gift session data: user_id=%d state=%s key=event_telegram_texts type=%T", userID, h.sessionManager.GetState(userID), textsRaw)
		return entity.DefaultEventTelegramTexts()
	}
	return entity.NormalizeEventTelegramTexts(texts)
}

func renderTelegramText(text string, values map[string]string) string {
	replacements := make([]string, 0, len(values)*2)
	for key, value := range values {
		replacements = append(replacements, "{"+key+"}", value)
	}
	return strings.NewReplacer(replacements...).Replace(text)
}

func giftEventIDFromSession(value interface{}) (uint, bool) {
	switch v := value.(type) {
	case uint:
		return v, v > 0
	case int:
		if v > 0 {
			return uint(v), true
		}
	case int64:
		if v > 0 {
			return uint(v), true
		}
	case uint64:
		return uint(v), v > 0
	}
	return 0, false
}

func isKnownGiftGender(gender string) bool {
	switch gender {
	case "male", "female", "all":
		return true
	}
	return false
}

func isKnownGiftBikeType(bikeType string) bool {
	switch bikeType {
	case "gravel", "mtb", "road", "single_speed", "tandem", "all":
		return true
	}
	return false
}

func isKnownGiftAttachmentFileType(fileType string) bool {
	switch fileType {
	case "photo", "document":
		return true
	}
	return false
}

func giftGenderLabel(gender string) string {
	labels := map[string]string{
		"male":   "👨 Мужской",
		"female": "👩 Женский",
		"all":    "👥 Любой",
	}
	if label, ok := labels[gender]; ok {
		return label
	}
	return gender
}

func giftBikeTypeLabel(bikeType string) string {
	labels := map[string]string{
		"gravel":       "🚵 Гравийник",
		"mtb":          "🏔 МТБ",
		"road":         "🚴 Шоссе",
		"single_speed": "⚡️ Фикс",
		"tandem":       "👥 Тандем",
		"all":          "🚲 Любой",
	}
	if label, ok := labels[bikeType]; ok {
		return label
	}
	return bikeType
}
