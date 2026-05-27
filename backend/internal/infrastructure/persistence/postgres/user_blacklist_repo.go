package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type userBlacklistRepository struct {
	db *sql.DB
}

type scanExecutor interface {
	Scan(dest ...any) error
}

// NewUserBlacklistRepository создаёт новый репозиторий blacklist пользователей.
func NewUserBlacklistRepository(db *sql.DB) repository.UserBlacklistRepository {
	return &userBlacklistRepository{db: db}
}

func (r *userBlacklistRepository) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	query := `
		SELECT b.telegram_user_id, b.reason, b.created_at, b.updated_at,
		       u.id, u.username, u.first_name, u.last_name, u.created_at, u.updated_at
		FROM user_blacklist b
		LEFT JOIN users u ON u.id = b.telegram_user_id
		ORDER BY b.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list user blacklist: %w", err)
	}
	defer rows.Close()

	entries := make([]*entity.UserBlacklist, 0)
	for rows.Next() {
		entry, err := scanUserBlacklist(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *userBlacklistRepository) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	query := `
		SELECT b.telegram_user_id, b.reason, b.created_at, b.updated_at,
		       u.id, u.username, u.first_name, u.last_name, u.created_at, u.updated_at
		FROM user_blacklist b
		LEFT JOIN users u ON u.id = b.telegram_user_id
		WHERE b.telegram_user_id = $1
	`

	entry, err := scanUserBlacklist(r.db.QueryRowContext(ctx, query, telegramUserID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", repository.ErrUserBlacklistEntryNotFound, telegramUserID)
	}
	if err != nil {
		return nil, err
	}

	return entry, nil
}

func (r *userBlacklistRepository) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_blacklist WHERE telegram_user_id = $1)`

	var isBlacklisted bool
	if err := r.db.QueryRowContext(ctx, query, telegramUserID).Scan(&isBlacklisted); err != nil {
		return false, fmt.Errorf("check user blacklist telegram_user_id=%d: %w", telegramUserID, err)
	}

	return isBlacklisted, nil
}

func (r *userBlacklistRepository) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	query := `
		INSERT INTO user_blacklist (telegram_user_id, reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (telegram_user_id) DO UPDATE SET
			reason = EXCLUDED.reason,
			updated_at = EXCLUDED.updated_at
		RETURNING created_at, updated_at
	`

	now := time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query,
		entry.TelegramUserID,
		entry.Reason,
		entry.CreatedAt,
		entry.UpdatedAt,
	).Scan(&entry.CreatedAt, &entry.UpdatedAt)
	if err != nil {
		log.Printf("User blacklist write failed: operation=upsert telegram_user_id=%d error=%v", entry.TelegramUserID, err)
		return fmt.Errorf("upsert user blacklist telegram_user_id=%d: %w", entry.TelegramUserID, err)
	}

	return nil
}

func (r *userBlacklistRepository) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	query := `
		UPDATE user_blacklist
		SET reason = $2, updated_at = $3
		WHERE telegram_user_id = $1
		RETURNING telegram_user_id, reason, created_at, updated_at
	`

	now := time.Now()
	entry := &entity.UserBlacklist{}
	err := r.db.QueryRowContext(ctx, query, telegramUserID, reason, now).Scan(
		&entry.TelegramUserID,
		&entry.Reason,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", repository.ErrUserBlacklistEntryNotFound, telegramUserID)
	}
	if err != nil {
		log.Printf("User blacklist write failed: operation=update_reason telegram_user_id=%d error=%v", telegramUserID, err)
		return nil, fmt.Errorf("update user blacklist reason telegram_user_id=%d: %w", telegramUserID, err)
	}

	return entry, nil
}

func (r *userBlacklistRepository) Delete(ctx context.Context, telegramUserID int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM user_blacklist WHERE telegram_user_id = $1`, telegramUserID)
	if err != nil {
		log.Printf("User blacklist write failed: operation=delete telegram_user_id=%d error=%v", telegramUserID, err)
		return fmt.Errorf("delete user blacklist telegram_user_id=%d: %w", telegramUserID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("User blacklist write failed: operation=delete_rows_affected telegram_user_id=%d error=%v", telegramUserID, err)
		return fmt.Errorf("delete user blacklist rows affected telegram_user_id=%d: %w", telegramUserID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: %d", repository.ErrUserBlacklistEntryNotFound, telegramUserID)
	}

	return nil
}

func scanUserBlacklist(scanner scanExecutor) (*entity.UserBlacklist, error) {
	entry := &entity.UserBlacklist{}
	var userID sql.NullInt64
	var username sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var userCreatedAt sql.NullTime
	var userUpdatedAt sql.NullTime

	err := scanner.Scan(
		&entry.TelegramUserID,
		&entry.Reason,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&userID,
		&username,
		&firstName,
		&lastName,
		&userCreatedAt,
		&userUpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if userID.Valid {
		entry.User = &entity.User{
			ID:        userID.Int64,
			Username:  username.String,
			FirstName: firstName.String,
			LastName:  lastName.String,
		}
		if userCreatedAt.Valid {
			entry.User.CreatedAt = userCreatedAt.Time
		}
		if userUpdatedAt.Valid {
			entry.User.UpdatedAt = userUpdatedAt.Time
		}
	}

	return entry, nil
}
