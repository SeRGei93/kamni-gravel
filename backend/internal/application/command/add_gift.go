package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

var (
	ErrEmptyDescription = errors.New("gift description cannot be empty")
)

// AddGiftCommand представляет команду добавления подарка
type AddGiftCommand struct {
	UserID         int64
	EventID        uint
	Description    string
	GenderFilter   string // male, female, all
	BikeTypeFilter string // gravel, mtb, road, single_speed, tandem, all
	Attachments    []GiftAttachmentData
}

// GiftAttachmentData представляет данные прикреплённого файла
type GiftAttachmentData struct {
	TelegramFileID string
	FileType       string // photo, document
}

// AddGiftHandler обрабатывает добавление подарка
type AddGiftHandler struct {
	userRepo  repository.UserRepository
	eventRepo repository.EventRepository
	giftRepo  repository.GiftRepository
}

// NewAddGiftHandler создаёт новый handler
func NewAddGiftHandler(
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
	giftRepo repository.GiftRepository,
) *AddGiftHandler {
	return &AddGiftHandler{
		userRepo:  userRepo,
		eventRepo: eventRepo,
		giftRepo:  giftRepo,
	}
}

// Handle выполняет команду добавления подарка
func (h *AddGiftHandler) Handle(ctx context.Context, cmd AddGiftCommand) (*entity.Gift, error) {
	// Проверяем существование пользователя
	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Проверяем существование события
	_, err = h.eventRepo.FindByID(ctx, cmd.EventID)
	if err != nil {
		return nil, ErrEventNotFound
	}

	// Валидация описания
	if cmd.Description == "" {
		return nil, ErrEmptyDescription
	}

	// Устанавливаем значения по умолчанию, если не указаны
	genderFilter := cmd.GenderFilter
	if genderFilter == "" {
		genderFilter = "all"
	}
	bikeTypeFilter := cmd.BikeTypeFilter
	if bikeTypeFilter == "" {
		bikeTypeFilter = "all"
	}

	// Создаём подарок
	gift := &entity.Gift{
		UserID:         cmd.UserID,
		EventID:        cmd.EventID,
		Description:    cmd.Description,
		GenderFilter:   genderFilter,
		BikeTypeFilter: bikeTypeFilter,
		CreatedAt:      time.Now(),
		User:           user,
	}

	// Сохраняем подарок в БД
	if err := h.giftRepo.Create(ctx, gift); err != nil {
		return nil, fmt.Errorf("failed to create gift: %w", err)
	}

	// Добавляем прикреплённые файлы
	for _, attachData := range cmd.Attachments {
		attachment := &entity.GiftAttachment{
			GiftID:         gift.ID,
			TelegramFileID: attachData.TelegramFileID,
			FileType:       attachData.FileType,
		}
		if err := h.giftRepo.AddAttachment(ctx, attachment); err != nil {
			return nil, fmt.Errorf("failed to add attachment: %w", err)
		}
		gift.Attachments = append(gift.Attachments, *attachment)
	}

	return gift, nil
}
