package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
	"gravel_bot/internal/infrastructure/telegram/session"
)

// RegistrationHandler обрабатывает регистрацию участника
type RegistrationHandler struct {
	sessionManager             *session.Manager
	eventRepo                  repository.EventRepository
	participantRepo            repository.ParticipantRepository
	registerParticipantHandler *command.RegisterParticipantHandler
}

// NewRegistrationHandler создаёт новый handler
func NewRegistrationHandler(
	sessionManager *session.Manager,
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	registerParticipantHandler *command.RegisterParticipantHandler,
) *RegistrationHandler {
	return &RegistrationHandler{
		sessionManager:             sessionManager,
		eventRepo:                  eventRepo,
		participantRepo:            participantRepo,
		registerParticipantHandler: registerParticipantHandler,
	}
}

// StartRegistration начинает процесс регистрации
func (h *RegistrationHandler) StartRegistration(ctx context.Context, userID int64) (string, *models.InlineKeyboardMarkup) {
	// Получаем активное событие
	event, err := h.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("Error finding active event: %v", err)
		return "Произошла ошибка. Попробуйте позже.", nil
	}

	if event == nil {
		return "В данный момент нет активных событий.", nil
	}

	// Проверяем, не зарегистрирован ли уже участник
	participant, err := h.participantRepo.FindByUserAndEvent(ctx, userID, event.ID)
	if err == nil && participant != nil {
		// Участник уже зарегистрирован
		return "Вы уже зарегистрированы на это событие!", nil
	}
	// Если ошибка - значит участник не найден, это нормально, продолжаем регистрацию

	// Сохраняем ID события в сессии
	h.sessionManager.SetData(userID, "event_id", event.ID)
	h.sessionManager.SetState(userID, session.StateAwaitingBikeType)

	// Создаём клавиатуру с типами велосипедов
	keyboard := keyboard.BikeTypeMenu()

	return "Выберите тип велосипеда:", &keyboard
}

// HandleBikeTypeSelection обрабатывает выбор типа велосипеда
func (h *RegistrationHandler) HandleBikeTypeSelection(ctx context.Context, userID int64, bikeType string) (string, *models.InlineKeyboardMarkup) {
	// Сохраняем тип велосипеда
	h.sessionManager.SetData(userID, "bike_type", bikeType)
	h.sessionManager.SetState(userID, session.StateAwaitingGender)

	// Создаём клавиатуру с выбором пола
	keyboard := keyboard.GenderMenu()

	return "Выберите пол:", &keyboard
}

// HandleGenderSelection обрабатывает выбор пола и показывает условия участия.
func (h *RegistrationHandler) HandleGenderSelection(ctx context.Context, userID int64, gender string) (string, *models.InlineKeyboardMarkup) {
	eventIDRaw, ok := h.sessionManager.GetData(userID, "event_id")
	if !ok {
		return "Ошибка: данные сессии не найдены. Начните регистрацию заново.", nil
	}
	eventID, ok := eventIDRaw.(uint)
	if !ok {
		log.Printf("WARN Invalid registration session data: user_id=%d key=event_id type=%T", userID, eventIDRaw)
		h.sessionManager.ResetState(userID)
		return "Ошибка: данные сессии повреждены. Начните регистрацию заново.", nil
	}

	event, err := h.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		log.Printf("WARN Failed to load event for registration consent: user_id=%d event_id=%d error=%v", userID, eventID, err)
		return "Не удалось загрузить условия участия. Попробуйте позже.", nil
	}
	if event == nil {
		log.Printf("WARN Registration consent skipped: user_id=%d event_id=%d reason=event_missing", userID, eventID)
		return "Не удалось загрузить условия участия. Попробуйте позже.", nil
	}

	h.sessionManager.SetData(userID, "gender", gender)
	h.sessionManager.SetState(userID, session.StateAwaitingRegistrationConsent)
	log.Printf("INFO Participant registration conditions shown: telegram_user_id=%d event_id=%d gender=%s", userID, eventID, gender)

	markup := keyboard.RegistrationConsentMenu()
	text := fmt.Sprintf("%s\n\nПодтвердите, что принимаете условия участия.", EventConditionsText(event))
	return text, &markup
}

// ConfirmRegistration завершает регистрацию после согласия с условиями участия.
func (h *RegistrationHandler) ConfirmRegistration(ctx context.Context, userID int64) (string, error) {
	eventIDRaw, ok := h.sessionManager.GetData(userID, "event_id")
	if !ok {
		return "Ошибка: данные сессии не найдены. Начните регистрацию заново.", nil
	}
	eventID, ok := eventIDRaw.(uint)
	if !ok {
		log.Printf("WARN Invalid registration session data: user_id=%d key=event_id type=%T", userID, eventIDRaw)
		h.sessionManager.ResetState(userID)
		return "Ошибка: данные сессии повреждены. Начните регистрацию заново.", nil
	}

	bikeTypeRaw, ok := h.sessionManager.GetData(userID, "bike_type")
	if !ok {
		return "Ошибка: данные сессии не найдены. Начните регистрацию заново.", nil
	}
	bikeType, ok := bikeTypeRaw.(string)
	if !ok {
		log.Printf("WARN Invalid registration session data: user_id=%d key=bike_type type=%T", userID, bikeTypeRaw)
		h.sessionManager.ResetState(userID)
		return "Ошибка: данные сессии повреждены. Начните регистрацию заново.", nil
	}

	genderRaw, ok := h.sessionManager.GetData(userID, "gender")
	if !ok {
		return "Ошибка: данные сессии не найдены. Начните регистрацию заново.", nil
	}
	gender, ok := genderRaw.(string)
	if !ok {
		log.Printf("WARN Invalid registration session data: user_id=%d key=gender type=%T", userID, genderRaw)
		h.sessionManager.ResetState(userID)
		return "Ошибка: данные сессии повреждены. Начните регистрацию заново.", nil
	}
	log.Printf("INFO Participant registration conditions accepted: telegram_user_id=%d event_id=%d bike_type=%s gender=%s", userID, eventID, bikeType, gender)

	// Выполняем команду регистрации
	cmd := command.RegisterParticipantCommand{
		UserID:   userID,
		EventID:  eventID,
		BikeType: bikeType,
		Gender:   gender,
	}

	_, err := h.registerParticipantHandler.Handle(ctx, cmd)
	if err != nil {
		log.Printf("Error registering participant: telegram_user_id=%d event_id=%d bike_type=%s gender=%s error=%v", userID, eventID, bikeType, gender, err)
		return fmt.Sprintf("Ошибка при регистрации: %v", err), err
	}

	// Очищаем сессию
	h.sessionManager.ResetState(userID)

	bikeTypeText := map[string]string{
		"gravel":       "🚵 Гравийник",
		"mtb":          "🏔 МТБ",
		"road":         "🚴 Шоссе",
		"single_speed": "⚡️ Фикс",
		"tandem":       "👥 Тандем",
	}[bikeType]

	genderText := map[string]string{
		"male":   "👨 Мужской",
		"female": "👩 Женский",
	}[gender]

	return fmt.Sprintf(`✅ Регистрация успешно завершена!

📋 Ваши данные:
• Тип велосипеда: %s
• Пол: %s

🎉 Поздравляем! Вы зарегистрированы на мероприятие!

Теперь вы можете:
• Добавить приз в призовой фонд 🎁
• Отправить результат после заезда 🏁

💪 Желаем удачи на трассе! Увидимся на старте! 🚴✨`, bikeTypeText, genderText), nil
}

// DeclineRegistration отменяет регистрацию из-за отказа от условий участия.
func (h *RegistrationHandler) DeclineRegistration(userID int64) string {
	log.Printf("INFO Participant registration conditions declined: telegram_user_id=%d", userID)
	h.sessionManager.ResetState(userID)
	return "Регистрация отменена. Без согласия с условиями участие не оформлено."
}

// CancelRegistration отменяет регистрацию
func (h *RegistrationHandler) CancelRegistration(userID int64) string {
	h.sessionManager.ResetState(userID)
	return "Регистрация отменена."
}
