package postgres

import (
	"context"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestChatMemberRepositoryUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`INSERT INTO chat_members`).
		WithArgs(int64(100), "rider", "Anna", "R", false, true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewChatMemberRepository(db)
	err = repo.Upsert(context.Background(), &entity.ChatMember{
		TelegramUserID: 100,
		Username:       "rider",
		FirstName:      "Anna",
		LastName:       "R",
		IsBot:          false,
		IsAdmin:        true,
	})
	if err != nil {
		t.Fatalf("Upsert error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestChatMemberRepositoryBulkUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectPrepare(`INSERT INTO chat_members`)
	mock.ExpectExec(`INSERT INTO chat_members`).
		WithArgs(int64(1), "a", "A", "", false, false, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO chat_members`).
		WithArgs(int64(2), "b", "B", "", true, false, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewChatMemberRepository(db)
	err = repo.BulkUpsert(context.Background(), []*entity.ChatMember{
		{TelegramUserID: 1, Username: "a", FirstName: "A"},
		{TelegramUserID: 2, Username: "b", FirstName: "B", IsBot: true},
	})
	if err != nil {
		t.Fatalf("BulkUpsert error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestChatMemberRepositoryBulkUpsertEmptyNoOp(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	repo := NewChatMemberRepository(db)
	if err := repo.BulkUpsert(context.Background(), nil); err != nil {
		t.Fatalf("BulkUpsert empty error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestChatMemberRepositoryDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`DELETE FROM chat_members WHERE telegram_user_id = \$1`).
		WithArgs(int64(100)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	repo := NewChatMemberRepository(db)
	if err := repo.Delete(context.Background(), 100); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestChatMemberRepositoryGetAllAndCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{"telegram_user_id", "username", "first_name", "last_name", "is_bot", "is_admin", "joined_at", "updated_at"}).
		AddRow(int64(1), "a", "Anna", "", false, false, time.Now(), time.Now()).
		AddRow(int64(2), "b", "Boris", "", false, true, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT telegram_user_id, username, first_name, last_name, is_bot, is_admin, joined_at, updated_at\s+FROM chat_members`).
		WillReturnRows(rows)

	repo := NewChatMemberRepository(db)
	members, err := repo.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll error: %v", err)
	}
	if len(members) != 2 || members[1].TelegramUserID != 2 || !members[1].IsAdmin {
		t.Fatalf("GetAll mismatch: %+v", members)
	}

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM chat_members`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	count, err := repo.Count(context.Background())
	if err != nil || count != 2 {
		t.Fatalf("Count = %d, err = %v", count, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
