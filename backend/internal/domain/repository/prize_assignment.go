package repository

import (
	"context"
	"gravel_bot/internal/domain/entity"
)

// PrizeAssignmentRepository определяет интерфейс для работы с назначениями призов
type PrizeAssignmentRepository interface {
	// Assign назначает приз участнику
	Assign(ctx context.Context, assignment *entity.PrizeAssignment) error
	
	// Update обновляет назначение
	Update(ctx context.Context, assignment *entity.PrizeAssignment) error
	
	// FindByID находит назначение по ID
	FindByID(ctx context.Context, id uint) (*entity.PrizeAssignment, error)
	
	// FindByParticipant находит все призы участника
	FindByParticipant(ctx context.Context, participantID uint) ([]*entity.PrizeAssignment, error)
	
	// FindByEvent находит все назначения призов события
	FindByEvent(ctx context.Context, eventID uint) ([]*entity.PrizeAssignment, error)
	
	// Remove удаляет назначение приза
	Remove(ctx context.Context, id uint) error
}
