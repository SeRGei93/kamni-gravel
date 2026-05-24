package handler

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/repository"
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
func (h *GiftHandler) StartAddGift(ctx context.Context, userID int64) (string, *tgbotapi.InlineKeyboardMarkup) {
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
	h.sessionManager.SetState(userID, session.StateAwaitingGiftGender)

	text := `🎁 Добавление подарка

Шаг 1/4: Выберите пол участника`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Мужской", "gift_gender_male"),
			tgbotapi.NewInlineKeyboardButtonData("👩 Женский", "gift_gender_female"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Любой", "gift_gender_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)

	return text, &keyboard
}

// HandleGiftGenderSelection обрабатывает выбор гендера для подарка
func (h *GiftHandler) HandleGiftGenderSelection(ctx context.Context, userID int64, gender string) (string, *tgbotapi.InlineKeyboardMarkup) {
	// Сохраняем гендер
	h.sessionManager.SetData(userID, "gift_gender", gender)
	h.sessionManager.SetState(userID, session.StateAwaitingGiftBikeType)

	text := `🎁 Добавление подарка

Шаг 2/4: Выберите тип велосипеда`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚵 Гравийник", "gift_bike_gravel"),
			tgbotapi.NewInlineKeyboardButtonData("🏔 МТБ", "gift_bike_mtb"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚴 Шоссе", "gift_bike_road"),
			tgbotapi.NewInlineKeyboardButtonData("⚡️ Фикс", "gift_bike_single_speed"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Тандем", "gift_bike_tandem"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚲 Любой", "gift_bike_all"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)

	return text, &keyboard
}

// HandleGiftBikeTypeSelection обрабатывает выбор типа велосипеда для подарка
func (h *GiftHandler) HandleGiftBikeTypeSelection(ctx context.Context, userID int64, bikeType string) (string, *tgbotapi.InlineKeyboardMarkup) {
	// Сохраняем тип велосипеда
	h.sessionManager.SetData(userID, "gift_bike_type", bikeType)
	h.sessionManager.SetState(userID, session.StateAwaitingGiftDesc)

	text := `🎁 Добавление подарка

Шаг 3/4: Отправьте описание подарка.

Укажите за что этот подарок (номинацию) и что именно вы дарите.

Примеры:
• Самый быстрый на гревеле - Парафиновая смазка Мамкина забота
• Выпито больше всего пива на маршруте - Упаковка кислых червячков
• Лучшее фото у камней - Топкеп Спаси и сохрани
• Последнее место в общем зачете - Проездной на общественный транспорт
• Бутылка водки "Налибоки" за первое место МТБ
• Первое место абсолют - Кирпич`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)

	return text, &keyboard
}

// HandleGiftDescription обрабатывает описание подарка
func (h *GiftHandler) HandleGiftDescription(ctx context.Context, userID int64, description string) (string, *tgbotapi.InlineKeyboardMarkup) {
	// Сохраняем описание
	h.sessionManager.SetData(userID, "gift_description", description)
	h.sessionManager.SetState(userID, session.StateAwaitingGiftPhoto)

	text := `🎁 Добавление подарка

Шаг 4/4: Отправьте фото подарка (можно несколько).

Когда закончите, нажмите "Завершить" или "Пропустить", если фото нет.`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Завершить", "finish_gift"),
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "skip_photos"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		),
	)

	return text, &keyboard
}

// HandleGiftPhoto обрабатывает фото подарка
func (h *GiftHandler) HandleGiftPhoto(userID int64, fileID string) string {
	// Получаем существующие фото
	attachmentsRaw, ok := h.sessionManager.GetData(userID, "gift_attachments")
	var attachments []command.GiftAttachmentData
	if ok {
		attachments = attachmentsRaw.([]command.GiftAttachmentData)
	}

	// Добавляем новое фото
	attachments = append(attachments, command.GiftAttachmentData{
		TelegramFileID: fileID,
		FileType:       "photo",
	})

	h.sessionManager.SetData(userID, "gift_attachments", attachments)

	return fmt.Sprintf("Фото добавлено! Всего фото: %d. Отправьте ещё или нажмите \"Завершить\".", len(attachments))
}

// FinishAddGift завершает добавление подарка
func (h *GiftHandler) FinishAddGift(ctx context.Context, userID int64) (string, error) {
	// Получаем сохранённые данные
	eventIDRaw, ok := h.sessionManager.GetData(userID, "event_id")
	if !ok {
		return "Ошибка: данные сессии не найдены. Начните добавление подарка заново.", nil
	}
	eventID := eventIDRaw.(uint)

	genderRaw, ok := h.sessionManager.GetData(userID, "gift_gender")
	if !ok {
		return "Ошибка: гендер подарка не найден. Начните заново.", nil
	}
	gender := genderRaw.(string)

	bikeTypeRaw, ok := h.sessionManager.GetData(userID, "gift_bike_type")
	if !ok {
		return "Ошибка: тип велосипеда не найден. Начните заново.", nil
	}
	bikeType := bikeTypeRaw.(string)

	descriptionRaw, ok := h.sessionManager.GetData(userID, "gift_description")
	if !ok {
		return "Ошибка: описание подарка не найдено. Начните заново.", nil
	}
	description := descriptionRaw.(string)

	// Получаем прикреплённые файлы (если есть)
	var attachments []command.GiftAttachmentData
	attachmentsRaw, ok := h.sessionManager.GetData(userID, "gift_attachments")
	if ok {
		attachments = attachmentsRaw.([]command.GiftAttachmentData)
	}

	// Выполняем команду добавления подарка
	cmd := command.AddGiftCommand{
		UserID:         userID,
		EventID:        eventID,
		Description:    description,
		GenderFilter:   gender,
		BikeTypeFilter: bikeType,
		Attachments:    attachments,
	}

	gift, err := h.addGiftHandler.Handle(ctx, cmd)
	if err != nil {
		log.Printf("Error adding gift: %v", err)
		return fmt.Sprintf("Ошибка при добавлении подарка: %v", err), err
	}

	// Очищаем сессию
	h.sessionManager.ResetState(userID)

	// Форматируем гендер для отображения
	genderText := map[string]string{
		"male":   "👨 Мужской",
		"female": "👩 Женский",
		"all":    "👥 Любой",
	}[gender]

	// Форматируем тип велосипеда для отображения
	bikeTypeText := map[string]string{
		"gravel":       "🚵 Гравийник",
		"mtb":          "🏔 МТБ",
		"road":         "🚴 Шоссе",
		"single_speed": "⚡️ Фикс",
		"tandem":       "👥 Тандем",
		"all":          "🚲 Любой",
	}[bikeType]

	photoText := ""
	if len(gift.Attachments) > 0 {
		photoText = fmt.Sprintf("\n• Фото: %d", len(gift.Attachments))
	}

	return fmt.Sprintf(`✅ Подарок успешно добавлен в призовой фонд!

📋 Детали подарка:
• Пол участника: %s
• Тип велосипеда: %s
• Описание: %s%s

🙏 Огромное спасибо за ваш вклад! 
Вы делаете наше мероприятие ещё лучше! 🎁✨

Ваш подарок будет разыгран среди участников по итогам заезда.`, genderText, bikeTypeText, gift.Description, photoText), nil
}

// CancelAddGift отменяет добавление подарка
func (h *GiftHandler) CancelAddGift(userID int64) string {
	h.sessionManager.ResetState(userID)
	return "Добавление подарка отменено."
}
