package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

var (
	ErrEmptyDescription          = errors.New("gift description cannot be empty")
	ErrInvalidAttachmentFileType = errors.New("gift attachment file type must be photo or document")
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
	userRepo          repository.UserRepository
	eventRepo         repository.EventRepository
	giftRepo          repository.GiftRepository
	userBlacklistRepo repository.UserBlacklistRepository
}

// NewAddGiftHandler создаёт новый handler
func NewAddGiftHandler(
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
	giftRepo repository.GiftRepository,
	userBlacklistRepo repository.UserBlacklistRepository,
) *AddGiftHandler {
	return &AddGiftHandler{
		userRepo:          userRepo,
		eventRepo:         eventRepo,
		giftRepo:          giftRepo,
		userBlacklistRepo: userBlacklistRepo,
	}
}

// Handle выполняет команду добавления подарка
func (h *AddGiftHandler) Handle(ctx context.Context, cmd AddGiftCommand) (*entity.Gift, error) {
	isBlacklisted, err := h.userBlacklistRepo.IsBlacklisted(ctx, cmd.UserID)
	if err != nil {
		log.Printf("ERROR Gift creation blacklist check failed: telegram_user_id=%d event_id=%d error=%v", cmd.UserID, cmd.EventID, err)
		return nil, fmt.Errorf("check user blacklist: %w", err)
	}
	if isBlacklisted {
		log.Printf("WARN Gift creation blocked: telegram_user_id=%d event_id=%d reason=blacklisted", cmd.UserID, cmd.EventID)
		return nil, ErrUserBlacklisted
	}

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
		ReviewStatus:   entity.GiftReviewStatusPendingReview,
		CreatedAt:      time.Now(),
		User:           user,
	}

	// Готовим прикреплённые файлы до создания domain entities с невалидным типом.
	attachments := make([]*entity.GiftAttachment, 0, len(cmd.Attachments))
	for _, attachData := range cmd.Attachments {
		if !isValidGiftAttachmentFileType(attachData.FileType) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidAttachmentFileType, attachData.FileType)
		}
		attachments = append(attachments, &entity.GiftAttachment{
			TelegramFileID: attachData.TelegramFileID,
			FileType:       attachData.FileType,
		})
	}

	// Сохраняем подарок и файлы атомарно.
	if err := h.giftRepo.CreateWithAttachments(ctx, gift, attachments); err != nil {
		return nil, fmt.Errorf("failed to create gift with attachments: %w", err)
	}

	gift.Attachments = make([]entity.GiftAttachment, len(attachments))
	for i, attachment := range attachments {
		gift.Attachments[i] = *attachment
	}

	return gift, nil
}

func isValidGiftAttachmentFileType(fileType string) bool {
	switch fileType {
	case "photo", "document":
		return true
	}
	return false
}
