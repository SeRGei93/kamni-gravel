-- +goose Up
ALTER TABLE results
    ADD COLUMN started_at      TIMESTAMP,
    ADD COLUMN finished_at     TIMESTAMP,
    ADD COLUMN distance_meters INTEGER,
    ADD COLUMN avg_heart_rate  INTEGER,
    ADD COLUMN max_heart_rate  INTEGER,
    ADD COLUMN peak_speed_kmh  DOUBLE PRECISION,
    ADD COLUMN avg_cadence     INTEGER,
    ADD COLUMN calories        INTEGER;

-- +goose Down
ALTER TABLE results
    DROP COLUMN IF EXISTS started_at,
    DROP COLUMN IF EXISTS finished_at,
    DROP COLUMN IF EXISTS distance_meters,
    DROP COLUMN IF EXISTS avg_heart_rate,
    DROP COLUMN IF EXISTS max_heart_rate,
    DROP COLUMN IF EXISTS peak_speed_kmh,
    DROP COLUMN IF EXISTS avg_cadence,
    DROP COLUMN IF EXISTS calories;
