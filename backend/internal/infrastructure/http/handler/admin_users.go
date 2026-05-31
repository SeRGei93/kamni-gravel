package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/infrastructure/http/response"
)

// AdminUsersHandler обрабатывает admin API для управления администраторами.
type AdminUsersHandler struct {
	listHandler           *query.ListAdminUsersHandler
	createHandler         *command.CreateAdminHandler
	changePasswordHandler *command.ChangeAdminPasswordHandler
}

// NewAdminUsersHandler создаёт новый handler.
func NewAdminUsersHandler(
	listHandler *query.ListAdminUsersHandler,
	createHandler *command.CreateAdminHandler,
	changePasswordHandler *command.ChangeAdminPasswordHandler,
) *AdminUsersHandler {
	return &AdminUsersHandler{
		listHandler:           listHandler,
		createHandler:         createHandler,
		changePasswordHandler: changePasswordHandler,
	}
}

// CreateAdminRequest представляет запрос создания администратора.
type CreateAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ChangePasswordRequest представляет запрос смены пароля текущего администратора.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// GetAll обрабатывает GET /api/admin-users.
func (h *AdminUsersHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	admins, err := h.listHandler.Handle(r.Context())
	if err != nil {
		log.Printf("ERROR Admin users list failed in HTTP: operation=list_admin_users error=%v", err)
		response.InternalServerError(w, "Failed to get admin users")
		return
	}

	response.Success(w, dto.AdminListResponse{
		Admins: dto.FromAdmins(admins),
		Total:  len(admins),
	})
}

// Create обрабатывает POST /api/admin-users.
func (h *AdminUsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("WARN Admin user create failed: operation=decode_body error=%v", err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	admin, err := h.createHandler.Handle(r.Context(), command.CreateAdminCommand{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, command.ErrAdminUsernameRequired),
			errors.Is(err, command.ErrAdminPasswordRequired),
			errors.Is(err, command.ErrAdminPasswordTooShort):
			response.BadRequest(w, err.Error())
		case errors.Is(err, command.ErrAdminUsernameTaken):
			response.Conflict(w, err.Error())
		default:
			log.Printf("ERROR Admin user create failed in HTTP: operation=create_admin username=%q error=%v", req.Username, err)
			response.InternalServerError(w, "Failed to create admin user")
		}
		return
	}

	response.Created(w, dto.FromAdmin(admin))
}

// ChangeOwnPassword обрабатывает PUT /api/auth/me/password.
func (h *AdminUsersHandler) ChangeOwnPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		log.Printf("WARN Admin password change failed: operation=auth_context_missing")
		response.Unauthorized(w, "User not found in context")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("WARN Admin password change failed: operation=decode_body admin_id=%d error=%v", claims.UserID, err)
		response.BadRequest(w, "Invalid request body")
		return
	}

	err := h.changePasswordHandler.Handle(r.Context(), command.ChangeAdminPasswordCommand{
		AdminID:         claims.UserID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		switch {
		case errors.Is(err, command.ErrAdminPasswordRequired),
			errors.Is(err, command.ErrAdminPasswordTooShort):
			response.BadRequest(w, err.Error())
		case errors.Is(err, command.ErrCurrentPasswordInvalid):
			log.Printf("WARN Admin password change failed: operation=current_password_mismatch admin_id=%d", claims.UserID)
			response.Unauthorized(w, err.Error())
		case errors.Is(err, command.ErrAdminNotFound):
			response.NotFound(w, "Admin user not found")
		default:
			log.Printf("ERROR Admin password change failed in HTTP: operation=change_admin_password admin_id=%d error=%v", claims.UserID, err)
			response.InternalServerError(w, "Failed to change password")
		}
		return
	}

	response.NoContent(w)
}
