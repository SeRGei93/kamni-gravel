package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
)

// StartHandler обрабатывает команду /start
type StartHandler struct {
	userRepo        repository.UserRepository
	eventRepo       repository.EventRepository
	participantRepo repository.ParticipantRepository
	miniappURL      string
}

// NewStartHandler создаёт новый handler
func NewStartHandler(
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	miniappURL string,
) *StartHandler {
	return &StartHandler{
		userRepo:        userRepo,
		eventRepo:       eventRepo,
		participantRepo: participantRepo,
		miniappURL:      miniappURL,
	}
}

// Handle обрабатывает команду /start
func (h *StartHandler) Handle(ctx context.Context, msg *models.Message) (string, *models.InlineKeyboardMarkup) {
	if msg == nil || msg.From == nil {
		log.Printf("Start command ignored: missing Telegram sender")
		return "Не удалось определить пользователя. Попробуйте отправить /start ещё раз.", nil
	}

	userID := msg.From.ID
	username := msg.From.Username
	firstName := msg.From.FirstName
	lastName := msg.From.LastName

	// Проверяем, существует ли пользователь
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		// Если пользователь не найден - создаём его
		log.Printf("User not found, creating new user: %d", userID)
		user = &entity.User{
			ID:        userID,
			Username:  username,
			FirstName: firstName,
			LastName:  lastName,
		}

		if err := h.userRepo.Create(ctx, user); err != nil {
			log.Printf("Error creating user: %v", err)
			return "Произошла ошибка при регистрации. Попробуйте позже.", nil
		}

		log.Printf("User created successfully: %d", userID)
	}

	// Получаем активное событие
	event, err := h.eventRepo.FindActive(ctx)
	if err != nil {
		log.Printf("Error finding active event: %v", err)
		return "Произошла ошибка. Попробуйте позже.", nil
	}

	if event == nil {
		return "В данный момент нет активных событий. Следите за обновлениями!", nil
	}

	payload := parseStartPayload(msg.Text)
	if payload == "conditions" {
		return EventConditionsText(event), nil
	}

	if payload != "" {
		log.Printf("WARN Unrecognized /start payload: user_id=%d payload=%q", userID, payload)
	}

	isRegistered := h.isUserRegisteredForActiveEvent(ctx, userID, event.ID)

	// Формируем приветственное сообщение
	text := fmt.Sprintf(`Привет, %s! 👋

Добро пожаловать в бот велогонки "%s"!

%s

Что ты хочешь сделать?`, firstName, event.Name, event.Description)

	markup := keyboard.MainMenu(true, isRegistered, h.miniappURL, nil)

	return text, &markup
}

func (h *StartHandler) isUserRegisteredForActiveEvent(ctx context.Context, userID int64, eventID uint) bool {
	if h.participantRepo == nil {
		return false
	}

	participant, err := h.participantRepo.FindByUserAndEvent(ctx, userID, eventID)
	if err == nil && participant != nil {
		return true
	}

	if err != nil && !errors.Is(err, repository.ErrParticipantNotFound) {
		log.Printf("WARN Failed to load participant status: user_id=%d event_id=%d error=%v", userID, eventID, err)
	}

	return false
}

func EventConditionsText(event *entity.Event) string {
	description := strings.TrimSpace(event.Description)
	if description == "" {
		return "Условия участия для этого события не заданы."
	}

	return description
}

func parseStartPayload(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || !strings.HasPrefix(strings.ToLower(text), "/start") {
		return ""
	}

	tail := strings.TrimSpace(text[len("/start"):])
	if tail == "" {
		return ""
	}

	if strings.HasPrefix(tail, "@") {
		parts := strings.Fields(tail)
		if len(parts) == 1 {
			return ""
		}
		tail = strings.TrimSpace(strings.Join(parts[1:], " "))
	}

	parts := strings.Fields(tail)
	if len(parts) == 0 {
		return ""
	}

	return strings.ToLower(parts[0])
}
