package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type chatMemberRepository struct {
	db *sql.DB
}

// NewChatMemberRepository создаёт новый репозиторий ростера публичного чата.
func NewChatMemberRepository(db *sql.DB) repository.ChatMemberRepository {
	return &chatMemberRepository{db: db}
}

const chatMemberUpsertQuery = `
	INSERT INTO chat_members (telegram_user_id, username, first_name, last_name, is_bot, is_admin, joined_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (telegram_user_id) DO UPDATE SET
		username = EXCLUDED.username,
		first_name = EXCLUDED.first_name,
		last_name = EXCLUDED.last_name,
		is_bot = EXCLUDED.is_bot,
		is_admin = EXCLUDED.is_admin,
		updated_at = EXCLUDED.updated_at
`

func (r *chatMemberRepository) Upsert(ctx context.Context, member *entity.ChatMember) error {
	now := time.Now()
	joinedAt := member.JoinedAt
	if joinedAt.IsZero() {
		joinedAt = now
	}

	_, err := r.db.ExecContext(ctx, chatMemberUpsertQuery,
		member.TelegramUserID,
		member.Username,
		member.FirstName,
		member.LastName,
		member.IsBot,
		member.IsAdmin,
		joinedAt,
		now,
	)
	if err != nil {
		log.Printf("Chat member write failed: operation=upsert telegram_user_id=%d error=%v", member.TelegramUserID, err)
		return fmt.Errorf("upsert chat member telegram_user_id=%d: %w", member.TelegramUserID, err)
	}

	member.JoinedAt = joinedAt
	member.UpdatedAt = now
	return nil
}

func (r *chatMemberRepository) BulkUpsert(ctx context.Context, members []*entity.ChatMember) error {
	if len(members) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Chat member write failed: operation=bulk_upsert_begin error=%v", err)
		return fmt.Errorf("begin bulk upsert chat members: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, chatMemberUpsertQuery)
	if err != nil {
		log.Printf("Chat member write failed: operation=bulk_upsert_prepare error=%v", err)
		return fmt.Errorf("prepare bulk upsert chat members: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, member := range members {
		if member == nil {
			continue
		}
		joinedAt := member.JoinedAt
		if joinedAt.IsZero() {
			joinedAt = now
		}
		if _, err := stmt.ExecContext(ctx,
			member.TelegramUserID,
			member.Username,
			member.FirstName,
			member.LastName,
			member.IsBot,
			member.IsAdmin,
			joinedAt,
			now,
		); err != nil {
			log.Printf("Chat member write failed: operation=bulk_upsert telegram_user_id=%d error=%v", member.TelegramUserID, err)
			return fmt.Errorf("bulk upsert chat member telegram_user_id=%d: %w", member.TelegramUserID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Chat member write failed: operation=bulk_upsert_commit error=%v", err)
		return fmt.Errorf("commit bulk upsert chat members: %w", err)
	}

	return nil
}

func (r *chatMemberRepository) Delete(ctx context.Context, telegramUserID int64) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM chat_members WHERE telegram_user_id = $1`, telegramUserID); err != nil {
		log.Printf("Chat member write failed: operation=delete telegram_user_id=%d error=%v", telegramUserID, err)
		return fmt.Errorf("delete chat member telegram_user_id=%d: %w", telegramUserID, err)
	}
	return nil
}

func (r *chatMemberRepository) GetAll(ctx context.Context) ([]*entity.ChatMember, error) {
	query := `
		SELECT telegram_user_id, username, first_name, last_name, is_bot, is_admin, joined_at, updated_at
		FROM chat_members
		ORDER BY first_name, telegram_user_id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list chat members: %w", err)
	}
	defer rows.Close()

	members := make([]*entity.ChatMember, 0)
	for rows.Next() {
		member := &entity.ChatMember{}
		if err := rows.Scan(
			&member.TelegramUserID,
			&member.Username,
			&member.FirstName,
			&member.LastName,
			&member.IsBot,
			&member.IsAdmin,
			&member.JoinedAt,
			&member.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chat member: %w", err)
		}
		members = append(members, member)
	}

	return members, rows.Err()
}

func (r *chatMemberRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chat_members`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count chat members: %w", err)
	}
	return count, nil
}
