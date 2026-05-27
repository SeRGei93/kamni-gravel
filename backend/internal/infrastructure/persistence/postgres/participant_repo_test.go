package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestParticipantRepositoryDeleteWithResultCriteriaCleansResultCriteriaBeforeParticipant(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	repo := NewParticipantRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM results WHERE participant_id = \$1`).
		WithArgs(int64(55)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)).AddRow(int64(11)))
	mock.ExpectExec(`DELETE FROM entity_criteria`).
		WithArgs(int64(55)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM participants WHERE id = \$1`).
		WithArgs(int64(55)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.DeleteWithResultCriteria(context.Background(), 55); err != nil {
		t.Fatalf("DeleteWithResultCriteria error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
