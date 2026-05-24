package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

type criteriaRepository struct {
	db *sql.DB
}

func NewCriteriaRepository(db *sql.DB) repository.CriteriaRepository {
	return &criteriaRepository{db: db}
}

func (r *criteriaRepository) Create(ctx context.Context, criteria *entity.Criteria) error {
	query := `INSERT INTO criteria (name, description, criteria_type, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
	
	if criteria.CreatedAt.IsZero() {
		criteria.CreatedAt = time.Now()
	}
	
	err := r.db.QueryRowContext(ctx, query, criteria.Name, criteria.Description, criteria.CriteriaType.String(), criteria.CreatedAt).Scan(&criteria.ID)
	return err
}

func (r *criteriaRepository) Update(ctx context.Context, criteria *entity.Criteria) error {
	query := `UPDATE criteria SET name = $1, description = $2, criteria_type = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, criteria.Name, criteria.Description, criteria.CriteriaType.String(), criteria.ID)
	return err
}

func (r *criteriaRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM criteria WHERE id = $1`, id)
	return err
}

func (r *criteriaRepository) FindByID(ctx context.Context, id uint) (*entity.Criteria, error) {
	query := `SELECT id, name, description, criteria_type, created_at FROM criteria WHERE id = $1`
	
	criteria := &entity.Criteria{}
	var criteriaTypeStr string
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&criteria.ID,
		&criteria.Name,
		&criteria.Description,
		&criteriaTypeStr,
		&criteria.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("criteria not found: %d", id)
	}
	if err != nil {
		return nil, err
	}
	
	criteriaType, err := valueobject.NewCriteriaType(criteriaTypeStr)
	if err != nil {
		return nil, err
	}
	criteria.CriteriaType = criteriaType
	
	return criteria, nil
}

func (r *criteriaRepository) FindAll(ctx context.Context) ([]*entity.Criteria, error) {
	query := `SELECT id, name, description, criteria_type, created_at FROM criteria ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var criteriaList []*entity.Criteria
	for rows.Next() {
		criteria := &entity.Criteria{}
		var criteriaTypeStr string
		
		err := rows.Scan(
			&criteria.ID,
			&criteria.Name,
			&criteria.Description,
			&criteriaTypeStr,
			&criteria.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		criteriaType, err := valueobject.NewCriteriaType(criteriaTypeStr)
		if err != nil {
			return nil, err
		}
		criteria.CriteriaType = criteriaType

		criteriaList = append(criteriaList, criteria)
	}

	return criteriaList, rows.Err()
}

func (r *criteriaRepository) FindByResult(ctx context.Context, resultID uint) ([]*entity.Criteria, error) {
	query := `
		SELECT c.id, c.name, c.description, c.criteria_type, c.created_at
		FROM criteria c
		JOIN entity_criteria ec ON ec.criteria_id = c.id
		WHERE ec.entity_type = 'result' AND ec.entity_id = $1
		ORDER BY c.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, resultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var criteriaList []*entity.Criteria
	for rows.Next() {
		criteria := &entity.Criteria{}
		var criteriaTypeStr string

		err := rows.Scan(
			&criteria.ID,
			&criteria.Name,
			&criteria.Description,
			&criteriaTypeStr,
			&criteria.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		criteriaType, err := valueobject.NewCriteriaType(criteriaTypeStr)
		if err != nil {
			return nil, err
		}
		criteria.CriteriaType = criteriaType

		criteriaList = append(criteriaList, criteria)
	}

	return criteriaList, rows.Err()
}

func (r *criteriaRepository) FindByType(ctx context.Context, criteriaType valueobject.CriteriaType) ([]*entity.Criteria, error) {
	query := `SELECT id, name, description, criteria_type, created_at FROM criteria WHERE criteria_type = $1 ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query, criteriaType.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var criteriaList []*entity.Criteria
	for rows.Next() {
		criteria := &entity.Criteria{}
		var criteriaTypeStr string
		
		err := rows.Scan(
			&criteria.ID,
			&criteria.Name,
			&criteria.Description,
			&criteriaTypeStr,
			&criteria.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		ct, err := valueobject.NewCriteriaType(criteriaTypeStr)
		if err != nil {
			return nil, err
		}
		criteria.CriteriaType = ct
		
		criteriaList = append(criteriaList, criteria)
	}
	
	return criteriaList, rows.Err()
}

func (r *criteriaRepository) FindByGift(ctx context.Context, giftID uint) ([]*entity.Criteria, error) {
	query := `
		SELECT c.id, c.name, c.description, c.criteria_type, c.created_at
		FROM criteria c
		JOIN entity_criteria ec ON ec.criteria_id = c.id
		WHERE ec.entity_type = 'gift' AND ec.entity_id = $1
		ORDER BY c.created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query, giftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var criteriaList []*entity.Criteria
	for rows.Next() {
		criteria := &entity.Criteria{}
		var criteriaTypeStr string
		
		err := rows.Scan(
			&criteria.ID,
			&criteria.Name,
			&criteria.Description,
			&criteriaTypeStr,
			&criteria.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		criteriaType, err := valueobject.NewCriteriaType(criteriaTypeStr)
		if err != nil {
			return nil, err
		}
		criteria.CriteriaType = criteriaType

		criteriaList = append(criteriaList, criteria)
	}

	return criteriaList, rows.Err()
}
