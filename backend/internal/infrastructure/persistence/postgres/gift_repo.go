package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"

	"github.com/lib/pq"
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

type queryContextExecutor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func NewGiftRepository(db *sql.DB) repository.ManualGiftRepository {
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
	query := `INSERT INTO gifts (user_id, event_id, description, gender_filter, bike_type_filter, review_status, place, manual_distribution, manual_recipient_participant_id, created_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`

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
		genderFilter, bikeTypeFilter, gift.ReviewStatus.String(), gift.Place,
		gift.ManualDistribution, gift.ManualRecipientParticipantID, gift.CreatedAt,
	).Scan(&gift.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *giftRepository) Update(ctx context.Context, gift *entity.Gift) error {
	if err := normalizeGiftPlaceRuleForUpdate(gift); err != nil {
		return err
	}
	return updateGiftFields(ctx, r.db, gift)
}

func (r *giftRepository) UpdateWithCriteria(ctx context.Context, gift *entity.Gift, criteriaIDs []uint) error {
	if err := normalizeGiftPlaceRuleForUpdate(gift); err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("ERROR gift update transaction begin failed: gift_id=%d rule_type=%s error=%v", gift.ID, gift.PlaceRule.Type(), err)
		return fmt.Errorf("begin gift update transaction for gift %d: %w", gift.ID, err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			log.Printf("ERROR gift update transaction rollback failed: gift_id=%d rule_type=%s error=%v", gift.ID, gift.PlaceRule.Type(), rollbackErr)
		}
	}()

	if err := updateGiftFields(ctx, tx, gift); err != nil {
		log.Printf("ERROR gift update failed: gift_id=%d stage=update_fields rule_type=%s error=%v", gift.ID, gift.PlaceRule.Type(), err)
		return fmt.Errorf("update gift fields: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM entity_criteria WHERE entity_type = 'gift' AND entity_id = $1`, gift.ID); err != nil {
		log.Printf("ERROR gift update failed: gift_id=%d stage=delete_criteria rule_type=%s error=%v", gift.ID, gift.PlaceRule.Type(), err)
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
			log.Printf("ERROR gift update failed: gift_id=%d stage=insert_criteria criteria_index=%d criteria_id=%d rule_type=%s error=%v", gift.ID, index, criteriaID, gift.PlaceRule.Type(), err)
			return fmt.Errorf("replace gift criteria: insert criteria %d: %w", criteriaID, err)
		}
	}

	if err := replaceGiftPlaceRule(ctx, tx, gift); err != nil {
		log.Printf("ERROR gift update failed: gift_id=%d stage=replace_place_rule %s error=%v", gift.ID, giftPlaceRuleLogMeta(gift.PlaceRule), err)
		return fmt.Errorf("replace gift place rule: %w", err)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ERROR gift update transaction commit failed: gift_id=%d criteria_count=%d %s error=%v", gift.ID, len(criteriaIDs), giftPlaceRuleLogMeta(gift.PlaceRule), err)
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

	query := `UPDATE gifts SET description = $1, gender_filter = $2, bike_type_filter = $3, review_status = $4, place = $5, manual_distribution = $6, manual_recipient_participant_id = $7 WHERE id = $8`
	_, err := exec.ExecContext(
		ctx,
		query,
		gift.Description,
		gift.GenderFilter,
		gift.BikeTypeFilter,
		gift.ReviewStatus.String(),
		gift.Place,
		gift.ManualDistribution,
		gift.ManualRecipientParticipantID,
		gift.ID,
	)
	return err
}

func (r *giftRepository) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	query := `
		SELECT g.id, g.user_id, g.event_id, g.description,
		       g.gender_filter, g.bike_type_filter, g.review_status, g.place,
		       g.manual_distribution, g.manual_recipient_participant_id, g.created_at,
		       u.username, u.first_name, u.last_name
		FROM gifts g
		JOIN users u ON u.id = g.user_id
		WHERE g.id = $1
	`

	gift := &entity.Gift{User: &entity.User{}}
	var genderFilter, bikeTypeFilter, reviewStatus sql.NullString
	var place sql.NullInt32
	var manualDistribution sql.NullBool
	var manualRecipientParticipantID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
		&genderFilter, &bikeTypeFilter, &reviewStatus, &place,
		&manualDistribution, &manualRecipientParticipantID, &gift.CreatedAt,
		&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: %d", repository.ErrGiftNotFound, id)
	}
	if err != nil {
		return nil, err
	}

	if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, reviewStatus, place, manualDistribution, manualRecipientParticipantID); err != nil {
		return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
	}
	gift.User.ID = gift.UserID
	if err := r.loadGiftPlaceRules(ctx, []*entity.Gift{gift}); err != nil {
		return nil, err
	}
	if err := r.loadManualRecipients(ctx, []*entity.Gift{gift}); err != nil {
		return nil, err
	}
	return gift, nil
}

func (r *giftRepository) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	query := `
		SELECT g.id, g.user_id, g.event_id, g.description,
		       g.gender_filter, g.bike_type_filter, g.review_status, g.place,
		       g.manual_distribution, g.manual_recipient_participant_id, g.created_at,
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
		var manualDistribution sql.NullBool
		var manualRecipientParticipantID sql.NullInt64
		err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &reviewStatus, &place,
			&manualDistribution, &manualRecipientParticipantID, &gift.CreatedAt,
			&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
		)
		if err != nil {
			return nil, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, reviewStatus, place, manualDistribution, manualRecipientParticipantID); err != nil {
			return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
		}
		gift.User.ID = gift.UserID
		gifts = append(gifts, gift)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadGiftPlaceRules(ctx, gifts); err != nil {
		return nil, err
	}
	if err := r.loadManualRecipients(ctx, gifts); err != nil {
		return nil, err
	}
	return gifts, nil
}

func (r *giftRepository) FindByEventAndReviewStatus(ctx context.Context, eventID uint, reviewStatus entity.GiftReviewStatus) ([]*entity.Gift, error) {
	if !reviewStatus.IsValid() {
		return nil, fmt.Errorf("invalid gift review status: %s", reviewStatus)
	}

	query := `
		SELECT g.id, g.user_id, g.event_id, g.description,
		       g.gender_filter, g.bike_type_filter, g.review_status, g.place,
		       g.manual_distribution, g.manual_recipient_participant_id, g.created_at,
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
		var manualDistribution sql.NullBool
		var manualRecipientParticipantID sql.NullInt64
		err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &scannedReviewStatus, &place,
			&manualDistribution, &manualRecipientParticipantID, &gift.CreatedAt,
			&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
		)
		if err != nil {
			return nil, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, scannedReviewStatus, place, manualDistribution, manualRecipientParticipantID); err != nil {
			return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
		}
		gift.User.ID = gift.UserID
		gifts = append(gifts, gift)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadGiftPlaceRules(ctx, gifts); err != nil {
		return nil, err
	}
	if err := r.loadManualRecipients(ctx, gifts); err != nil {
		return nil, err
	}
	return gifts, nil
}

func (r *giftRepository) ListByEventPaged(ctx context.Context, eventID uint, reviewStatus *entity.GiftReviewStatus, limit, offset int) ([]*entity.Gift, int, error) {
	return r.ListByEventFilteredPaged(ctx, eventID, repository.GiftListFilter{
		ReviewStatus: reviewStatus,
	}, limit, offset)
}

func (r *giftRepository) ListByEventFilteredPaged(ctx context.Context, eventID uint, filter repository.GiftListFilter, limit, offset int) ([]*entity.Gift, int, error) {
	whereClauses := []string{"g.event_id = $1"}
	args := []any{eventID}
	if filter.ReviewStatus != nil {
		if !filter.ReviewStatus.IsValid() {
			return nil, 0, fmt.Errorf("invalid gift review status: %s", filter.ReviewStatus)
		}
		args = append(args, filter.ReviewStatus.String())
		whereClauses = append(whereClauses, fmt.Sprintf("g.review_status = $%d", len(args)))
	}
	if filter.OwnerUserID != nil {
		args = append(args, *filter.OwnerUserID)
		whereClauses = append(whereClauses, fmt.Sprintf("g.user_id = $%d", len(args)))
	}
	if searchQuery := strings.TrimSpace(filter.SearchQuery); searchQuery != "" {
		args = append(args, "%"+searchQuery+"%")
		placeholder := len(args)
		whereClauses = append(whereClauses, fmt.Sprintf(
			"(g.description ILIKE $%d OR u.username ILIKE $%d OR CONCAT_WS(' ', u.first_name, u.last_name) ILIKE $%d)",
			placeholder,
			placeholder,
			placeholder,
		))
	}
	where := "WHERE " + strings.Join(whereClauses, " AND ")
	from := "FROM gifts g JOIN users u ON u.id = g.user_id"

	// Общее количество с учётом фильтра.
	var total int
	countQuery := "SELECT COUNT(*) " + from + " " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Страница. limit <= 0 — все строки.
	listArgs := append([]interface{}{}, args...)
	limitClause := ""
	if limit > 0 {
		listArgs = append(listArgs, limit, offset)
		limitClause = fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	}
	listQuery := fmt.Sprintf(`
		SELECT g.id, g.user_id, g.event_id, g.description,
		       g.gender_filter, g.bike_type_filter, g.review_status, g.place,
		       g.manual_distribution, g.manual_recipient_participant_id, g.created_at,
		       u.username, u.first_name, u.last_name
		%s
		%s
		ORDER BY g.created_at DESC%s
	`, from, where, limitClause)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	gifts := make([]*entity.Gift, 0, limit)
	for rows.Next() {
		gift := &entity.Gift{User: &entity.User{}}
		var genderFilter, bikeTypeFilter, scannedReviewStatus sql.NullString
		var place sql.NullInt32
		var manualDistribution sql.NullBool
		var manualRecipientParticipantID sql.NullInt64
		if err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &scannedReviewStatus, &place,
			&manualDistribution, &manualRecipientParticipantID, &gift.CreatedAt,
			&gift.User.Username, &gift.User.FirstName, &gift.User.LastName,
		); err != nil {
			return nil, 0, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, scannedReviewStatus, place, manualDistribution, manualRecipientParticipantID); err != nil {
			return nil, 0, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
		}
		gift.User.ID = gift.UserID
		gifts = append(gifts, gift)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := r.loadGiftPlaceRules(ctx, gifts); err != nil {
		return nil, 0, err
	}
	if err := r.loadManualRecipients(ctx, gifts); err != nil {
		return nil, 0, err
	}
	return gifts, total, nil
}

func (r *giftRepository) CountsByReviewStatus(ctx context.Context, eventID uint) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT review_status, COUNT(*) FROM gifts WHERE event_id = $1 GROUP BY review_status`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Гарантируем наличие известных ключей даже при нулевом количестве.
	counts := map[string]int{
		entity.GiftReviewStatusPendingReview.String(): 0,
		entity.GiftReviewStatusApproved.String():      0,
	}
	total := 0
	for rows.Next() {
		var status sql.NullString
		var c int
		if err := rows.Scan(&status, &c); err != nil {
			return nil, err
		}
		if status.Valid {
			counts[status.String] = c
		}
		total += c
	}
	counts["all"] = total
	return counts, rows.Err()
}

func (r *giftRepository) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	query := `
		SELECT id, user_id, event_id, description, gender_filter, bike_type_filter, review_status, place,
		       manual_distribution, manual_recipient_participant_id, created_at
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
		var manualDistribution sql.NullBool
		var manualRecipientParticipantID sql.NullInt64
		err := rows.Scan(
			&gift.ID, &gift.UserID, &gift.EventID, &gift.Description,
			&genderFilter, &bikeTypeFilter, &reviewStatus, &place,
			&manualDistribution, &manualRecipientParticipantID, &gift.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, reviewStatus, place, manualDistribution, manualRecipientParticipantID); err != nil {
			return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
		}
		gifts = append(gifts, gift)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := r.loadGiftPlaceRules(ctx, gifts); err != nil {
		return nil, err
	}
	if err := r.loadManualRecipients(ctx, gifts); err != nil {
		return nil, err
	}
	return gifts, nil
}

func (r *giftRepository) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) ([]*entity.Gift, error) {
	const query = `
		SELECT id, user_id, event_id, description, gender_filter, bike_type_filter, review_status, place,
		       manual_distribution, manual_recipient_participant_id, created_at
		FROM gifts
		WHERE user_id = $1 AND event_id = $2
		ORDER BY created_at DESC
	`

	log.Printf("DEBUG gift owner lookup started: user_id=%d event_id=%d", userID, eventID)
	rows, err := r.db.QueryContext(ctx, query, userID, eventID)
	gifts, err := scanGifts(rows, err)
	if err != nil {
		log.Printf("ERROR gift owner lookup failed: user_id=%d event_id=%d stage=scan_gifts error=%v", userID, eventID, err)
		return nil, err
	}
	if err := r.loadGiftPlaceRules(ctx, gifts); err != nil {
		log.Printf("ERROR gift owner lookup failed: user_id=%d event_id=%d stage=load_place_rules error=%v", userID, eventID, err)
		return nil, err
	}
	if err := r.loadManualRecipients(ctx, gifts); err != nil {
		log.Printf("ERROR gift owner lookup failed: user_id=%d event_id=%d stage=load_manual_recipients error=%v", userID, eventID, err)
		return nil, err
	}

	log.Printf("DEBUG gift owner lookup completed: user_id=%d event_id=%d gift_count=%d", userID, eventID, len(gifts))
	return gifts, nil
}

func (r *giftRepository) HasByUserAndEvent(ctx context.Context, userID int64, eventID uint) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1
			FROM gifts
			WHERE user_id = $1 AND event_id = $2
		)
	`

	var hasGifts bool
	if err := r.db.QueryRowContext(ctx, query, userID, eventID).Scan(&hasGifts); err != nil {
		log.Printf("ERROR gift owner existence lookup failed: user_id=%d event_id=%d error=%v", userID, eventID, err)
		return false, err
	}

	log.Printf("DEBUG gift owner existence lookup completed: user_id=%d event_id=%d has_gifts=%t", userID, eventID, hasGifts)
	return hasGifts, nil
}

func scanGifts(rows *sql.Rows, queryErr error) ([]*entity.Gift, error) {
	if queryErr != nil {
		return nil, queryErr
	}
	defer rows.Close()

	gifts := make([]*entity.Gift, 0)
	for rows.Next() {
		gift := &entity.Gift{}
		var genderFilter, bikeTypeFilter, reviewStatus sql.NullString
		var place sql.NullInt32
		var manualDistribution sql.NullBool
		var manualRecipientParticipantID sql.NullInt64
		if err := rows.Scan(
			&gift.ID,
			&gift.UserID,
			&gift.EventID,
			&gift.Description,
			&genderFilter,
			&bikeTypeFilter,
			&reviewStatus,
			&place,
			&manualDistribution,
			&manualRecipientParticipantID,
			&gift.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := applyGiftNullableFields(gift, genderFilter, bikeTypeFilter, reviewStatus, place, manualDistribution, manualRecipientParticipantID); err != nil {
			return nil, fmt.Errorf("invalid stored gift fields for gift %d: %w", gift.ID, err)
		}
		gifts = append(gifts, gift)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return gifts, nil
}

func (r *giftRepository) SetManualRecipient(ctx context.Context, giftID uint, recipientParticipantID *uint) error {
	const query = `
		WITH target_gift AS (
			SELECT id, event_id, manual_distribution
			FROM gifts
			WHERE id = $1
		),
		target_participant AS (
			SELECT id, event_id
			FROM participants
			WHERE id = $2
		),
		updated AS (
			UPDATE gifts AS g
			SET manual_recipient_participant_id = $2
			WHERE g.id = $1
			  AND g.manual_distribution
			  AND (
				$2 IS NULL
				OR EXISTS (
					SELECT 1
					FROM participants AS p
					WHERE p.id = $2 AND p.event_id = g.event_id
				)
			  )
			RETURNING g.id
		)
		SELECT CASE
			WHEN EXISTS (SELECT 1 FROM updated) THEN 'updated'
			WHEN NOT EXISTS (SELECT 1 FROM target_gift) THEN 'gift_not_found'
			WHEN NOT (SELECT manual_distribution FROM target_gift) THEN 'manual_distribution_disabled'
			WHEN $2 IS NOT NULL AND NOT EXISTS (SELECT 1 FROM target_participant) THEN 'recipient_not_found'
			WHEN $2 IS NOT NULL
				AND (SELECT event_id FROM target_participant) <> (SELECT event_id FROM target_gift)
				THEN 'recipient_event_mismatch'
			ELSE 'update_rejected'
		END
	`

	recipientIDLogValue := manualRecipientIDLogValue(recipientParticipantID)
	log.Printf("DEBUG manual gift recipient update started: gift_id=%d recipient_participant_id=%s", giftID, recipientIDLogValue)
	var outcome string
	if err := r.db.QueryRowContext(ctx, query, giftID, recipientParticipantID).Scan(&outcome); err != nil {
		log.Printf("ERROR manual gift recipient update failed: gift_id=%d recipient_participant_id=%s stage=execute error=%v", giftID, recipientIDLogValue, err)
		return fmt.Errorf("update manual gift recipient for gift %d: %w", giftID, err)
	}

	switch outcome {
	case "updated":
		log.Printf("DEBUG manual gift recipient update completed: gift_id=%d recipient_participant_id=%s", giftID, recipientIDLogValue)
		return nil
	case "gift_not_found":
		log.Printf("WARN manual gift recipient update rejected: gift_id=%d recipient_participant_id=%s reason=gift_not_found", giftID, recipientIDLogValue)
		return fmt.Errorf("%w: %d", repository.ErrGiftNotFound, giftID)
	case "manual_distribution_disabled":
		log.Printf("WARN manual gift recipient update rejected: gift_id=%d recipient_participant_id=%s reason=manual_distribution_disabled", giftID, recipientIDLogValue)
		return fmt.Errorf("%w: gift_id=%d", repository.ErrManualDistributionDisabled, giftID)
	case "recipient_not_found":
		log.Printf("WARN manual gift recipient update rejected: gift_id=%d recipient_participant_id=%s reason=recipient_not_found", giftID, recipientIDLogValue)
		return fmt.Errorf("%w: participant_id=%d", repository.ErrManualRecipientNotFound, *recipientParticipantID)
	case "recipient_event_mismatch":
		log.Printf("WARN manual gift recipient update rejected: gift_id=%d recipient_participant_id=%s reason=recipient_event_mismatch", giftID, recipientIDLogValue)
		return fmt.Errorf("%w: gift_id=%d participant_id=%d", repository.ErrManualRecipientEventMismatch, giftID, *recipientParticipantID)
	default:
		log.Printf("ERROR manual gift recipient update failed: gift_id=%d recipient_participant_id=%s stage=classify outcome=%s", giftID, recipientIDLogValue, outcome)
		return fmt.Errorf("manual gift recipient update rejected for gift %d", giftID)
	}
}

func (r *giftRepository) ManualRecipientCountsByEvent(ctx context.Context, eventID uint) (map[uint]int, error) {
	const query = `
		SELECT manual_recipient_participant_id, COUNT(*)
		FROM gifts
		WHERE event_id = $1
		  AND manual_distribution = TRUE
		  AND manual_recipient_participant_id IS NOT NULL
		GROUP BY manual_recipient_participant_id
	`

	log.Printf("DEBUG manual gift recipient counts query started: event_id=%d", eventID)
	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		log.Printf("ERROR manual gift recipient counts query failed: event_id=%d stage=query error=%v", eventID, err)
		return nil, fmt.Errorf("manual gift recipient counts for event %d: %w", eventID, err)
	}
	defer rows.Close()

	counts := make(map[uint]int)
	for rows.Next() {
		var participantID uint
		var count int
		if err := rows.Scan(&participantID, &count); err != nil {
			log.Printf("ERROR manual gift recipient counts query failed: event_id=%d stage=scan error=%v", eventID, err)
			return nil, fmt.Errorf("scan manual gift recipient count: %w", err)
		}
		counts[participantID] = count
	}
	if err := rows.Err(); err != nil {
		log.Printf("ERROR manual gift recipient counts query failed: event_id=%d stage=iterate error=%v", eventID, err)
		return nil, fmt.Errorf("iterate manual gift recipient counts: %w", err)
	}
	log.Printf("DEBUG manual gift recipient counts query completed: event_id=%d participant_count=%d", eventID, len(counts))
	return counts, nil
}

func manualRecipientIDLogValue(recipientParticipantID *uint) string {
	if recipientParticipantID == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *recipientParticipantID)
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
	manualDistribution sql.NullBool,
	manualRecipientParticipantID sql.NullInt64,
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
	} else {
		gift.Place = nil
	}

	// До миграции значение могло быть NULL только в устаревших тестовых данных.
	// Такое состояние соответствует автоматическому распределению по умолчанию.
	gift.ManualDistribution = manualDistribution.Valid && manualDistribution.Bool
	if manualRecipientParticipantID.Valid {
		if manualRecipientParticipantID.Int64 < 0 {
			return fmt.Errorf("manual recipient participant ID must be non-negative")
		}
		recipientID := uint(manualRecipientParticipantID.Int64)
		gift.ManualRecipientParticipantID = &recipientID
	} else {
		gift.ManualRecipientParticipantID = nil
	}

	return nil
}

func normalizeGiftPlaceRuleForUpdate(gift *entity.Gift) error {
	if gift.PlaceRule.IsNone() && gift.Place != nil {
		rule, err := valueobject.NewGiftPlaceRulePlaces([]int{*gift.Place})
		if err != nil {
			return err
		}
		gift.PlaceRule = rule
	}
	gift.Place = gift.PlaceRule.FirstLegacyPlace()
	return nil
}

func (r *giftRepository) loadGiftPlaceRules(ctx context.Context, gifts []*entity.Gift) error {
	return loadGiftPlaceRules(ctx, r.db, gifts)
}

func (r *giftRepository) loadManualRecipients(ctx context.Context, gifts []*entity.Gift) error {
	if len(gifts) == 0 {
		return nil
	}

	giftsByRecipientID := make(map[uint][]*entity.Gift)
	recipientIDs := make([]uint, 0)
	for _, gift := range gifts {
		if gift.ManualRecipientParticipantID == nil {
			continue
		}
		recipientID := *gift.ManualRecipientParticipantID
		if _, exists := giftsByRecipientID[recipientID]; !exists {
			recipientIDs = append(recipientIDs, recipientID)
		}
		giftsByRecipientID[recipientID] = append(giftsByRecipientID[recipientID], gift)
	}
	if len(recipientIDs) == 0 {
		return nil
	}

	log.Printf("DEBUG manual gift recipients loading: recipient_count=%d", len(recipientIDs))
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.user_id, p.event_id, p.status,
		       u.username, u.first_name, u.last_name
		FROM participants p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = ANY($1)
	`, pq.Array(recipientIDs))
	if err != nil {
		log.Printf("ERROR manual gift recipients loading failed: recipient_count=%d stage=query error=%v", len(recipientIDs), err)
		return fmt.Errorf("load manual gift recipients: %w", err)
	}
	defer rows.Close()

	loadedRecipients := 0
	for rows.Next() {
		recipient := &entity.Participant{User: &entity.User{}}
		var status string
		if err := rows.Scan(
			&recipient.ID,
			&recipient.UserID,
			&recipient.EventID,
			&status,
			&recipient.User.Username,
			&recipient.User.FirstName,
			&recipient.User.LastName,
		); err != nil {
			log.Printf("ERROR manual gift recipients loading failed: recipient_count=%d stage=scan error=%v", len(recipientIDs), err)
			return fmt.Errorf("scan manual gift recipient: %w", err)
		}

		participantStatus, err := valueobject.NewParticipantStatus(status)
		if err != nil {
			log.Printf("ERROR manual gift recipients loading failed: recipient_participant_id=%d stage=validate_status error=%v", recipient.ID, err)
			return fmt.Errorf("invalid manual gift recipient status for participant %d: %w", recipient.ID, err)
		}
		recipient.Status = participantStatus
		recipient.User.ID = recipient.UserID
		for _, gift := range giftsByRecipientID[recipient.ID] {
			gift.ManualRecipient = recipient
		}
		loadedRecipients++
	}
	if err := rows.Err(); err != nil {
		log.Printf("ERROR manual gift recipients loading failed: recipient_count=%d stage=iterate error=%v", len(recipientIDs), err)
		return fmt.Errorf("iterate manual gift recipients: %w", err)
	}
	log.Printf("DEBUG manual gift recipients loaded: requested_count=%d loaded_count=%d", len(recipientIDs), loadedRecipients)
	return nil
}

func loadGiftPlaceRules(ctx context.Context, db queryContextExecutor, gifts []*entity.Gift) error {
	if len(gifts) == 0 {
		return nil
	}

	giftsByID := make(map[uint]*entity.Gift, len(gifts))
	giftIDs := make([]uint, 0, len(gifts))
	for _, gift := range gifts {
		giftsByID[gift.ID] = gift
		giftIDs = append(giftIDs, gift.ID)
	}

	query := `
		SELECT r.gift_id, r.rule_type, r.last_count, p.place
		FROM gift_place_rules r
		LEFT JOIN gift_place_rule_places p ON p.gift_id = r.gift_id
		WHERE r.gift_id = ANY($1)
		ORDER BY r.gift_id, p.place
	`

	rows, err := db.QueryContext(ctx, query, pq.Array(giftIDs))
	if err != nil {
		return fmt.Errorf("load gift place rules: %w", err)
	}
	defer rows.Close()

	storedRules := make(map[uint]*storedGiftPlaceRule)
	for rows.Next() {
		var giftID uint
		var ruleType string
		var lastCount sql.NullInt32
		var place sql.NullInt32
		if err := rows.Scan(&giftID, &ruleType, &lastCount, &place); err != nil {
			return fmt.Errorf("scan gift place rule: %w", err)
		}
		if _, ok := giftsByID[giftID]; !ok {
			continue
		}

		storedRule := storedRules[giftID]
		if storedRule == nil {
			storedRule = &storedGiftPlaceRule{ruleType: ruleType, lastCount: lastCount}
			storedRules[giftID] = storedRule
		}
		if place.Valid {
			storedRule.places = append(storedRule.places, int(place.Int32))
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate gift place rules: %w", err)
	}

	for giftID, storedRule := range storedRules {
		rule, err := storedRule.toValueObject()
		if err != nil {
			return fmt.Errorf("invalid stored gift place rule for gift %d: %w", giftID, err)
		}
		giftsByID[giftID].PlaceRule = rule
		giftsByID[giftID].Place = rule.FirstLegacyPlace()
	}

	return nil
}

type storedGiftPlaceRule struct {
	ruleType  string
	lastCount sql.NullInt32
	places    []int
}

func (r *storedGiftPlaceRule) toValueObject() (valueobject.GiftPlaceRule, error) {
	switch valueobject.GiftPlaceRuleType(r.ruleType) {
	case valueobject.GiftPlaceRuleTypePlaces:
		return valueobject.NewGiftPlaceRulePlaces(r.places)
	case valueobject.GiftPlaceRuleTypeLastN:
		if !r.lastCount.Valid {
			return valueobject.GiftPlaceRule{}, fmt.Errorf("last_n rule is missing last_count")
		}
		return valueobject.NewGiftPlaceRuleLastN(int(r.lastCount.Int32))
	default:
		return valueobject.GiftPlaceRule{}, fmt.Errorf("unsupported rule type: %s", r.ruleType)
	}
}

func replaceGiftPlaceRule(ctx context.Context, exec execContextExecutor, gift *entity.Gift) error {
	if _, err := exec.ExecContext(ctx, `DELETE FROM gift_place_rules WHERE gift_id = $1`, gift.ID); err != nil {
		return fmt.Errorf("delete existing rule: %w", err)
	}

	switch gift.PlaceRule.Type() {
	case valueobject.GiftPlaceRuleTypeNone:
		return nil
	case valueobject.GiftPlaceRuleTypePlaces:
		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO gift_place_rules (gift_id, rule_type, last_count) VALUES ($1, $2, $3)`,
			gift.ID,
			string(valueobject.GiftPlaceRuleTypePlaces),
			nil,
		); err != nil {
			return fmt.Errorf("insert places rule: %w", err)
		}
		for _, place := range gift.PlaceRule.Places() {
			if _, err := exec.ExecContext(
				ctx,
				`INSERT INTO gift_place_rule_places (gift_id, place) VALUES ($1, $2)`,
				gift.ID,
				place,
			); err != nil {
				return fmt.Errorf("insert rule place %d: %w", place, err)
			}
		}
		return nil
	case valueobject.GiftPlaceRuleTypeLastN:
		if _, err := exec.ExecContext(
			ctx,
			`INSERT INTO gift_place_rules (gift_id, rule_type, last_count) VALUES ($1, $2, $3)`,
			gift.ID,
			string(valueobject.GiftPlaceRuleTypeLastN),
			gift.PlaceRule.LastCount(),
		); err != nil {
			return fmt.Errorf("insert last_n rule: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported gift place rule type: %s", gift.PlaceRule.Type())
	}
}

func giftPlaceRuleLogMeta(rule valueobject.GiftPlaceRule) string {
	return fmt.Sprintf("rule_type=%s place_count=%d last_count=%d", rule.Type(), len(rule.Places()), rule.LastCount())
}
