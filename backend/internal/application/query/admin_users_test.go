package query

import (
	"context"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestListAdminUsersHandlerReturnsAdmins(t *testing.T) {
	now := time.Now()
	repo := &queryAdminRepoFake{
		admins: []*entity.Admin{
			{ID: 2, Username: "second", Role: entity.AdminRoleAdmin, CreatedAt: now},
			{ID: 1, Username: "first", Role: entity.AdminRoleAdmin, CreatedAt: now.Add(-time.Hour)},
		},
	}

	h := NewListAdminUsersHandler(repo)
	admins, err := h.Handle(context.Background())
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if len(admins) != 2 {
		t.Fatalf("admin count mismatch: got %d want 2", len(admins))
	}
	if admins[0].Username != "second" || admins[1].Username != "first" {
		t.Fatalf("admin order mismatch: got %q, %q", admins[0].Username, admins[1].Username)
	}
	if !repo.listCalled {
		t.Fatal("List was not called")
	}
}

type queryAdminRepoFake struct {
	admins     []*entity.Admin
	listCalled bool
}

func (r *queryAdminRepoFake) Create(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (r *queryAdminRepoFake) List(ctx context.Context) ([]*entity.Admin, error) {
	r.listCalled = true
	return r.admins, nil
}

func (r *queryAdminRepoFake) FindByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	return nil, repository.ErrAdminNotFound
}

func (r *queryAdminRepoFake) FindByID(ctx context.Context, id uint) (*entity.Admin, error) {
	return nil, repository.ErrAdminNotFound
}

func (r *queryAdminRepoFake) UpdateLastLogin(ctx context.Context, id uint) error {
	return nil
}

func (r *queryAdminRepoFake) UpdatePassword(ctx context.Context, id uint, passwordHash string) error {
	return nil
}

func (r *queryAdminRepoFake) Update(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (r *queryAdminRepoFake) Delete(ctx context.Context, id uint) error {
	return nil
}
