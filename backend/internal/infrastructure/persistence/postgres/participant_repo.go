package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

func (r *participantRepository) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Participant delete transaction begin failed: participant_id=%d stage=begin error=%v", id, err)
		return fmt.Errorf("begin participant delete transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			log.Printf("Participant delete transaction rollback failed: participant_id=%d stage=rollback error=%v", id, rollbackErr)
		}
	}()

	resultIDs, err := participantResultIDs(ctx, tx, id)
	if err != nil {
		log.Printf("Participant delete failed: participant_id=%d stage=find_results error=%v", id, err)
		return fmt.Errorf("find participant results: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM entity_criteria
		 WHERE entity_type = 'result'
		   AND entity_id IN (SELECT id FROM results WHERE participant_id = $1)`,
		id,
	); err != nil {
		log.Printf("Participant delete failed: participant_id=%d stage=delete_result_criteria result_count=%d error=%v", id, len(resultIDs), err)
		return fmt.Errorf("delete participant result criteria: %w", err)
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM participants WHERE id = $1`, id)
	if err != nil {
		log.Printf("Participant delete failed: participant_id=%d stage=delete_participant error=%v", id, err)
		return fmt.Errorf("delete participant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Participant delete failed: participant_id=%d stage=rows_affected error=%v", id, err)
		return fmt.Errorf("delete participant rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: %d", repository.ErrParticipantNotFound, id)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Participant delete transaction commit failed: participant_id=%d stage=commit result_count=%d error=%v", id, len(resultIDs), err)
		return fmt.Errorf("commit participant delete transaction: %w", err)
	}
	committed = true

	log.Printf("Participant delete transaction completed: participant_id=%d result_count=%d", id, len(resultIDs))
	return nil
}

func participantResultIDs(ctx context.Context, tx *sql.Tx, participantID uint) ([]uint, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id FROM results WHERE participant_id = $1`, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resultIDs := make([]uint, 0)
	for rows.Next() {
		var resultID uint
		if err := rows.Scan(&resultID); err != nil {
			return nil, err
		}
		resultIDs = append(resultIDs, resultID)
	}

	return resultIDs, rows.Err()
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
		return nil, repository.ErrParticipantNotFound
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
