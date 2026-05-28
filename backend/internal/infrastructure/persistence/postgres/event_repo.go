package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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
		INSERT INTO events (name, description, participation_conditions, active, start_date, end_date, gpx_file_path, telegram_texts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	telegramTextsJSON, err := eventTelegramTextsJSON(event.TelegramTexts)
	if err != nil {
		return err
	}

	now := time.Now()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.UpdatedAt = now

	err = r.db.QueryRowContext(ctx, query,
		event.Name,
		event.Description,
		entity.NormalizeEventParticipationConditions(event.ParticipationConditions),
		event.Active,
		event.StartDate,
		event.EndDate,
		event.GPXFilePath,
		telegramTextsJSON,
		event.CreatedAt,
		event.UpdatedAt,
	).Scan(&event.ID)

	return err
}

func (r *eventRepository) Update(ctx context.Context, event *entity.Event) error {
	query := `
		UPDATE events
		SET name = $1, description = $2, participation_conditions = $3, active = $4, start_date = $5, end_date = $6, gpx_file_path = $7, telegram_texts = $8, updated_at = $9
		WHERE id = $10
	`

	telegramTextsJSON, err := eventTelegramTextsJSON(event.TelegramTexts)
	if err != nil {
		return err
	}

	event.UpdatedAt = time.Now()

	_, err = r.db.ExecContext(ctx, query,
		event.Name,
		event.Description,
		entity.NormalizeEventParticipationConditions(event.ParticipationConditions),
		event.Active,
		event.StartDate,
		event.EndDate,
		event.GPXFilePath,
		telegramTextsJSON,
		event.UpdatedAt,
		event.ID,
	)

	return err
}

func (r *eventRepository) FindByID(ctx context.Context, id uint) (*entity.Event, error) {
	query := `
		SELECT id, name, description, participation_conditions, active, start_date, end_date, gpx_file_path, telegram_texts, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	event, err := scanEvent(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found: %d", id)
	}

	return event, err
}

func (r *eventRepository) FindByName(ctx context.Context, name string) (*entity.Event, error) {
	query := `
		SELECT id, name, description, participation_conditions, active, start_date, end_date, gpx_file_path, telegram_texts, created_at, updated_at
		FROM events
		WHERE name = $1
	`

	event, err := scanEvent(r.db.QueryRowContext(ctx, query, name))
	if err == sql.ErrNoRows {
		return nil, nil // Событие не найдено - это нормально при проверке существования
	}

	return event, err
}

func (r *eventRepository) FindActive(ctx context.Context) (*entity.Event, error) {
	query := `
		SELECT id, name, description, participation_conditions, active, start_date, end_date, gpx_file_path, telegram_texts, created_at, updated_at
		FROM events
		WHERE active = true
		ORDER BY created_at DESC
		LIMIT 1
	`

	event, err := scanEvent(r.db.QueryRowContext(ctx, query))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active event found")
	}

	return event, err
}

func (r *eventRepository) GetAll(ctx context.Context) ([]*entity.Event, error) {
	query := `
		SELECT id, name, description, participation_conditions, active, start_date, end_date, gpx_file_path, telegram_texts, created_at, updated_at
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
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
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

type eventScanner interface {
	Scan(dest ...interface{}) error
}

func scanEvent(row eventScanner) (*entity.Event, error) {
	event := &entity.Event{}
	var gpxFilePath sql.NullString
	var telegramTextsRaw []byte
	err := row.Scan(
		&event.ID,
		&event.Name,
		&event.Description,
		&event.ParticipationConditions,
		&event.Active,
		&event.StartDate,
		&event.EndDate,
		&gpxFilePath,
		&telegramTextsRaw,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if gpxFilePath.Valid {
		event.GPXFilePath = gpxFilePath.String
	}
	event.ParticipationConditions = entity.NormalizeEventParticipationConditions(event.ParticipationConditions)
	if len(telegramTextsRaw) > 0 {
		if err := json.Unmarshal(telegramTextsRaw, &event.TelegramTexts); err != nil {
			return nil, fmt.Errorf("failed to decode event telegram texts: %w", err)
		}
	}
	event.TelegramTexts = entity.NormalizeEventTelegramTexts(event.TelegramTexts)
	return event, nil
}

func eventTelegramTextsJSON(texts entity.EventTelegramTexts) ([]byte, error) {
	payload, err := json.Marshal(entity.NormalizeEventTelegramTexts(texts))
	if err != nil {
		return nil, fmt.Errorf("failed to encode event telegram texts: %w", err)
	}
	return payload, nil
}
