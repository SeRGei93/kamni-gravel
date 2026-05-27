package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEventNotFound     = errors.New("event not found")
	ErrEventNotActive    = errors.New("event is not active")
	ErrAlreadyRegistered = errors.New("user already registered for this event")
	ErrInvalidBikeType   = errors.New("invalid bike type")
	ErrInvalidGender     = errors.New("invalid gender")
)

// RegisterParticipantCommand представляет команду регистрации участника на событие
type RegisterParticipantCommand struct {
	UserID   int64
	EventID  uint
	BikeType string
	Gender   string
}

// RegisterParticipantHandler обрабатывает регистрацию участника
type RegisterParticipantHandler struct {
	userRepo          repository.UserRepository
	eventRepo         repository.EventRepository
	participantRepo   repository.ParticipantRepository
	userBlacklistRepo repository.UserBlacklistRepository
}

// NewRegisterParticipantHandler создаёт новый handler
func NewRegisterParticipantHandler(
	userRepo repository.UserRepository,
	eventRepo repository.EventRepository,
	participantRepo repository.ParticipantRepository,
	userBlacklistRepo repository.UserBlacklistRepository,
) *RegisterParticipantHandler {
	return &RegisterParticipantHandler{
		userRepo:          userRepo,
		eventRepo:         eventRepo,
		participantRepo:   participantRepo,
		userBlacklistRepo: userBlacklistRepo,
	}
}

// Handle выполняет команду регистрации участника
func (h *RegisterParticipantHandler) Handle(ctx context.Context, cmd RegisterParticipantCommand) (*entity.Participant, error) {
	isBlacklisted, err := h.userBlacklistRepo.IsBlacklisted(ctx, cmd.UserID)
	if err != nil {
		log.Printf("ERROR Participant registration blacklist check failed: telegram_user_id=%d event_id=%d error=%v", cmd.UserID, cmd.EventID, err)
		return nil, fmt.Errorf("check user blacklist: %w", err)
	}
	if isBlacklisted {
		log.Printf("WARN Participant registration blocked: telegram_user_id=%d event_id=%d reason=blacklisted", cmd.UserID, cmd.EventID)
		return nil, ErrUserBlacklisted
	}

	// Проверяем существование пользователя
	user, err := h.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Проверяем существование и активность события
	event, err := h.eventRepo.FindByID(ctx, cmd.EventID)
	if err != nil {
		return nil, ErrEventNotFound
	}
	if !event.Active {
		return nil, ErrEventNotActive
	}

	// Проверяем, не зарегистрирован ли уже участник
	existing, err := h.participantRepo.FindByUserAndEvent(ctx, cmd.UserID, cmd.EventID)
	if err == nil && existing != nil {
		// Участник уже зарегистрирован
		return nil, ErrAlreadyRegistered
	}
	// Если ошибка - значит участник не найден, это нормально, продолжаем регистрацию

	// Валидируем и создаём BikeType
	bikeType, err := valueobject.NewBikeType(cmd.BikeType)
	if err != nil {
		return nil, ErrInvalidBikeType
	}

	// Валидируем и создаём Gender
	gender, err := valueobject.NewGender(cmd.Gender)
	if err != nil {
		return nil, ErrInvalidGender
	}

	// Создаём участника
	participant := &entity.Participant{
		UserID:       cmd.UserID,
		EventID:      cmd.EventID,
		BikeType:     bikeType,
		Gender:       gender,
		RegisteredAt: time.Now(),
		User:         user,
	}

	// Сохраняем в БД
	if err := h.participantRepo.Create(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to create participant: %w", err)
	}

	return participant, nil
}
