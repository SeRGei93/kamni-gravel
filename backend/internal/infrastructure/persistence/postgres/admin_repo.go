package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

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
		return mapAdminWriteError(err)
	}

	return nil
}

func (r *adminRepository) List(ctx context.Context) ([]*entity.Admin, error) {
	query := `
		SELECT id, username, password_hash, role, created_at, last_login
		FROM admin_users
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	admins := make([]*entity.Admin, 0)
	for rows.Next() {
		admin, err := scanAdmin(rows)
		if err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}

	return admins, rows.Err()
}

func (r *adminRepository) FindByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login FROM admin_users WHERE username = $1`

	admin, err := scanAdmin(r.db.QueryRowContext(ctx, query, username))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: username=%s", repository.ErrAdminNotFound, username)
	}
	if err != nil {
		return nil, err
	}

	return admin, nil
}

func (r *adminRepository) FindByID(ctx context.Context, id uint) (*entity.Admin, error) {
	query := `SELECT id, username, password_hash, role, created_at, last_login FROM admin_users WHERE id = $1`

	admin, err := scanAdmin(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: id=%d", repository.ErrAdminNotFound, id)
	}
	if err != nil {
		return nil, err
	}

	return admin, nil
}

func (r *adminRepository) UpdateLastLogin(ctx context.Context, id uint) error {
	query := `UPDATE admin_users SET last_login = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *adminRepository) UpdatePassword(ctx context.Context, id uint, passwordHash string) error {
	query := `UPDATE admin_users SET password_hash = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return mapAdminWriteError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check admin password update result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: id=%d", repository.ErrAdminNotFound, id)
	}

	return nil
}

func (r *adminRepository) Update(ctx context.Context, admin *entity.Admin) error {
	query := `UPDATE admin_users SET username = $1, password_hash = $2, role = $3 WHERE id = $4`
	result, err := r.db.ExecContext(ctx, query, admin.Username, admin.PasswordHash, admin.Role, admin.ID)
	if err != nil {
		return mapAdminWriteError(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check admin update result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: id=%d", repository.ErrAdminNotFound, admin.ID)
	}

	return nil
}

func (r *adminRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admin_users WHERE id = $1`, id)
	return err
}

type adminScanner interface {
	Scan(dest ...interface{}) error
}

func scanAdmin(row adminScanner) (*entity.Admin, error) {
	admin := &entity.Admin{}
	var role string

	err := row.Scan(&admin.ID, &admin.Username, &admin.PasswordHash, &role, &admin.CreatedAt, &admin.LastLogin)
	if err != nil {
		return nil, err
	}

	admin.Role = entity.AdminRole(role)
	return admin, nil
}

func mapAdminWriteError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
		return repository.ErrAdminUsernameTaken
	}
	return err
}
