package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type adminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) repository.AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) Create(ctx context.Context, admin *entity.Admin) error {
	query := `INSERT INTO admin_users (username, password_hash, role, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
	
	if admin.CreatedAt.IsZero() {
		admin.CreatedAt = time.Now()
	}
	
	err := r.db.QueryRowContext(ctx, query, admin.Username, admin.PasswordHash, admin.Role, admin.CreatedAt).Scan(&admin.ID)
	if err != nil {
		return err
	}
	
	return nil
}

func (r *adminRepository) FindByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login FROM admin_users WHERE username = $1`
	
	admin := &entity.Admin{}
	var role string
	
	err := r.db.QueryRowContext(ctx, query, username).Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &role, &admin.CreatedAt, &admin.LastLogin)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("admin not found: %s", username)
	}
	if err != nil {
		return nil, err
	}
	
	admin.Role = entity.AdminRole(role)
	
	return admin, nil
}

func (r *adminRepository) FindByID(ctx context.Context, id uint) (*entity.Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login FROM admin_users WHERE id = $1`
	
	admin := &entity.Admin{}
	var role string
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &role, &admin.CreatedAt, &admin.LastLogin)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("admin not found: %d", id)
	}
	if err != nil {
		return nil, err
	}
	
	admin.Role = entity.AdminRole(role)
	
	return admin, nil
}

func (r *adminRepository) UpdateLastLogin(ctx context.Context, id uint) error {
	query := `UPDATE admin_users SET last_login = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *adminRepository) Update(ctx context.Context, admin *entity.Admin) error {
	query := `UPDATE admin_users SET username = $1, password_hash = $2, role = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, admin.Username, admin.PasswordHash, admin.Role, admin.ID)
	return err
}

func (r *adminRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admin_users WHERE id = $1`, id)
	return err
}
