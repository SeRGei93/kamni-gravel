package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

var (
	ErrInvalidResultLink   = errors.New("invalid result link")
	ErrResultAlreadyExists = errors.New("result already submitted")
)

// SubmitResultCommand представляет команду отправки результата
type SubmitResultCommand struct {
	ParticipantID uint
	ResultLink    string
}

// SubmitResultHandler обрабатывает отправку результата
type SubmitResultHandler struct {
	participantRepo repository.ParticipantRepository
	resultRepo      repository.ResultRepository
}

// NewSubmitResultHandler создаёт новый handler
func NewSubmitResultHandler(
	participantRepo repository.ParticipantRepository,
	resultRepo repository.ResultRepository,
) *SubmitResultHandler {
	return &SubmitResultHandler{
		participantRepo: participantRepo,
		resultRepo:      resultRepo,
	}
}

// Handle выполняет команду отправки результата
func (h *SubmitResultHandler) Handle(ctx context.Context, cmd SubmitResultCommand) (*entity.Participant, error) {
	// Находим участника
	participant, err := h.participantRepo.FindByID(ctx, cmd.ParticipantID)
	if err != nil {
		return nil, ErrParticipantNotFound
	}

	// Валидируем ссылку на результат
	resultLink, err := valueobject.NewResultLink(cmd.ResultLink)
	if err != nil {
		return nil, ErrInvalidResultLink
	}

	// Создаём новый результат
	result := &entity.Result{
		ParticipantID: participant.ID,
		ResultLink:    resultLink,
		IsCurrent:     true,
		SubmittedAt:   time.Now(),
	}

	// Сохраняем результат (он автоматически пометит старые как неактуальные)
	if err := h.resultRepo.Create(ctx, result); err != nil {
		return nil, fmt.Errorf("failed to create result: %w", err)
	}

	// Привязываем результат к участнику для возврата
	participant.Result = result

	return participant, nil
}
