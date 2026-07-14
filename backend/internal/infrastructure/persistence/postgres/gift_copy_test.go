package postgres

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGiftRepositoryCopyCreatesIndependentCopiesWithCriteriaAndAttachments(t *testing.T) {
	repo, mock := newGiftCopyRepository(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT.*g\.event_id.*g\.place IS NOT NULL OR EXISTS.*FROM gift_place_rules.*FOR UPDATE`).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"event_id", "review_status", "has_place_constraint"}).
			AddRow(77, "approved", false))

	expectGiftCopyInsert(mock, 1, 101)
	expectGiftCopyChildren(mock, 1, 101)
	expectGiftCopyInsert(mock, 1, 102)
	expectGiftCopyChildren(mock, 1, 102)
	mock.ExpectCommit()

	result, err := repo.Copy(context.Background(), 1, 2)

	if err != nil {
		t.Fatalf("Copy() error = %v", err)
	}
	if result.EventID != 77 || result.ReviewStatus != entity.GiftReviewStatusApproved {
		t.Fatalf("Copy() result = %#v, want event 77 and approved status", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGiftRepositoryCopyRollsBackWhenCopyingChildrenFails(t *testing.T) {
	repo, mock := newGiftCopyRepository(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT.*g\.event_id.*g\.place IS NOT NULL OR EXISTS.*FROM gift_place_rules.*FOR UPDATE`).
		WithArgs(uint(1)).
		WillReturnRows(sqlmock.NewRows([]string{"event_id", "review_status", "has_place_constraint"}).
			AddRow(77, "pending_review", false))
	expectGiftCopyInsert(mock, 1, 101)
	mock.ExpectExec(`INSERT INTO entity_criteria`).
		WithArgs(uint(1), uint(101)).
		WillReturnError(errors.New("criteria insert failed"))
	mock.ExpectRollback()

	_, err := repo.Copy(context.Background(), 1, 1)

	if err == nil {
		t.Fatal("Copy() error = nil, want child insert error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGiftRepositoryCopyRejectsEveryPlaceConstraint(t *testing.T) {
	for _, testCase := range []string{"legacy place", "place rule"} {
		t.Run(testCase, func(t *testing.T) {
			repo, mock := newGiftCopyRepository(t)

			mock.ExpectBegin()
			mock.ExpectQuery(`(?s)SELECT.*g\.event_id.*g\.place IS NOT NULL OR EXISTS.*FROM gift_place_rules.*FOR UPDATE`).
				WithArgs(uint(1)).
				WillReturnRows(sqlmock.NewRows([]string{"event_id", "review_status", "has_place_constraint"}).
					AddRow(77, "approved", true))
			mock.ExpectRollback()

			_, err := repo.Copy(context.Background(), 1, 1)

			if !errors.Is(err, repository.ErrGiftCopyHasPlaceConstraint) {
				t.Fatalf("Copy() error = %v, want %v", err, repository.ErrGiftCopyHasPlaceConstraint)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sql expectations: %v", err)
			}
		})
	}
}

func newGiftCopyRepository(t *testing.T) (*giftRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return &giftRepository{db: db}, mock
}

func expectGiftCopyInsert(mock sqlmock.Sqlmock, sourceGiftID uint, copyID uint) {
	mock.ExpectQuery(`(?s)INSERT INTO gifts.*review_status, NULL, manual_distribution,\s*NULL, NOW\(\)`).
		WithArgs(sourceGiftID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(copyID))
}

func expectGiftCopyChildren(mock sqlmock.Sqlmock, sourceGiftID uint, copyID uint) {
	mock.ExpectExec(`INSERT INTO entity_criteria`).
		WithArgs(sourceGiftID, copyID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO gift_attachments`).
		WithArgs(sourceGiftID, copyID).
		WillReturnResult(sqlmock.NewResult(0, 1))
}
