package postgres

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCriteriaRepositoryFindParticipantIDsByResultCriteria(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT DISTINCT p\.id`).
		WithArgs(uint(77), uint(12)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(3))

	repo := NewCriteriaRepository(db)
	finder, ok := repo.(repository.ResultCriteriaParticipantRepository)
	if !ok {
		t.Fatal("criteria repository does not implement result criteria participant finder")
	}

	participantIDs, err := finder.FindParticipantIDsByResultCriteria(context.Background(), 77, 12)
	if err != nil {
		t.Fatalf("FindParticipantIDsByResultCriteria error: %v", err)
	}
	if len(participantIDs) != 2 {
		t.Fatalf("participant IDs count = %d, want 2", len(participantIDs))
	}
	if _, ok := participantIDs[1]; !ok {
		t.Fatal("participant 1 is missing")
	}
	if _, ok := participantIDs[3]; !ok {
		t.Fatal("participant 3 is missing")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
