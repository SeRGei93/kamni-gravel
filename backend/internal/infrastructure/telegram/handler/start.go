package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/telegram/keyboard"
)

// StartHandler обрабатывает команду /start
type StartHandler struct {
	userRepo   repository.UserRepository
	eventRepo  repository.EventRepository
	miniappURL string
}

// NewStartHandler создаёт новый handler
func NewStartHandler(
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
	miniappURL string,
) *StartHandler {
	return &StartHandler{
		userRepo:   userRepo,
		eventRepo:  eventRepo,
		miniappURL: miniappURL,
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

	// Формируем приветственное сообщение
	text := fmt.Sprintf(`Привет, %s! 👋

Добро пожаловать в бот велогонки "%s"!

%s

Что ты хочешь сделать?`, firstName, event.Name, event.Description)

	// Создаём клавиатуру с действиями
	markup := keyboard.MainMenu(h.miniappURL)

	return text, &markup
}
