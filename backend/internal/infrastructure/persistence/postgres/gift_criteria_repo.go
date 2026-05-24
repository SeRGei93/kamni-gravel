package postgres

import (
	"context"
	"database/sql"
	
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type giftCriteriaRepository struct {
	db *sql.DB
}

func NewGiftCriteriaRepository(db *sql.DB) repository.GiftCriteriaRepository {
	return &giftCriteriaRepository{db: db}
}

func (r *giftCriteriaRepository) AddCriteriaToGift(ctx context.Context, giftID uint, criteriaID uint) error {
	query := `INSERT INTO entity_criteria (entity_type, entity_id, criteria_id) VALUES ('gift', $1, $2) ON CONFLICT (entity_type, entity_id, criteria_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, giftID, criteriaID)
	return err
}

func (r *giftCriteriaRepository) RemoveCriteriaFromGift(ctx context.Context, giftID uint, criteriaID uint) error {
	query := `DELETE FROM entity_criteria WHERE entity_type = 'gift' AND entity_id = $1 AND criteria_id = $2`
	_, err := r.db.ExecContext(ctx, query, giftID, criteriaID)
	return err
}

func (r *giftCriteriaRepository) RemoveAllCriteriaFromGift(ctx context.Context, giftID uint) error {
	query := `DELETE FROM entity_criteria WHERE entity_type = 'gift' AND entity_id = $1`
	_, err := r.db.ExecContext(ctx, query, giftID)
	return err
}

func (r *giftCriteriaRepository) FindByGift(ctx context.Context, giftID uint) ([]*entity.GiftCriteria, error) {
	query := `SELECT id, entity_id, criteria_id FROM entity_criteria WHERE entity_type = 'gift' AND entity_id = $1`
	
	rows, err := r.db.QueryContext(ctx, query, giftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var giftCriteriaList []*entity.GiftCriteria
	for rows.Next() {
		gc := &entity.GiftCriteria{}
		err := rows.Scan(&gc.ID, &gc.GiftID, &gc.CriteriaID)
		if err != nil {
			return nil, err
		}
		giftCriteriaList = append(giftCriteriaList, gc)
	}
	
	return giftCriteriaList, rows.Err()
}

func (r *giftCriteriaRepository) FindByCriteria(ctx context.Context, criteriaID uint) ([]*entity.GiftCriteria, error) {
	query := `SELECT id, entity_id, criteria_id FROM entity_criteria WHERE entity_type = 'gift' AND criteria_id = $1`
	
	rows, err := r.db.QueryContext(ctx, query, criteriaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var giftCriteriaList []*entity.GiftCriteria
	for rows.Next() {
		gc := &entity.GiftCriteria{}
		err := rows.Scan(&gc.ID, &gc.GiftID, &gc.CriteriaID)
		if err != nil {
			return nil, err
		}
		giftCriteriaList = append(giftCriteriaList, gc)
	}
	
	return giftCriteriaList, rows.Err()
}
