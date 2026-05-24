package entity

import "time"

// Admin представляет администратора панели управления
type Admin struct {
	ID           uint
	Username     string
	PasswordHash string
	Role         AdminRole
	CreatedAt    time.Time
	LastLogin    *time.Time
}

// AdminRole представляет роль администратора
type AdminRole string

const (
	AdminRoleAdmin  AdminRole = "admin"
	AdminRoleViewer AdminRole = "viewer"
)

// IsAdmin проверяет, является ли пользователь полноправным администратором
func (a *Admin) IsAdmin() bool {
	return a.Role == AdminRoleAdmin
}

// CanEdit проверяет, может ли администратор редактировать данные
func (a *Admin) CanEdit() bool {
	return a.Role == AdminRoleAdmin
}
