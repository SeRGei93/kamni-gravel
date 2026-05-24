package dto

import (
	"time"
	
	"gravel_bot/internal/domain/entity"
)

// CriteriaDTO представляет DTO критерия для API
type CriteriaDTO struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CriteriaType string    `json:"criteria_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// FromCriteria создаёт DTO из entity.Criteria
func FromCriteria(c *entity.Criteria) *CriteriaDTO {
	return &CriteriaDTO{
		ID:           c.ID,
		Name:         c.Name,
		Description:  c.Description,
		CriteriaType: c.CriteriaType.String(),
		CreatedAt:    c.CreatedAt,
	}
}

// CriteriaListResponse представляет ответ со списком критериев
type CriteriaListResponse struct {
	Criteria []*CriteriaDTO `json:"criteria"`
	Total    int            `json:"total"`
}
