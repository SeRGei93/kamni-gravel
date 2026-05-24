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
	ErrGiftNotFound        = errors.New("gift not found")
	ErrGiftAlreadyAssigned = errors.New("gift already assigned to another participant")
	ErrParticipantNotFound = errors.New("participant not found")
)

// AssignPrizeCommand представляет команду назначения приза
type AssignPrizeCommand struct {
	ParticipantID uint
	GiftID        uint
	Comment       string
}

// AssignPrizeHandler обрабатывает назначение приза
type AssignPrizeHandler struct {
	participantRepo     repository.ParticipantRepository
	giftRepo            repository.GiftRepository
	prizeAssignmentRepo repository.PrizeAssignmentRepository
}

// NewAssignPrizeHandler создаёт новый handler
func NewAssignPrizeHandler(
	participantRepo repository.ParticipantRepository,
	giftRepo repository.GiftRepository,
	prizeAssignmentRepo repository.PrizeAssignmentRepository,
) *AssignPrizeHandler {
	return &AssignPrizeHandler{
		participantRepo:     participantRepo,
		giftRepo:            giftRepo,
		prizeAssignmentRepo: prizeAssignmentRepo,
	}
}

// Handle выполняет команду назначения приза
func (h *AssignPrizeHandler) Handle(ctx context.Context, cmd AssignPrizeCommand) (*entity.PrizeAssignment, error) {
	// Находим участника
	participant, err := h.participantRepo.FindByID(ctx, cmd.ParticipantID)
	if err != nil {
		return nil, ErrParticipantNotFound
	}

	// Находим подарок
	gift, err := h.giftRepo.FindByID(ctx, cmd.GiftID)
	if err != nil {
		return nil, ErrGiftNotFound
	}

	// Проверяем, что участник зарегистрирован на то же событие, что и подарок
	if participant.EventID != gift.EventID {
		return nil, errors.New("participant is not registered for this event")
	}

	// Проверяем, что подарок ещё не назначен
	// (можно добавить проверку через репозиторий, если нужно)

	// Создаём назначение приза
	assignment := &entity.PrizeAssignment{
		ParticipantID: cmd.ParticipantID,
		GiftID:        cmd.GiftID,
		Comment:       cmd.Comment,
		AssignedAt:    time.Now(),
		Participant:   participant,
		Gift:          gift,
	}

	// Сохраняем назначение
	if err := h.prizeAssignmentRepo.Assign(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to assign prize: %w", err)
	}

	return assignment, nil
}
