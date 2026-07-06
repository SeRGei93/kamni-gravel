-- +goose Up
-- Флаги ручной остановки приёма результатов и призов. Событие целиком
-- завершается по end_date, а эти флаги позволяют закрыть приём раньше
-- и независимо друг от друга.
ALTER TABLE events
    ADD COLUMN stop_results BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN stop_gifts BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE events
    DROP COLUMN IF EXISTS stop_results,
    DROP COLUMN IF EXISTS stop_gifts;
