-- +goose Up
ALTER TABLE events
    ADD COLUMN telegram_texts JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE events
    ADD CONSTRAINT chk_events_telegram_texts_object CHECK(jsonb_typeof(telegram_texts) = 'object');

-- +goose Down
ALTER TABLE events
    DROP CONSTRAINT IF EXISTS chk_events_telegram_texts_object;

ALTER TABLE events
    DROP COLUMN IF EXISTS telegram_texts;
