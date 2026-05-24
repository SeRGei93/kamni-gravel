package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type giftRepository struct {
	db *sql.DB
}

func NewGiftRepository(db *sql.DB) repository.GiftRepository {
	return &giftRepository{db: db}
}

func (r *giftRepository) Create(ctx context.Context, gift *entity.Gift) error {
	query := `INSERT INTO gifts (user_id, event_id, description, gender_filter, bike_type_filter, place, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	if gift.CreatedAt.IsZero() {
		gift.CreatedAt = time.Now()
	}

	// Устанавливаем значения по умолчанию
	genderFilter := gift.GenderFilter
	if genderFilter == "" {
		genderFilter = "all"
	}
	bikeTypeFilter := gift.BikeTypeFilter
	if bikeTypeFilter == "" {
		bikeTypeFilter = "all"
	}

	err := r.db.QueryRowContext(ctx, query,
		gift.UserID, gift.EventID, gift.Description,
		genderFilter, bikeTypeFilter, gift.Place, gift.CreatedAt,
	).Scan(&gift.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *giftRepository) Update(ctx context.Context, gift *entity.Gift) error {
	query := `UPDATE gifts SET description = $1, gender_filter = $2, bike_type_filter = $3, place = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, gift.Description, gift.GenderFilter, gift.BikeTypeFilter, gift.Place, gift.ID)
	return err
}

func (r *giftRepository) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	query := `
		SELECT g.id, g.user_id, g.event_id, g.description, 
		       g.gender_filter, g.bike_type_filter, g.place, g.created_at,
		       u.username, u.first_name, u.last_name
		FROM gifts g
		JOIN users u ON u.id = g.user_id
		WHERE g.id = $1
	`

	gift := &entity.Gift{User: &entity.User{}}
	var genderFilter, bikeTypeFilter sql.NullString
	var place sql.NullInt32
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
		&genderFilter, &bikeTypeFilter, &place, &gift.CreatedAt,
		&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("gift not found: %d", id)
	}

	gift.GenderFilter = genderFilter.String
	gift.BikeTypeFilter = bikeTypeFilter.String
	if place.Valid {
		p := int(place.Int32)
		gift.Place = &p
	}
	gift.User.ID = gift.UserID
	return gift, err
}

func (r *giftRepository) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	query := `
		SELECT g.id, g.user_id, g.event_id, g.description,
		       g.gender_filter, g.bike_type_filter, g.place, g.created_at,
		       u.username, u.first_name, u.last_name
		FROM gifts g
		JOIN users u ON u.id = g.user_id
		WHERE g.event_id = $1
		ORDER BY g.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []*entity.Gift
	for rows.Next() {
		gift := &entity.Gift{User: &entity.User{}}
		var genderFilter, bikeTypeFilter sql.NullString
		var place sql.NullInt32
		err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &place, &gift.CreatedAt,
			&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
		)
		if err != nil {
			return nil, err
		}
		gift.GenderFilter = genderFilter.String
		gift.BikeTypeFilter = bikeTypeFilter.String
		if place.Valid {
			p := int(place.Int32)
			gift.Place = &p
		}
		gift.User.ID = gift.UserID
		gifts = append(gifts, gift)
	}

	return gifts, rows.Err()
}

func (r *giftRepository) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	query := `SELECT id, user_id, event_id, description, created_at FROM gifts WHERE user_id = $1 ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var gifts []*entity.Gift
	for rows.Next() {
		gift := &entity.Gift{}
		err := rows.Scan(&gift.ID, &gift.UserID, &gift.EventID, &gift.Description, &gift.CreatedAt)
		if err != nil {
			return nil, err
		}
		gifts = append(gifts, gift)
	}
	
	return gifts, rows.Err()
}

func (r *giftRepository) Delete(ctx context.Context, id uint) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM gifts WHERE id = $1`, id)
	return err
}

func (r *giftRepository) AddAttachment(ctx context.Context, attachment *entity.GiftAttachment) error {
	query := `INSERT INTO gift_attachments (gift_id, telegram_file_id, file_type) VALUES ($1, $2, $3) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, attachment.GiftID, attachment.TelegramFileID, attachment.FileType).Scan(&attachment.ID)
	if err != nil {
		return err
	}
	
	return nil
}

func (r *giftRepository) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	query := `SELECT id, gift_id, telegram_file_id, file_type FROM gift_attachments WHERE gift_id = $1`
	
	rows, err := r.db.QueryContext(ctx, query, giftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var attachments []*entity.GiftAttachment
	for rows.Next() {
		att := &entity.GiftAttachment{}
		err := rows.Scan(&att.ID, &att.GiftID, &att.TelegramFileID, &att.FileType)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, att)
	}
	
	return attachments, rows.Err()
}
