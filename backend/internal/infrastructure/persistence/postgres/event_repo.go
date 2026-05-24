package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type eventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) repository.EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(ctx context.Context, event *entity.Event) error {
	query := `
		INSERT INTO events (name, description, active, start_date, end_date, gpx_file_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	
	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now
	
	err := r.db.QueryRowContext(ctx, query,
		event.Name,
		event.Description,
		event.Active,
		event.StartDate,
		event.EndDate,
		event.GPXFilePath,
		event.CreatedAt,
		event.UpdatedAt,
	).Scan(&event.ID)
	
	return err
}

func (r *eventRepository) Update(ctx context.Context, event *entity.Event) error {
	query := `
		UPDATE events
		SET name = $1, description = $2, active = $3, start_date = $4, end_date = $5, gpx_file_path = $6, updated_at = $7
		WHERE id = $8
	`
	
	event.UpdatedAt = time.Now()
	
	_, err := r.db.ExecContext(ctx, query,
		event.Name,
		event.Description,
		event.Active,
		event.StartDate,
		event.EndDate,
		event.GPXFilePath,
		event.UpdatedAt,
		event.ID,
	)
	
	return err
}

func (r *eventRepository) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	query := `
		SELECT id, name, description, active, start_date, end_date, gpx_file_path, created_at, updated_at
		FROM events
		WHERE id = $1
	`
	
	event := &entity.Event{}
	var gpxFilePath sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.Name,
		&event.Description,
		&event.Active,
		&event.StartDate,
		&event.EndDate,
		&gpxFilePath,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err == nil {
		if gpxFilePath.Valid {
			event.GPXFilePath = gpxFilePath.String
		}
	}
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found: %d", id)
	}
	
	return event, err
}

func (r *eventRepository) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	query := `
		SELECT id, name, description, active, start_date, end_date, gpx_file_path, created_at, updated_at
		FROM events
		WHERE name = $1
	`
	
	event := &entity.Event{}
	var gpxFilePath sql.NullString
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&event.ID,
		&event.Name,
		&event.Description,
		&event.Active,
		&event.StartDate,
		&event.EndDate,
		&gpxFilePath,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err == nil {
		if gpxFilePath.Valid {
			event.GPXFilePath = gpxFilePath.String
		}
	}
	
	if err == sql.ErrNoRows {
		return nil, nil // Событие не найдено - это нормально при проверке существования
	}
	
	return event, err
}

func (r *eventRepository) FindActive(ctx context.Context) (*entity.Event, error) {
	query := `
		SELECT id, name, description, active, start_date, end_date, gpx_file_path, created_at, updated_at
		FROM events
		WHERE active = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	event := &entity.Event{}
	var gpxFilePath sql.NullString
	err := r.db.QueryRowContext(ctx, query).Scan(
		&event.ID,
		&event.Name,
		&event.Description,
		&event.Active,
		&event.StartDate,
		&event.EndDate,
		&gpxFilePath,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err == nil {
		if gpxFilePath.Valid {
			event.GPXFilePath = gpxFilePath.String
		}
	}
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active event found")
	}
	
	return event, err
}

func (r *eventRepository) GetAll(ctx context.Context) ([]*entity.Event, error) {
	query := `
		SELECT id, name, description, active, start_date, end_date, gpx_file_path, created_at, updated_at
		FROM events
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var events []*entity.Event
	for rows.Next() {
		event := &entity.Event{}
		var gpxFilePath sql.NullString
		err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.Description,
			&event.Active,
			&event.StartDate,
			&event.EndDate,
			&gpxFilePath,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if gpxFilePath.Valid {
			event.GPXFilePath = gpxFilePath.String
		}
		events = append(events, event)
	}
	
	return events, rows.Err()
}

func (r *eventRepository) Delete(ctx context.Context, id uint) error {
	query := `DELETE FROM events WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
