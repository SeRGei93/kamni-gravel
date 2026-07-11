-- +goose Up
ALTER TABLE gifts
    ADD COLUMN manual_distribution BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN manual_recipient_participant_id INTEGER NULL
        REFERENCES participants(id) ON DELETE SET NULL;

ALTER TABLE gifts
    ADD CONSTRAINT chk_gifts_manual_recipient_requires_manual_distribution
        CHECK (manual_distribution OR manual_recipient_participant_id IS NULL);

CREATE INDEX idx_gifts_manual_recipient_participant_id
    ON gifts(manual_recipient_participant_id)
    WHERE manual_recipient_participant_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_gifts_manual_recipient_participant_id;

ALTER TABLE gifts
    DROP CONSTRAINT IF EXISTS chk_gifts_manual_recipient_requires_manual_distribution;

ALTER TABLE gifts
    DROP COLUMN IF EXISTS manual_recipient_participant_id,
    DROP COLUMN IF EXISTS manual_distribution;
