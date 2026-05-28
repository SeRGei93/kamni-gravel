package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/valueobject"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGiftRepositoryFindByIDLoadsNoPlaceRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT g\.id, g\.user_id, g\.event_id, g\.description`).
		WithArgs(uint(1)).
		WillReturnRows(giftRows().AddRow(1, int64(100), 77, "gift", "all", "all", "approved", nil, time.Now(), "rider", "", ""))
	mock.ExpectQuery(`FROM gift_place_rules r`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(giftPlaceRuleRows())

	repo := NewGiftRepository(db)
	gift, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}

	if !gift.PlaceRule.IsNone() {
		t.Fatalf("place rule = %s, want none", gift.PlaceRule.Type())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGiftRepositoryFindByIDLoadsPlacesRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT g\.id, g\.user_id, g\.event_id, g\.description`).
		WithArgs(uint(1)).
		WillReturnRows(giftRows().AddRow(1, int64(100), 77, "gift", "all", "all", "approved", 10, time.Now(), "rider", "", ""))
	mock.ExpectQuery(`FROM gift_place_rules r`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(giftPlaceRuleRows().
			AddRow(1, "places", nil, 10).
			AddRow(1, "places", nil, 15))

	repo := NewGiftRepository(db)
	gift, err := repo.FindByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}

	assertGiftRepoPlaces(t, gift.PlaceRule.Places(), []int{10, 15})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGiftRepositoryFindByEventAndReviewStatusLoadsLastNRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT g\.id, g\.user_id, g\.event_id, g\.description`).
		WithArgs(uint(77), entity.GiftReviewStatusApproved.String()).
		WillReturnRows(giftRows().AddRow(1, int64(100), 77, "gift", "female", "gravel", "approved", nil, time.Now(), "rider", "", ""))
	mock.ExpectQuery(`FROM gift_place_rules r`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(giftPlaceRuleRows().AddRow(1, "last_n", 5, nil))

	repo := NewGiftRepository(db)
	gifts, err := repo.FindByEventAndReviewStatus(context.Background(), 77, entity.GiftReviewStatusApproved)
	if err != nil {
		t.Fatalf("FindByEventAndReviewStatus error: %v", err)
	}

	if len(gifts) != 1 {
		t.Fatalf("gifts count = %d, want 1", len(gifts))
	}
	if gifts[0].PlaceRule.Type() != valueobject.GiftPlaceRuleTypeLastN || gifts[0].PlaceRule.LastCount() != 5 {
		t.Fatalf("place rule = %s/%d, want last_n/5", gifts[0].PlaceRule.Type(), gifts[0].PlaceRule.LastCount())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGiftRepositoryUpdateWithCriteriaClearsPlaceRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	gift := giftRepoUpdateGift(t)
	gift.PlaceRule = valueobject.NewGiftPlaceRuleNone()

	mock.ExpectBegin()
	expectGiftFieldsUpdate(mock)
	mock.ExpectExec(`DELETE FROM entity_criteria`).
		WithArgs(gift.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM gift_place_rules`).
		WithArgs(gift.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewGiftRepository(db)
	if err := repo.UpdateWithCriteria(context.Background(), gift, nil); err != nil {
		t.Fatalf("UpdateWithCriteria error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGiftRepositoryUpdateWithCriteriaReplacesCriteriaAndPlacesRule(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	gift := giftRepoUpdateGift(t)
	gift.PlaceRule = mustGiftRepoPlacesRule(t, []int{10, 11})

	mock.ExpectBegin()
	expectGiftFieldsUpdate(mock)
	mock.ExpectExec(`DELETE FROM entity_criteria`).
		WithArgs(gift.ID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`INSERT INTO entity_criteria`).
		WithArgs(gift.ID, uint(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO entity_criteria`).
		WithArgs(gift.ID, uint(8)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM gift_place_rules`).
		WithArgs(gift.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO gift_place_rules`).
		WithArgs(gift.ID, "places", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO gift_place_rule_places`).
		WithArgs(gift.ID, 10).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO gift_place_rule_places`).
		WithArgs(gift.ID, 11).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := NewGiftRepository(db)
	if err := repo.UpdateWithCriteria(context.Background(), gift, []uint{7, 8}); err != nil {
		t.Fatalf("UpdateWithCriteria error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGiftRepositoryUpdateWithCriteriaRollsBackOnRuleInsertFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	gift := giftRepoUpdateGift(t)
	gift.PlaceRule = mustGiftRepoPlacesRule(t, []int{10})
	insertErr := errors.New("insert rule failed")

	mock.ExpectBegin()
	expectGiftFieldsUpdate(mock)
	mock.ExpectExec(`DELETE FROM entity_criteria`).
		WithArgs(gift.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM gift_place_rules`).
		WithArgs(gift.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO gift_place_rules`).
		WithArgs(gift.ID, "places", nil).
		WillReturnError(insertErr)
	mock.ExpectRollback()

	repo := NewGiftRepository(db)
	if err := repo.UpdateWithCriteria(context.Background(), gift, nil); err == nil {
		t.Fatal("UpdateWithCriteria error = nil, want error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func giftRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"user_id",
		"event_id",
		"description",
		"gender_filter",
		"bike_type_filter",
		"review_status",
		"place",
		"created_at",
		"username",
		"first_name",
		"last_name",
	})
}

func giftPlaceRuleRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"gift_id", "rule_type", "last_count", "place"})
}

func giftRepoUpdateGift(t *testing.T) *entity.Gift {
	t.Helper()

	return &entity.Gift{
		ID:             1,
		UserID:         100,
		EventID:        77,
		Description:    "gift",
		GenderFilter:   "all",
		BikeTypeFilter: "all",
		ReviewStatus:   entity.GiftReviewStatusApproved,
	}
}

func expectGiftFieldsUpdate(mock sqlmock.Sqlmock) {
	mock.ExpectExec(`UPDATE gifts SET description`).
		WithArgs("gift", "all", "all", entity.GiftReviewStatusApproved.String(), sqlmock.AnyArg(), uint(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func mustGiftRepoPlacesRule(t *testing.T, places []int) valueobject.GiftPlaceRule {
	t.Helper()

	rule, err := valueobject.NewGiftPlaceRulePlaces(places)
	if err != nil {
		t.Fatalf("NewGiftPlaceRulePlaces error: %v", err)
	}
	return rule
}

func assertGiftRepoPlaces(t *testing.T, got []int, want []int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("places = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("places = %v, want %v", got, want)
		}
	}
}
