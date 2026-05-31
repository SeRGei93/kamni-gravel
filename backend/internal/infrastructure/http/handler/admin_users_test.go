package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gravel_bot/internal/application/command"
	"gravel_bot/internal/application/dto"
	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/pkg/jwt"
)

func TestAdminUsersHandlerGetAllReturnsAdmins(t *testing.T) {
	repo := &adminUsersRepoFake{
		admins: []*entity.Admin{
			{ID: 1, Username: "admin", Role: entity.AdminRoleAdmin, CreatedAt: time.Now()},
		},
	}
	h := newTestAdminUsersHandler(repo, &adminUsersPasswordServiceFake{hashResult: "hashed-password"})

	req := httptest.NewRequest(http.MethodGet, "/api/admin-users", nil)
	rr := httptest.NewRecorder()

	h.GetAll(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var got dto.AdminListResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 1 || got.Admins[0].Username != "admin" {
		t.Fatalf("response mismatch: %+v", got)
	}
}

func TestAdminUsersHandlerCreateReturnsCreatedAdminWithoutSecrets(t *testing.T) {
	repo := &adminUsersRepoFake{createID: 3}
	h := newTestAdminUsersHandler(repo, &adminUsersPasswordServiceFake{hashResult: "hashed-password"})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/admin-users",
		bytes.NewBufferString(`{"username":"new-admin","password":"valid-password"}`),
	)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status mismatch: got %d want %d body=%s", rr.Code, http.StatusCreated, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "valid-password") || strings.Contains(rr.Body.String(), "hashed-password") {
		t.Fatal("response exposed password data")
	}
	var got dto.AdminDTO
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != 3 || got.Username != "new-admin" || got.Role != string(entity.AdminRoleAdmin) {
		t.Fatalf("created admin mismatch: %+v", got)
	}
}

func TestAdminUsersHandlerCreateReturnsConflictForDuplicateUsername(t *testing.T) {
	repo := &adminUsersRepoFake{createErr: repository.ErrAdminUsernameTaken}
	h := newTestAdminUsersHandler(repo, &adminUsersPasswordServiceFake{hashResult: "hashed-password"})

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/admin-users",
		bytes.NewBufferString(`{"username":"admin","password":"valid-password"}`),
	)
	rr := httptest.NewRecorder()

	h.Create(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status mismatch: got %d want %d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
	}
}

func TestAdminUsersHandlerChangePasswordRequiresAuthContext(t *testing.T) {
	h := newTestAdminUsersHandler(&adminUsersRepoFake{}, &adminUsersPasswordServiceFake{hashResult: "new-hash"})

	req := httptest.NewRequest(
		http.MethodPut,
		"/api/auth/me/password",
		bytes.NewBufferString(`{"current_password":"current-password","new_password":"new-password"}`),
	)
	rr := httptest.NewRecorder()

	h.ChangeOwnPassword(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status mismatch: got %d want %d body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

func TestAdminUsersHandlerChangePasswordReturnsUnauthorizedForMismatch(t *testing.T) {
	repo := &adminUsersRepoFake{
		findByIDAdmin: &entity.Admin{ID: 5, Username: "admin", PasswordHash: "stored-hash", Role: entity.AdminRoleAdmin},
	}
	h := newTestAdminUsersHandler(repo, &adminUsersPasswordServiceFake{
		hashResult: "new-hash",
		compareErr: errors.New("mismatch"),
	})

	req := newPasswordChangeRequestWithClaims(5)
	rr := httptest.NewRecorder()

	h.ChangeOwnPassword(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status mismatch: got %d want %d body=%s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
	if repo.updatedPasswordHash != "" {
		t.Fatal("password hash was updated after mismatch")
	}
}

func TestAdminUsersHandlerChangePasswordSucceeds(t *testing.T) {
	repo := &adminUsersRepoFake{
		findByIDAdmin: &entity.Admin{ID: 5, Username: "admin", PasswordHash: "stored-hash", Role: entity.AdminRoleAdmin},
	}
	h := newTestAdminUsersHandler(repo, &adminUsersPasswordServiceFake{hashResult: "new-hash"})

	req := newPasswordChangeRequestWithClaims(5)
	rr := httptest.NewRecorder()

	h.ChangeOwnPassword(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got %d want %d body=%s", rr.Code, http.StatusNoContent, rr.Body.String())
	}
	if repo.updatedPasswordID != 5 || repo.updatedPasswordHash != "new-hash" {
		t.Fatalf("password update mismatch: id=%d hash=%q", repo.updatedPasswordID, repo.updatedPasswordHash)
	}
}

func newPasswordChangeRequestWithClaims(adminID uint) *http.Request {
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/auth/me/password",
		bytes.NewBufferString(`{"current_password":"current-password","new_password":"new-password"}`),
	)
	claims := &jwt.Claims{UserID: adminID, Username: "admin", Role: string(entity.AdminRoleAdmin)}
	return req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, claims))
}

func newTestAdminUsersHandler(repo *adminUsersRepoFake, passwords *adminUsersPasswordServiceFake) *AdminUsersHandler {
	return NewAdminUsersHandler(
		query.NewListAdminUsersHandler(repo),
		command.NewCreateAdminHandler(repo, passwords),
		command.NewChangeAdminPasswordHandler(repo, passwords),
	)
}

type adminUsersRepoFake struct {
	admins              []*entity.Admin
	createErr           error
	createID            uint
	findByIDAdmin       *entity.Admin
	findByIDErr         error
	updatedPasswordID   uint
	updatedPasswordHash string
}

func (r *adminUsersRepoFake) Create(ctx context.Context, admin *entity.Admin) error {
	if r.createErr != nil {
		return r.createErr
	}
	if r.createID != 0 {
		admin.ID = r.createID
	}
	if admin.CreatedAt.IsZero() {
		admin.CreatedAt = time.Now()
	}
	return nil
}

func (r *adminUsersRepoFake) List(ctx context.Context) ([]*entity.Admin, error) {
	return r.admins, nil
}

func (r *adminUsersRepoFake) FindByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	return nil, repository.ErrAdminNotFound
}

func (r *adminUsersRepoFake) FindByID(ctx context.Context, id uint) (*entity.Admin, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	if r.findByIDAdmin == nil {
		return nil, repository.ErrAdminNotFound
	}
	return r.findByIDAdmin, nil
}

func (r *adminUsersRepoFake) UpdateLastLogin(ctx context.Context, id uint) error {
	return nil
}

func (r *adminUsersRepoFake) UpdatePassword(ctx context.Context, id uint, passwordHash string) error {
	r.updatedPasswordID = id
	r.updatedPasswordHash = passwordHash
	return nil
}

func (r *adminUsersRepoFake) Update(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (r *adminUsersRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}

type adminUsersPasswordServiceFake struct {
	hashResult      string
	compareErr      error
	compareHash     string
	comparePassword string
}

func (s *adminUsersPasswordServiceFake) Hash(password string) (string, error) {
	return s.hashResult, nil
}

func (s *adminUsersPasswordServiceFake) Compare(hash, password string) error {
	s.compareHash = hash
	s.comparePassword = password
	return s.compareErr
}
