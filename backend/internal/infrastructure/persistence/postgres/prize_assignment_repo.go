package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type prizeAssignmentRepository struct {
	db *sql.DB
}

func NewPrizeAssignmentRepository(db *sql.DB) repository.PrizeAssignmentRepository {
	return &prizeAssignmentRepository{db: db}
}

func (r *prizeAssignmentRepository) Assign(ctx context.Context, pa *entity.PrizeAssignment) error {
	query := `INSERT INTO prize_assignments (participant_id, gift_id, comment, assigned_at) VALUES ($1, $2, $3, $4) RETURNING id`
	
	if pa.AssignedAt.IsZero() {
		pa.AssignedAt = time.Now()
	}
	
	err := r.db.QueryRowContext(ctx, query, pa.ParticipantID, pa.GiftID, pa.Comment, pa.AssignedAt).Scan(&pa.ID)
	if err != nil {
		return err
	}
	
	return nil
}

func (r *prizeAssignmentRepository) Update(ctx context.Context, pa *entity.PrizeAssignment) error {
	query := `UPDATE prize_assignments SET comment = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, pa.Comment, pa.ID)
	return err
}

func (r *prizeAssignmentRepository) FindByID(ctx context.Context, id uint) (*entity.PrizeAssignment, error) {
	query := `SELECT id, participant_id, gift_id, comment, assigned_at FROM prize_assignments WHERE id = $1`
	
	pa := &entity.PrizeAssignment{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&pa.ID, &pa.ParticipantID, &pa.GiftID, &pa.Comment, &pa.AssignedAt)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prize assignment not found: %d", id)
	}
	
	return pa, err
}

func (r *prizeAssignmentRepository) FindByParticipant(ctx context.Context, participantID uint) ([]*entity.PrizeAssignment, error) {
	query := `SELECT id, participant_id, gift_id, comment, assigned_at FROM prize_assignments WHERE participant_id = $1`
	
	rows, err := r.db.QueryContext(ctx, query, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assignments []*entity.PrizeAssignment
	for rows.Next() {
		pa := &entity.PrizeAssignment{}
		err := rows.Scan(&pa.ID, &pa.ParticipantID, &pa.GiftID, &pa.Comment, &pa.AssignedAt)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, pa)
	}
	
	return assignments, rows.Err()
}

func (r *prizeAssignmentRepository) FindByEvent(ctx context.Context, eventID uint) ([]*entity.PrizeAssignment, error) {
	query := `
		SELECT pa.id, pa.participant_id, pa.gift_id, pa.comment, pa.assigned_at
		FROM prize_assignments pa
		JOIN participants p ON p.id = pa.participant_id
		WHERE p.event_id = $1
	`
	
	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assignments []*entity.PrizeAssignment
	for rows.Next() {
		pa := &entity.PrizeAssignment{}
		err := rows.Scan(&pa.ID, &pa.ParticipantID, &pa.GiftID, &pa.Comment, &pa.AssignedAt)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, pa)
	}
	
	return assignments, rows.Err()
}

func (r *prizeAssignmentRepository) Remove(ctx context.Context, id uint) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM prize_assignments WHERE id = $1`, id)
	return err
}
