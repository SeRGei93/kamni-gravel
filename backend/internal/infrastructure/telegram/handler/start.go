package handler

import (
	"context"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// StartHandler обрабатывает команду /start
type StartHandler struct {
	userRepo  repository.UserRepository
	eventRepo repository.EventRepository
}

// NewStartHandler создаёт новый handler
func NewStartHandler(
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
) *StartHandler {
	return &StartHandler{
		userRepo:  userRepo,
		eventRepo: eventRepo,
	}
}

// Handle обрабатывает команду /start
func (h *StartHandler) Handle(ctx context.Context, msg *tgbotapi.Message) (string, *tgbotapi.InlineKeyboardMarkup) {
	userID := msg.From.ID
	username := msg.From.UserName
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
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚴 Зарегистрироваться", "register"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Добавить подарок", "add_gift"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏁 Отправить результат", "submit_result"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Информация", "info"),
		),
	)

	return text, &keyboard
}
