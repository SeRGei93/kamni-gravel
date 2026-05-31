package dto

import (
	"time"

	"gravel_bot/internal/domain/entity"
)

// AdminDTO представляет администратора для API без чувствительных полей.
type AdminDTO struct {
	ID        uint       `json:"id"`
	Username  string     `json:"username"`
	Role      string     `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	LastLogin *time.Time `json:"last_login"`
}

// FromAdmin создаёт DTO из entity.Admin.
func FromAdmin(admin *entity.Admin) *AdminDTO {
	return &AdminDTO{
		ID:        admin.ID,
		Username:  admin.Username,
		Role:      string(admin.Role),
		CreatedAt: admin.CreatedAt,
		LastLogin: admin.LastLogin,
	}
}

// FromAdmins создаёт список DTO из списка entity.Admin.
func FromAdmins(admins []*entity.Admin) []*AdminDTO {
	result := make([]*AdminDTO, 0, len(admins))
	for _, admin := range admins {
		result = append(result, FromAdmin(admin))
	}
	return result
}

// AdminListResponse представляет ответ со списком администраторов.
type AdminListResponse struct {
	Admins []*AdminDTO `json:"admins"`
	Total  int         `json:"total"`
}
