package repository

import (
	"context"
	
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"
)

// CriteriaRepository представляет репозиторий для работы с критериями
type CriteriaRepository interface {
	// Create создаёт новый критерий
	Create(ctx context.Context, criteria *entity.Criteria) error
	
	// Update обновляет существующий критерий
	Update(ctx context.Context, criteria *entity.Criteria) error
	
	// Delete удаляет критерий
	Delete(ctx context.Context, id uint) error
	
	// FindByID находит критерий по ID
	FindByID(ctx context.Context, id uint) (*entity.Criteria, error)
	
	// FindAll возвращает все критерии
	FindAll(ctx context.Context) ([]*entity.Criteria, error)
	
	// FindByType возвращает критерии по типу
	FindByType(ctx context.Context, criteriaType valueobject.CriteriaType) ([]*entity.Criteria, error)
	
	// FindByGift возвращает критерии, привязанные к подарку
	FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error)

	// FindByResult возвращает критерии, привязанные к результату
	FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error)
}

// GiftCriteriaRepository представляет репозиторий для связи подарков и критериев
type GiftCriteriaRepository interface {
	// AddCriteriaToGift добавляет критерий к подарку
	AddCriteriaToGift(ctx context.Context, giftID uint, criteriaID uint) error

	// RemoveCriteriaFromGift удаляет критерий от подарка
	RemoveCriteriaFromGift(ctx context.Context, giftID uint, criteriaID uint) error

	// RemoveAllCriteriaFromGift удаляет все критерии от подарка
	RemoveAllCriteriaFromGift(ctx context.Context, giftID uint) error

	// FindByGift возвращает все связи для подарка
	FindByGift(ctx context.Context, giftID uint) ([]*entity.GiftCriteria, error)

	// FindByCriteria возвращает все связи для критерия
	FindByCriteria(ctx context.Context, criteriaID uint) ([]*entity.GiftCriteria, error)
}
