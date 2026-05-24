package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type userRepository struct {
	db *sql.DB
}

// NewUserRepository создаёт новый репозиторий пользователей
func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (id, username, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(id) DO UPDATE SET
			username = EXCLUDED.username,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			updated_at = EXCLUDED.updated_at
	`
	
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now
	
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.CreatedAt,
		user.UpdatedAt,
	)
	
	return err
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users
		SET username = $1, first_name = $2, last_name = $3, updated_at = $4
		WHERE id = $5
	`
	
	user.UpdatedAt = time.Now()
	
	result, err := r.db.ExecContext(ctx, query,
		user.Username,
		user.FirstName,
		user.LastName,
		user.UpdatedAt,
		user.ID,
	)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %d", user.ID)
	}
	
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id int64) (*entity.User, error) {
	query := `
		SELECT id, username, first_name, last_name, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	
	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %d", id)
	}
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

func (r *userRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %d", id)
	}
	
	return nil
}

func (r *userRepository) GetAll(ctx context.Context) ([]*entity.User, error) {
	query := `
		SELECT id, username, first_name, last_name, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []*entity.User
	for rows.Next() {
		user := &entity.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.FirstName,
			&user.LastName,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, rows.Err()
}
