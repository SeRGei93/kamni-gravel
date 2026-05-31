package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestCreateAdminHandlerValidatesUsername(t *testing.T) {
	repo := &commandAdminRepoFake{}
	passwords := &commandPasswordServiceFake{hashResult: "hashed-password"}
	h := NewCreateAdminHandler(repo, passwords)

	_, err := h.Handle(context.Background(), CreateAdminCommand{
		Username: "  ",
		Password: "valid-password",
	})
	if !errors.Is(err, ErrAdminUsernameRequired) {
		t.Fatalf("error mismatch: got %v want %v", err, ErrAdminUsernameRequired)
	}
	if passwords.hashCalls != 0 {
		t.Fatalf("hash calls mismatch: got %d want 0", passwords.hashCalls)
	}
}

func TestCreateAdminHandlerMapsDuplicateUsername(t *testing.T) {
	repo := &commandAdminRepoFake{createErr: repository.ErrAdminUsernameTaken}
	h := NewCreateAdminHandler(repo, &commandPasswordServiceFake{hashResult: "hashed-password"})

	_, err := h.Handle(context.Background(), CreateAdminCommand{
		Username: "admin",
		Password: "valid-password",
	})
	if !errors.Is(err, ErrAdminUsernameTaken) {
		t.Fatalf("error mismatch: got %v want %v", err, ErrAdminUsernameTaken)
	}
}

func TestCreateAdminHandlerHashesPasswordAndCreatesAdmin(t *testing.T) {
	repo := &commandAdminRepoFake{createID: 7}
	passwords := &commandPasswordServiceFake{hashResult: "hashed-password"}
	h := NewCreateAdminHandler(repo, passwords)

	admin, err := h.Handle(context.Background(), CreateAdminCommand{
		Username: "  admin  ",
		Password: "valid-password",
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if admin.ID != 7 || admin.Username != "admin" || admin.Role != entity.AdminRoleAdmin {
		t.Fatalf("admin mismatch: got %+v", admin)
	}
	if repo.created == nil {
		t.Fatal("admin was not created")
	}
	if repo.created.PasswordHash != "hashed-password" {
		t.Fatalf("password hash mismatch: got %q", repo.created.PasswordHash)
	}
	if passwords.hashCalls != 1 || passwords.lastHashInput != "valid-password" {
		t.Fatalf("hash dependency was not used as expected")
	}
}

func TestChangeAdminPasswordHandlerRejectsCurrentPasswordMismatch(t *testing.T) {
	repo := &commandAdminRepoFake{
		findByIDAdmin: &entity.Admin{ID: 5, Username: "admin", PasswordHash: "stored-hash"},
	}
	passwords := &commandPasswordServiceFake{compareErr: errors.New("mismatch")}
	h := NewChangeAdminPasswordHandler(repo, passwords)

	err := h.Handle(context.Background(), ChangeAdminPasswordCommand{
		AdminID:         5,
		CurrentPassword: "current-password",
		NewPassword:     "new-password",
	})
	if !errors.Is(err, ErrCurrentPasswordInvalid) {
		t.Fatalf("error mismatch: got %v want %v", err, ErrCurrentPasswordInvalid)
	}
	if repo.updatedPasswordHash != "" {
		t.Fatal("password hash was updated after mismatch")
	}
}

func TestChangeAdminPasswordHandlerUpdatesHash(t *testing.T) {
	repo := &commandAdminRepoFake{
		findByIDAdmin: &entity.Admin{ID: 5, Username: "admin", PasswordHash: "stored-hash"},
	}
	passwords := &commandPasswordServiceFake{hashResult: "new-hash"}
	h := NewChangeAdminPasswordHandler(repo, passwords)

	err := h.Handle(context.Background(), ChangeAdminPasswordCommand{
		AdminID:         5,
		CurrentPassword: "current-password",
		NewPassword:     "new-password",
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if passwords.compareHash != "stored-hash" || passwords.comparePassword != "current-password" {
		t.Fatal("compare dependency was not used as expected")
	}
	if repo.updatedPasswordID != 5 || repo.updatedPasswordHash != "new-hash" {
		t.Fatalf("password update mismatch: id=%d hash=%q", repo.updatedPasswordID, repo.updatedPasswordHash)
	}
}

type commandAdminRepoFake struct {
	admins              []*entity.Admin
	createErr           error
	createID            uint
	created             *entity.Admin
	findByIDAdmin       *entity.Admin
	findByIDErr         error
	updatedPasswordID   uint
	updatedPasswordHash string
	updatePasswordErr   error
}

func (r *commandAdminRepoFake) Create(ctx context.Context, admin *entity.Admin) error {
	if r.createErr != nil {
		return r.createErr
	}
	if r.createID != 0 {
		admin.ID = r.createID
	}
	if admin.CreatedAt.IsZero() {
		admin.CreatedAt = time.Now()
	}
	copied := *admin
	r.created = &copied
	return nil
}

func (r *commandAdminRepoFake) List(ctx context.Context) ([]*entity.Admin, error) {
	return r.admins, nil
}

func (r *commandAdminRepoFake) FindByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	return nil, repository.ErrAdminNotFound
}

func (r *commandAdminRepoFake) FindByID(ctx context.Context, id uint) (*entity.Admin, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	if r.findByIDAdmin == nil {
		return nil, repository.ErrAdminNotFound
	}
	return r.findByIDAdmin, nil
}

func (r *commandAdminRepoFake) UpdateLastLogin(ctx context.Context, id uint) error {
	return nil
}

func (r *commandAdminRepoFake) UpdatePassword(ctx context.Context, id uint, passwordHash string) error {
	if r.updatePasswordErr != nil {
		return r.updatePasswordErr
	}
	r.updatedPasswordID = id
	r.updatedPasswordHash = passwordHash
	return nil
}

func (r *commandAdminRepoFake) Update(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (r *commandAdminRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}

type commandPasswordServiceFake struct {
	hashResult      string
	hashErr         error
	hashCalls       int
	lastHashInput   string
	compareErr      error
	compareHash     string
	comparePassword string
}

func (s *commandPasswordServiceFake) Hash(password string) (string, error) {
	s.hashCalls++
	s.lastHashInput = password
	if s.hashErr != nil {
		return "", s.hashErr
	}
	return s.hashResult, nil
}

func (s *commandPasswordServiceFake) Compare(hash, password string) error {
	s.compareHash = hash
	s.comparePassword = password
	return s.compareErr
}
