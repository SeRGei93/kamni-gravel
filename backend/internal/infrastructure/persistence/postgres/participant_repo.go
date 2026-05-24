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

type participantRepository struct {
	db *sql.DB
}

func NewParticipantRepository(db *sql.DB) repository.ParticipantRepository {
	return &participantRepository{db: db}
}

func (r *participantRepository) Create(ctx context.Context, p *entity.Participant) error {
	query := `
		INSERT INTO participants (user_id, event_id, bike_type, gender, notes, registered_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	if p.RegisteredAt.IsZero() {
		p.RegisteredAt = time.Now()
	}

	err := r.db.QueryRowContext(ctx, query,
		p.UserID,
		p.EventID,
		p.BikeType,
		p.Gender,
		p.Notes,
		p.RegisteredAt,
	).Scan(&p.ID)

	return err
}

func (r *participantRepository) Update(ctx context.Context, p *entity.Participant) error {
	query := `
		UPDATE participants
		SET bike_type = $1, gender = $2, notes = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query,
		p.BikeType,
		p.Gender,
		p.Notes,
		p.ID,
	)

	return err
}

func (r *participantRepository) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	query := `
		SELECT p.id, p.user_id, p.event_id, p.bike_type, p.gender, p.notes, p.registered_at,
		       u.username, u.first_name, u.last_name,
		       r.id, r.result_link, r.elapsed_time_sec, r.moving_time_sec, r.is_current, r.submitted_at
		FROM participants p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN results r ON r.participant_id = p.id AND r.is_current = true
		WHERE p.id = $1
	`

	return r.scanParticipant(r.db.QueryRowContext(ctx, query, id))
}

func (r *participantRepository) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	query := `
		SELECT p.id, p.user_id, p.event_id, p.bike_type, p.gender, p.notes, p.registered_at,
		       u.username, u.first_name, u.last_name,
		       r.id, r.result_link, r.elapsed_time_sec, r.moving_time_sec, r.is_current, r.submitted_at
		FROM participants p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN results r ON r.participant_id = p.id AND r.is_current = true
		WHERE p.user_id = $1 AND p.event_id = $2
	`

	return r.scanParticipant(r.db.QueryRowContext(ctx, query, userID, eventID))
}

func (r *participantRepository) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	query := `
		SELECT p.id, p.user_id, p.event_id, p.bike_type, p.gender, p.notes, p.registered_at,
		       u.username, u.first_name, u.last_name,
		       r.id, r.result_link, r.elapsed_time_sec, r.moving_time_sec, r.is_current, r.submitted_at
		FROM participants p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN results r ON r.participant_id = p.id AND r.is_current = true
		WHERE p.event_id = $1
		ORDER BY p.registered_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*entity.Participant
	for rows.Next() {
		p, err := r.scanParticipantFromRows(rows)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, rows.Err()
}

func (r *participantRepository) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	query := `
		SELECT p.id, p.user_id, p.event_id, p.bike_type, p.gender, p.notes, p.registered_at,
		       u.username, u.first_name, u.last_name,
		       r.id, r.result_link, r.elapsed_time_sec, r.moving_time_sec, r.is_current, r.submitted_at
		FROM participants p
		JOIN users u ON u.id = p.user_id
		JOIN results r ON r.participant_id = p.id AND r.is_current = true
		WHERE p.event_id = $1 AND r.elapsed_time_sec IS NOT NULL
		ORDER BY r.elapsed_time_sec ASC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*entity.Participant
	for rows.Next() {
		p, err := r.scanParticipantFromRows(rows)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, rows.Err()
}

func (r *participantRepository) UpdateNotes(ctx context.Context, id uint, notes string) error {
	query := `UPDATE participants SET notes = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, notes, id)
	return err
}

func (r *participantRepository) Delete(ctx context.Context, id uint) error {
	query := `DELETE FROM participants WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *participantRepository) scanParticipant(row *sql.Row) (*entity.Participant, error) {
	p := &entity.Participant{User: &entity.User{}}
	var bikeType, gender string

	// Поля результата (nullable из-за LEFT JOIN)
	var resultID sql.NullInt64
	var resultLink sql.NullString
	var elapsedTimeSec sql.NullInt64
	var movingTimeSec sql.NullInt64
	var isCurrent sql.NullBool
	var submittedAt sql.NullTime

	err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.EventID,
		&bikeType,
		&gender,
		&p.Notes,
		&p.RegisteredAt,
		&p.User.Username,
		&p.User.FirstName,
		&p.User.LastName,
		&resultID,
		&resultLink,
		&elapsedTimeSec,
		&movingTimeSec,
		&isCurrent,
		&submittedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("participant not found")
	}
	if err != nil {
		return nil, err
	}

	p.BikeType, _ = valueobject.NewBikeType(bikeType)
	p.Gender, _ = valueobject.NewGender(gender)
	p.User.ID = p.UserID

	// Заполняем Result если есть данные
	if resultID.Valid {
		p.Result = &entity.Result{
			ID:            uint(resultID.Int64),
			ParticipantID: p.ID,
			IsCurrent:     isCurrent.Bool,
		}
		if submittedAt.Valid {
			p.Result.SubmittedAt = submittedAt.Time
		}
		if elapsedTimeSec.Valid {
			elapsed := int(elapsedTimeSec.Int64)
			p.Result.ElapsedTimeSec = &elapsed
		}
		if movingTimeSec.Valid {
			moving := int(movingTimeSec.Int64)
			p.Result.MovingTimeSec = &moving
		}
		if resultLink.Valid && resultLink.String != "" {
			p.Result.ResultLink, _ = valueobject.NewResultLink(resultLink.String)
		}
	}

	return p, nil
}

func (r *participantRepository) scanParticipantFromRows(rows *sql.Rows) (*entity.Participant, error) {
	p := &entity.Participant{User: &entity.User{}}
	var bikeType, gender string

	// Поля результата (nullable из-за LEFT JOIN)
	var resultID sql.NullInt64
	var resultLink sql.NullString
	var elapsedTimeSec sql.NullInt64
	var movingTimeSec sql.NullInt64
	var isCurrent sql.NullBool
	var submittedAt sql.NullTime

	err := rows.Scan(
		&p.ID,
		&p.UserID,
		&p.EventID,
		&bikeType,
		&gender,
		&p.Notes,
		&p.RegisteredAt,
		&p.User.Username,
		&p.User.FirstName,
		&p.User.LastName,
		&resultID,
		&resultLink,
		&elapsedTimeSec,
		&movingTimeSec,
		&isCurrent,
		&submittedAt,
	)

	if err != nil {
		return nil, err
	}

	p.BikeType, _ = valueobject.NewBikeType(bikeType)
	p.Gender, _ = valueobject.NewGender(gender)
	p.User.ID = p.UserID

	// Заполняем Result если есть данные
	if resultID.Valid {
		p.Result = &entity.Result{
			ID:            uint(resultID.Int64),
			ParticipantID: p.ID,
			IsCurrent:     isCurrent.Bool,
		}
		if submittedAt.Valid {
			p.Result.SubmittedAt = submittedAt.Time
		}
		if elapsedTimeSec.Valid {
			elapsed := int(elapsedTimeSec.Int64)
			p.Result.ElapsedTimeSec = &elapsed
		}
		if movingTimeSec.Valid {
			moving := int(movingTimeSec.Int64)
			p.Result.MovingTimeSec = &moving
		}
		if resultLink.Valid && resultLink.String != "" {
			p.Result.ResultLink, _ = valueobject.NewResultLink(resultLink.String)
		}
	}

	return p, nil
}
