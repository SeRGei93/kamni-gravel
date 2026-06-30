-- +goose Up
-- Статус участия в зачёте: active (по умолчанию), dnf (сошёл с дистанции),
-- disqualified (дисквалификация). DNF и disqualified исключаются из зачёта и
-- призов по местам; disqualified — также из призов по критериям.
ALTER TABLE participants
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'dnf', 'disqualified'));

CREATE INDEX idx_participants_event_status ON participants(event_id, status);

-- +goose Down
DROP INDEX IF EXISTS idx_participants_event_status;
ALTER TABLE participants DROP COLUMN IF EXISTS status;
