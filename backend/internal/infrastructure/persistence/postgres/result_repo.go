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

type resultRepository struct {
	db *sql.DB
}

func NewResultRepository(db *sql.DB) repository.ResultRepository {
	return &resultRepository{db: db}
}

func (r *resultRepository) Create(ctx context.Context, result *entity.Result) error {
	// Сначала помечаем все предыдущие результаты как неактуальные
	_, err := r.db.ExecContext(ctx,
		`UPDATE results SET is_current = false WHERE participant_id = $1`,
		result.ParticipantID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark old results as not current: %w", err)
	}

	query := `
		INSERT INTO results (participant_id, result_link, elapsed_time_sec, moving_time_sec, is_current, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	if result.SubmittedAt.IsZero() {
		result.SubmittedAt = time.Now()
	}

	var resultLink *string
	if result.ResultLink != nil {
		s := result.ResultLink.String()
		resultLink = &s
	}

	err = r.db.QueryRowContext(ctx, query,
		result.ParticipantID,
		resultLink,
		result.ElapsedTimeSec,
		result.MovingTimeSec,
		true, // Новый результат всегда актуальный
		result.SubmittedAt,
	).Scan(&result.ID)

	if err != nil {
		return fmt.Errorf("failed to create result: %w", err)
	}

	result.IsCurrent = true
	return nil
}

func (r *resultRepository) FindByID(ctx context.Context, id uint) (*entity.Result, error) {
	query := `
		SELECT id, participant_id, result_link, elapsed_time_sec, moving_time_sec, is_current, submitted_at
		FROM results
		WHERE id = $1
	`

	return r.scanResult(r.db.QueryRowContext(ctx, query, id))
}

func (r *resultRepository) FindCurrentByParticipant(ctx context.Context, participantID uint) (*entity.Result, error) {
	query := `
		SELECT id, participant_id, result_link, elapsed_time_sec, moving_time_sec, is_current, submitted_at
		FROM results
		WHERE participant_id = $1 AND is_current = true
		ORDER BY submitted_at DESC
		LIMIT 1
	`

	return r.scanResult(r.db.QueryRowContext(ctx, query, participantID))
}

func (r *resultRepository) FindByParticipant(ctx context.Context, participantID uint) ([]*entity.Result, error) {
	query := `
		SELECT id, participant_id, result_link, elapsed_time_sec, moving_time_sec, is_current, submitted_at
		FROM results
		WHERE participant_id = $1
		ORDER BY submitted_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*entity.Result
	for rows.Next() {
		result, err := r.scanResultFromRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

func (r *resultRepository) UpdateTime(ctx context.Context, id uint, elapsedSec, movingSec *int) error {
	query := `UPDATE results SET elapsed_time_sec = $1, moving_time_sec = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, elapsedSec, movingSec, id)
	return err
}

func (r *resultRepository) MarkAsNotCurrent(ctx context.Context, id uint) error {
	query := `UPDATE results SET is_current = false WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *resultRepository) Delete(ctx context.Context, id uint) error {
	query := `DELETE FROM results WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *resultRepository) scanResult(row *sql.Row) (*entity.Result, error) {
	result := &entity.Result{}
	var resultLink *string

	err := row.Scan(
		&result.ID,
		&result.ParticipantID,
		&resultLink,
		&result.ElapsedTimeSec,
		&result.MovingTimeSec,
		&result.IsCurrent,
		&result.SubmittedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if resultLink != nil && *resultLink != "" {
		result.ResultLink, _ = valueobject.NewResultLink(*resultLink)
	}

	return result, nil
}

func (r *resultRepository) scanResultFromRows(rows *sql.Rows) (*entity.Result, error) {
	result := &entity.Result{}
	var resultLink *string

	err := rows.Scan(
		&result.ID,
		&result.ParticipantID,
		&resultLink,
		&result.ElapsedTimeSec,
		&result.MovingTimeSec,
		&result.IsCurrent,
		&result.SubmittedAt,
	)

	if err != nil {
		return nil, err
	}

	if resultLink != nil && *resultLink != "" {
		result.ResultLink, _ = valueobject.NewResultLink(*resultLink)
	}

	return result, nil
}

func (r *resultRepository) AddCriteria(ctx context.Context, resultID, criteriaID uint) error {
	query := `INSERT INTO entity_criteria (entity_type, entity_id, criteria_id) VALUES ('result', $1, $2) ON CONFLICT (entity_type, entity_id, criteria_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, resultID, criteriaID)
	return err
}

func (r *resultRepository) RemoveCriteria(ctx context.Context, resultID, criteriaID uint) error {
	query := `DELETE FROM entity_criteria WHERE entity_type = 'result' AND entity_id = $1 AND criteria_id = $2`
	_, err := r.db.ExecContext(ctx, query, resultID, criteriaID)
	return err
}

func (r *resultRepository) FindWithCriteria(ctx context.Context, resultID uint) (*entity.Result, error) {
	result, err := r.FindByID(ctx, resultID)
	if err != nil || result == nil {
		return result, err
	}

	// Загружаем критерии
	query := `
		SELECT c.id, c.name, c.description, c.criteria_type, c.created_at
		FROM criteria c
		JOIN entity_criteria ec ON ec.criteria_id = c.id
		WHERE ec.entity_type = 'result' AND ec.entity_id = $1
		ORDER BY c.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, resultID)
	if err != nil {
		return nil, fmt.Errorf("failed to load criteria: %w", err)
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
			return nil, fmt.Errorf("failed to scan criteria: %w", err)
		}

		criteriaType, err := valueobject.NewCriteriaType(criteriaTypeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid criteria type: %w", err)
		}
		criteria.CriteriaType = criteriaType

		criteriaList = append(criteriaList, criteria)
	}

	result.Criteria = criteriaList
	return result, rows.Err()
}

func (r *resultRepository) FindByEventWithPlaces(ctx context.Context, eventID uint) ([]*repository.ResultWithPlace, error) {
	query := `
		SELECT 
			r.id, r.participant_id, r.result_link, r.elapsed_time_sec, r.moving_time_sec, r.is_current, r.submitted_at,
			p.gender, p.bike_type,
			ROW_NUMBER() OVER (PARTITION BY p.event_id ORDER BY r.elapsed_time_sec) as place_absolute,
			ROW_NUMBER() OVER (PARTITION BY p.event_id, p.gender ORDER BY r.elapsed_time_sec) as place_by_gender,
			ROW_NUMBER() OVER (PARTITION BY p.event_id, p.gender, p.bike_type ORDER BY r.elapsed_time_sec) as place_by_gender_bike
		FROM results r
		JOIN participants p ON r.participant_id = p.id
		WHERE p.event_id = $1 AND r.is_current = true AND r.elapsed_time_sec IS NOT NULL
		ORDER BY r.elapsed_time_sec
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query results with places: %w", err)
	}
	defer rows.Close()

	var results []*repository.ResultWithPlace
	for rows.Next() {
		result := &entity.Result{}
		var resultLink *string
		var gender, bikeType string
		var placeAbsolute, placeByGender, placeByGenderBike int

		err := rows.Scan(
			&result.ID,
			&result.ParticipantID,
			&resultLink,
			&result.ElapsedTimeSec,
			&result.MovingTimeSec,
			&result.IsCurrent,
			&result.SubmittedAt,
			&gender,
			&bikeType,
			&placeAbsolute,
			&placeByGender,
			&placeByGenderBike,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		if resultLink != nil && *resultLink != "" {
			result.ResultLink, _ = valueobject.NewResultLink(*resultLink)
		}

		results = append(results, &repository.ResultWithPlace{
			Result:              result,
			ParticipantGender:   gender,
			ParticipantBikeType: bikeType,
			PlaceAbsolute:       placeAbsolute,
			PlaceByGender:       placeByGender,
			PlaceByGenderBike:   placeByGenderBike,
		})
	}

	return results, rows.Err()
}
