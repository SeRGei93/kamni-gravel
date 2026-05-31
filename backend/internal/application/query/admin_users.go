package query

import (
	"context"
	"fmt"
	"log"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// ListAdminUsersHandler обрабатывает запрос списка администраторов.
type ListAdminUsersHandler struct {
	adminRepo repository.AdminRepository
}

// NewListAdminUsersHandler создаёт новый handler.
func NewListAdminUsersHandler(adminRepo repository.AdminRepository) *ListAdminUsersHandler {
	return &ListAdminUsersHandler{adminRepo: adminRepo}
}

// Handle выполняет запрос списка администраторов.
func (h *ListAdminUsersHandler) Handle(ctx context.Context) ([]*entity.Admin, error) {
	admins, err := h.adminRepo.List(ctx)
	if err != nil {
		log.Printf("ERROR Admin users list failed: operation=list_admin_users error=%v", err)
		return nil, fmt.Errorf("list admin users: %w", err)
	}

	return admins, nil
}
