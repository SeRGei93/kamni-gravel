package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

type giftRepository struct {
	db *sql.DB
}

type queryRowExecutor interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execContextExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func NewGiftRepository(db *sql.DB) repository.GiftRepository {
	return &giftRepository{db: db}
}

func (r *giftRepository) Create(ctx context.Context, gift *entity.Gift) error {
	return insertGift(ctx, r.db, gift)
}

func (r *giftRepository) CreateWithAttachments(ctx context.Context, gift *entity.Gift, attachments []*entity.GiftAttachment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Gift create transaction begin failed: user_id=%d event_id=%d error=%v", gift.UserID, gift.EventID, err)
		return fmt.Errorf("begin gift create transaction for user %d event %d: %w", gift.UserID, gift.EventID, err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			log.Printf("Gift create transaction rollback failed: gift_id=%d user_id=%d event_id=%d error=%v", gift.ID, gift.UserID, gift.EventID, rollbackErr)
		}
	}()

	if err := insertGift(ctx, tx, gift); err != nil {
		return fmt.Errorf("create gift with attachments: insert gift user_id=%d event_id=%d: %w", gift.UserID, gift.EventID, err)
	}

	for index, attachment := range attachments {
		attachment.GiftID = gift.ID
		if err := insertGiftAttachment(ctx, tx, attachment); err != nil {
			return fmt.Errorf("create gift with attachments: insert attachment gift_id=%d attachment_index=%d: %w", gift.ID, index, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Gift create transaction commit failed: gift_id=%d user_id=%d event_id=%d attachments=%d error=%v", gift.ID, gift.UserID, gift.EventID, len(attachments), err)
		return fmt.Errorf("commit gift create transaction for gift %d: %w", gift.ID, err)
	}
	committed = true

	return nil
}

func insertGift(ctx context.Context, exec queryRowExecutor, gift *entity.Gift) error {
	query := `INSERT INTO gifts (user_id, event_id, description, gender_filter, bike_type_filter, review_status, place, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

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
	if gift.ReviewStatus == "" {
		gift.ReviewStatus = entity.GiftReviewStatusPendingReview
	}
	if !gift.ReviewStatus.IsValid() {
		return fmt.Errorf("invalid gift review status: %s", gift.ReviewStatus)
	}

	err := exec.QueryRowContext(ctx, query,
		gift.UserID, gift.EventID, gift.Description,
		genderFilter, bikeTypeFilter, gift.ReviewStatus.String(), gift.Place, gift.CreatedAt,
	).Scan(&gift.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *giftRepository) Update(ctx context.Context, gift *entity.Gift) error {
	return updateGiftFields(ctx, r.db, gift)
}

func (r *giftRepository) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Gift update transaction begin failed: gift_id=%d error=%v", gift.ID, err)
		return fmt.Errorf("begin gift update transaction for gift %d: %w", gift.ID, err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			log.Printf("Gift update transaction rollback failed: gift_id=%d error=%v", gift.ID, rollbackErr)
		}
	}()

	if err := updateGiftFields(ctx, tx, gift); err != nil {
		log.Printf("Gift update failed: gift_id=%d stage=update_fields error=%v", gift.ID, err)
		return fmt.Errorf("update gift fields: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_criteria WHERE entity_type = 'gift' AND entity_id = $1`, gift.ID); err != nil {
		log.Printf("Gift update failed: gift_id=%d stage=delete_criteria error=%v", gift.ID, err)
		return fmt.Errorf("replace gift criteria: delete old criteria: %w", err)
	}

	for index, criteriaID := range criteriaIDs {
		_, err := tx.ExecContext(
			ctx,
			`INSERT INTO entity_criteria (entity_type, entity_id, criteria_id) VALUES ('gift', $1, $2) ON CONFLICT (entity_type, entity_id, criteria_id) DO NOTHING`,
			gift.ID,
			criteriaID,
		)
		if err != nil {
			log.Printf("Gift update failed: gift_id=%d stage=insert_criteria criteria_index=%d criteria_id=%d error=%v", gift.ID, index, criteriaID, err)
			return fmt.Errorf("replace gift criteria: insert criteria %d: %w", criteriaID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Gift update transaction commit failed: gift_id=%d criteria_count=%d error=%v", gift.ID, len(criteriaIDs), err)
		return fmt.Errorf("commit gift update transaction for gift %d: %w", gift.ID, err)
	}
	committed = true

	return nil
}

func updateGiftFields(ctx context.Context, exec execContextExecutor, gift *entity.Gift) error {
	if gift.ReviewStatus == "" {
		gift.ReviewStatus = entity.GiftReviewStatusPendingReview
	}
	if !gift.ReviewStatus.IsValid() {
		return fmt.Errorf("invalid gift review status: %s", gift.ReviewStatus)
	}

	query := `UPDATE gifts SET description = $1, gender_filter = $2, bike_type_filter = $3, review_status = $4, place = $5 WHERE id = $6`
	_, err := exec.ExecContext(ctx, query, gift.Description, gift.GenderFilter, gift.BikeTypeFilter, gift.ReviewStatus.String(), gift.Place, gift.ID)
	return err
}

func (r *giftRepository) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	query := `
		SELECT g.id, g.user_id, g.event_id, g.description, 
		       g.gender_filter, g.bike_type_filter, g.review_status, g.place, g.created_at,
		       u.username, u.first_name, u.last_name
		FROM gifts g
		JOIN users u ON u.id = g.user_id
		WHERE g.id = $1
	`

	gift := &entity.Gift{User: &entity.User{}}
	var genderFilter, bikeTypeFilter, reviewStatus sql.NullString
	var place sql.NullInt32
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
		&genderFilter, &bikeTypeFilter, &reviewStatus, &place, &gift.CreatedAt,
		&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("gift not found: %d", id)
	}
	if err != nil {
		return nil, err
	}

	if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, reviewStatus, place); err != nil {
		return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
	}
	gift.User.ID = gift.UserID
	return gift, nil
}

func (r *giftRepository) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	query := `
		SELECT g.id, g.user_id, g.event_id, g.description,
		       g.gender_filter, g.bike_type_filter, g.review_status, g.place, g.created_at,
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
		var genderFilter, bikeTypeFilter, reviewStatus sql.NullString
		var place sql.NullInt32
		err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &reviewStatus, &place, &gift.CreatedAt,
			&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
		)
		if err != nil {
			return nil, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, reviewStatus, place); err != nil {
			return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
		}
		gift.User.ID = gift.UserID
		gifts = append(gifts, gift)
	}

	return gifts, rows.Err()
}

func (r *giftRepository) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	if !reviewStatus.IsValid() {
		return nil, fmt.Errorf("invalid gift review status: %s", reviewStatus)
	}

	query := `
		SELECT g.id, g.user_id, g.event_id, g.description,
		       g.gender_filter, g.bike_type_filter, g.review_status, g.place, g.created_at,
		       u.username, u.first_name, u.last_name
		FROM gifts g
		JOIN users u ON u.id = g.user_id
		WHERE g.event_id = $1 AND g.review_status = $2
		ORDER BY g.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID, reviewStatus.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []*entity.Gift
	for rows.Next() {
		gift := &entity.Gift{User: &entity.User{}}
		var genderFilter, bikeTypeFilter, scannedReviewStatus sql.NullString
		var place sql.NullInt32
		err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &scannedReviewStatus, &place, &gift.CreatedAt,
			&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
		)
		if err != nil {
			return nil, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, scannedReviewStatus, place); err != nil {
			return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
		}
		gift.User.ID = gift.UserID
		gifts = append(gifts, gift)
	}

	return gifts, rows.Err()
}

func (r *giftRepository) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	query := `
		SELECT id, user_id, event_id, description, gender_filter, bike_type_filter, review_status, place, created_at
		FROM gifts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []*entity.Gift
	for rows.Next() {
		gift := &entity.Gift{}
		var genderFilter, bikeTypeFilter, reviewStatus sql.NullString
		var place sql.NullInt32
		err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &reviewStatus, &place, &gift.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, reviewStatus, place); err != nil {
			return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
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
	return insertGiftAttachment(ctx, r.db, attachment)
}

func insertGiftAttachment(ctx context.Context, exec queryRowExecutor, attachment *entity.GiftAttachment) error {
	query := `INSERT INTO gift_attachments (gift_id, telegram_file_id, file_type) VALUES ($1, $2, $3) RETURNING id`
	err := exec.QueryRowContext(ctx, query, attachment.GiftID, attachment.TelegramFileID, attachment.FileType).Scan(&attachment.ID)
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

func applyGiftNullableFields(
	gift *entity.Gift,
	genderFilter sql.NullString,
	bikeTypeFilter sql.NullString,
	reviewStatus sql.NullString,
	place sql.NullInt32,
) error {
	gift.GenderFilter = genderFilter.String
	if !genderFilter.Valid || gift.GenderFilter == "" {
		gift.GenderFilter = "all"
	}

	gift.BikeTypeFilter = bikeTypeFilter.String
	if !bikeTypeFilter.Valid || gift.BikeTypeFilter == "" {
		gift.BikeTypeFilter = "all"
	}

	statusValue := reviewStatus.String
	if !reviewStatus.Valid || statusValue == "" {
		statusValue = entity.GiftReviewStatusPendingReview.String()
	}
	status, err := entity.NewGiftReviewStatus(statusValue)
	if err != nil {
		return err
	}
	gift.ReviewStatus = status

	if place.Valid {
		p := int(place.Int32)
		gift.Place = &p
	}

	return nil
}
