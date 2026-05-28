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
	log.Printf("INFO Participant registration requested: telegram_user_id=%d event_id=%d bike_type=%q gender=%q", cmd.UserID, cmd.EventID, cmd.BikeType, cmd.Gender)

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
		log.Printf("WARN Participant registration failed: telegram_user_id=%d event_id=%d stage=find_user error=%v", cmd.UserID, cmd.EventID, err)
		return nil, ErrUserNotFound
	}

	// Проверяем существование и активность события
	event, err := h.eventRepo.FindByID(ctx, cmd.EventID)
	if err != nil {
		log.Printf("WARN Participant registration failed: telegram_user_id=%d event_id=%d stage=find_event error=%v", cmd.UserID, cmd.EventID, err)
		return nil, ErrEventNotFound
	}
	if !event.Active {
		log.Printf("INFO Participant registration blocked: telegram_user_id=%d event_id=%d stage=validate_event reason=event_inactive", cmd.UserID, cmd.EventID)
		return nil, ErrEventNotActive
	}

	// Проверяем, не зарегистрирован ли уже участник
	existing, err := h.participantRepo.FindByUserAndEvent(ctx, cmd.UserID, cmd.EventID)
	if err == nil && existing != nil {
		// Участник уже зарегистрирован
		log.Printf("INFO Participant registration skipped: telegram_user_id=%d event_id=%d participant_id=%d reason=already_registered", cmd.UserID, cmd.EventID, existing.ID)
		return nil, ErrAlreadyRegistered
	}
	// Если ошибка - значит участник не найден, это нормально, продолжаем регистрацию
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			log.Printf("INFO Participant registration continuing: telegram_user_id=%d event_id=%d stage=find_existing_participant reason=not_registered", cmd.UserID, cmd.EventID)
		} else {
			log.Printf("WARN Participant registration continuing after existing participant lookup error: telegram_user_id=%d event_id=%d stage=find_existing_participant error=%v", cmd.UserID, cmd.EventID, err)
		}
	}

	// Валидируем и создаём BikeType
	bikeType, err := valueobject.NewBikeType(cmd.BikeType)
	if err != nil {
		log.Printf("WARN Participant registration failed: telegram_user_id=%d event_id=%d stage=validate_bike_type bike_type=%q error=%v", cmd.UserID, cmd.EventID, cmd.BikeType, err)
		return nil, ErrInvalidBikeType
	}

	// Валидируем и создаём Gender
	gender, err := valueobject.NewGender(cmd.Gender)
	if err != nil {
		log.Printf("WARN Participant registration failed: telegram_user_id=%d event_id=%d stage=validate_gender gender=%q error=%v", cmd.UserID, cmd.EventID, cmd.Gender, err)
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
		log.Printf("ERROR Participant registration failed: telegram_user_id=%d event_id=%d stage=create_participant bike_type=%s gender=%s error=%v", cmd.UserID, cmd.EventID, bikeType, gender, err)
		return nil, fmt.Errorf("failed to create participant: %w", err)
	}

	log.Printf("INFO Participant registration completed: telegram_user_id=%d event_id=%d participant_id=%d bike_type=%s gender=%s", cmd.UserID, cmd.EventID, participant.ID, bikeType, gender)
	return participant, nil
}
