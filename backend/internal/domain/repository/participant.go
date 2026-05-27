package repository

import (
	"context"
	"errors"

	"gravel_bot/internal/domain/entity"
)

// ErrParticipantNotFound означает, что участник не найден.
var ErrParticipantNotFound = errors.New("participant not found")

// ParticipantRepository определяет интерфейс для работы с участниками
type ParticipantRepository interface {
	// Create создаёт нового участника
	Create(ctx context.Context, participant *entity.Participant) error

	// Update обновляет данные участника (только поля participants, не results)
	Update(ctx context.Context, participant *entity.Participant) error

	// FindByID находит участника по ID (с результатом через LEFT JOIN)
	FindByID(ctx context.Context, id uint) (*entity.Participant, error)

	// FindByUserAndEvent находит участника по user_id и event_id (с результатом)
	FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error)

	// FindByEvent находит всех участников события (с результатами)
	FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error)

	// UpdateNotes обновляет заметки участника
	UpdateNotes(ctx context.Context, id uint, notes string) error

	// Delete удаляет участника
	Delete(ctx context.Context, id uint) error

	// DeleteWithResultCriteria удаляет критерии результатов участника и участника транзакционно
	DeleteWithResultCriteria(ctx context.Context, id uint) error

	// GetFinishedByEvent возвращает финишировавших участников события, отсортированных по времени
	GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error)
}
