package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/infrastructure/http/middleware"
	"gravel_bot/internal/pkg/jwt"
)

func TestAuthHandlerMeUsesAuthenticatedUserFromMiddleware(t *testing.T) {
	adminRepo := &authAdminRepoFake{
		admin: &entity.Admin{
			ID:       1,
			Username: "admin",
			Role:     entity.AdminRoleAdmin,
		},
	}
	jwtManager := jwt.NewManager("test-secret", time.Minute, time.Hour)
	tokenPair, err := jwtManager.GenerateTokenPair(1, "admin", string(entity.AdminRoleAdmin))
	if err != nil {
		t.Fatalf("GenerateTokenPair error: %v", err)
	}

	authHandler := NewAuthHandler(adminRepo, jwtManager)
	protectedHandler := middleware.Auth(jwtManager)(
		middleware.RequireRole(jwtManager, string(entity.AdminRoleAdmin))(
			http.HandlerFunc(authHandler.Me),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)
	rr := httptest.NewRecorder()

	protectedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status mismatch: got %d want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	if adminRepo.findByID != 1 {
		t.Fatalf("FindByID id mismatch: got %d want 1", adminRepo.findByID)
	}

	var got UserInfo
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != 1 || got.Username != "admin" || got.Role != string(entity.AdminRoleAdmin) {
		t.Fatalf("user mismatch: got %+v", got)
	}
}

type authAdminRepoFake struct {
	admin    *entity.Admin
	findByID uint
}

func (r *authAdminRepoFake) Create(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (r *authAdminRepoFake) FindByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	return r.admin, nil
}

func (r *authAdminRepoFake) FindByID(ctx context.Context, id uint) (*entity.Admin, error) {
	r.findByID = id
	return r.admin, nil
}

func (r *authAdminRepoFake) UpdateLastLogin(ctx context.Context, id uint) error {
	return nil
}

func (r *authAdminRepoFake) Update(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (r *authAdminRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}
