-- +goose Up
ALTER TABLE gifts
    ADD COLUMN review_status VARCHAR(50) NOT NULL DEFAULT 'approved';

ALTER TABLE gifts
    ADD CONSTRAINT chk_gifts_review_status CHECK(review_status IN ('pending_review', 'approved'));

CREATE INDEX idx_gifts_event_review_status ON gifts(event_id, review_status);

ALTER TABLE gifts
    ALTER COLUMN review_status SET DEFAULT 'pending_review';

-- +goose Down
DROP INDEX IF EXISTS idx_gifts_event_review_status;

ALTER TABLE gifts
    DROP CONSTRAINT IF EXISTS chk_gifts_review_status;

ALTER TABLE gifts
    DROP COLUMN IF EXISTS review_status;
