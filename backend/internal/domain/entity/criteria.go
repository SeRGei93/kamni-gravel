package entity

import (
	"time"
	
	"gravel_bot/internal/domain/valueobject"
)

// Criteria представляет критерий для подарков
type Criteria struct {
	ID           uint
	Name         string
	Description  string
	CriteriaType valueobject.CriteriaType
	CreatedAt    time.Time
}

// GiftCriteria представляет связь подарка с критерием
type GiftCriteria struct {
	ID         uint
	GiftID     uint
	CriteriaID uint
	
	// Связанные сущности
	Gift     *Gift
	Criteria *Criteria
}

// ResultCriteria представляет связь результата с критерием
type ResultCriteria struct {
	ID         uint
	ResultID   uint
	CriteriaID uint
	
	// Связанные сущности
	Result   *Result
	Criteria *Criteria
}
