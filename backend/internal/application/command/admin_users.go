package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

const MinAdminPasswordLength = 8

var (
	ErrAdminUsernameRequired  = errors.New("admin username is required")
	ErrAdminPasswordRequired  = errors.New("admin password is required")
	ErrAdminPasswordTooShort  = errors.New("admin password is too short")
	ErrAdminUsernameTaken     = errors.New("admin username already taken")
	ErrAdminNotFound          = errors.New("admin not found")
	ErrCurrentPasswordInvalid = errors.New("current password is invalid")
)

// PasswordService описывает хэширование и проверку паролей для admin commands.
type PasswordService interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// CreateAdminCommand представляет команду создания администратора.
type CreateAdminCommand struct {
	Username string
	Password string
}

// CreateAdminHandler обрабатывает создание администратора.
type CreateAdminHandler struct {
	adminRepo       repository.AdminRepository
	passwordService PasswordService
}

// NewCreateAdminHandler создаёт новый handler.
func NewCreateAdminHandler(
	adminRepo repository.AdminRepository,
	passwordService PasswordService,
) *CreateAdminHandler {
	return &CreateAdminHandler{
		adminRepo:       adminRepo,
		passwordService: passwordService,
	}
}

// Handle выполняет команду создания администратора.
func (h *CreateAdminHandler) Handle(ctx context.Context, cmd CreateAdminCommand) (*entity.Admin, error) {
	username := strings.TrimSpace(cmd.Username)
	if err := validateAdminUsername(username, "create_admin"); err != nil {
		return nil, err
	}
	if err := validateAdminPassword(cmd.Password, "create_admin", username); err != nil {
		return nil, err
	}

	passwordHash, err := h.passwordService.Hash(cmd.Password)
	if err != nil {
		log.Printf("ERROR Admin create failed: operation=hash_password username=%q error=%v", username, err)
		return nil, fmt.Errorf("hash admin password: %w", err)
	}

	admin := &entity.Admin{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         entity.AdminRoleAdmin,
		CreatedAt:    time.Now(),
	}

	if err := h.adminRepo.Create(ctx, admin); err != nil {
		if errors.Is(err, repository.ErrAdminUsernameTaken) {
			log.Printf("WARN Admin create rejected: operation=create_admin username=%q reason=username_taken", username)
			return nil, ErrAdminUsernameTaken
		}
		log.Printf("ERROR Admin create failed: operation=create_admin username=%q error=%v", username, err)
		return nil, fmt.Errorf("create admin: %w", err)
	}

	return admin, nil
}

// ChangeAdminPasswordCommand представляет команду смены пароля текущего администратора.
type ChangeAdminPasswordCommand struct {
	AdminID         uint
	CurrentPassword string
	NewPassword     string
}

// ChangeAdminPasswordHandler обрабатывает смену пароля текущего администратора.
type ChangeAdminPasswordHandler struct {
	adminRepo       repository.AdminRepository
	passwordService PasswordService
}

// NewChangeAdminPasswordHandler создаёт новый handler.
func NewChangeAdminPasswordHandler(
	adminRepo repository.AdminRepository,
	passwordService PasswordService,
) *ChangeAdminPasswordHandler {
	return &ChangeAdminPasswordHandler{
		adminRepo:       adminRepo,
		passwordService: passwordService,
	}
}

// Handle выполняет команду смены пароля текущего администратора.
func (h *ChangeAdminPasswordHandler) Handle(ctx context.Context, cmd ChangeAdminPasswordCommand) error {
	if cmd.AdminID == 0 {
		log.Printf("WARN Admin password change rejected: operation=change_admin_password admin_id=%d reason=missing_admin_id", cmd.AdminID)
		return ErrAdminNotFound
	}
	if strings.TrimSpace(cmd.CurrentPassword) == "" {
		log.Printf("WARN Admin password change rejected: operation=change_admin_password admin_id=%d reason=current_password_required", cmd.AdminID)
		return ErrAdminPasswordRequired
	}
	if err := validateAdminPassword(cmd.NewPassword, "change_admin_password", ""); err != nil {
		return err
	}

	admin, err := h.adminRepo.FindByID(ctx, cmd.AdminID)
	if errors.Is(err, repository.ErrAdminNotFound) {
		log.Printf("WARN Admin password change rejected: operation=change_admin_password admin_id=%d reason=admin_not_found", cmd.AdminID)
		return ErrAdminNotFound
	}
	if err != nil {
		log.Printf("ERROR Admin password change failed: operation=find_admin admin_id=%d error=%v", cmd.AdminID, err)
		return fmt.Errorf("find admin: %w", err)
	}

	if err := h.passwordService.Compare(admin.PasswordHash, cmd.CurrentPassword); err != nil {
		log.Printf("WARN Admin password change rejected: operation=change_admin_password admin_id=%d reason=current_password_mismatch", cmd.AdminID)
		return ErrCurrentPasswordInvalid
	}

	passwordHash, err := h.passwordService.Hash(cmd.NewPassword)
	if err != nil {
		log.Printf("ERROR Admin password change failed: operation=hash_password admin_id=%d error=%v", cmd.AdminID, err)
		return fmt.Errorf("hash admin password: %w", err)
	}

	if err := h.adminRepo.UpdatePassword(ctx, cmd.AdminID, passwordHash); err != nil {
		if errors.Is(err, repository.ErrAdminNotFound) {
			log.Printf("WARN Admin password change rejected: operation=update_password admin_id=%d reason=admin_not_found", cmd.AdminID)
			return ErrAdminNotFound
		}
		log.Printf("ERROR Admin password change failed: operation=update_password admin_id=%d error=%v", cmd.AdminID, err)
		return fmt.Errorf("update admin password: %w", err)
	}

	return nil
}

func validateAdminUsername(username, operation string) error {
	if username == "" {
		log.Printf("WARN Admin validation failed: operation=%s reason=username_required", operation)
		return ErrAdminUsernameRequired
	}
	return nil
}

func validateAdminPassword(password, operation, username string) error {
	if strings.TrimSpace(password) == "" {
		log.Printf("WARN Admin validation failed: operation=%s username=%q reason=password_required", operation, username)
		return ErrAdminPasswordRequired
	}
	if utf8.RuneCountInString(password) < MinAdminPasswordLength {
		log.Printf("WARN Admin validation failed: operation=%s username=%q reason=password_too_short", operation, username)
		return ErrAdminPasswordTooShort
	}
	return nil
}
