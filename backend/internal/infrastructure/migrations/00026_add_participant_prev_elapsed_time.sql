-- +goose Up
-- Ручное «время прошлого года» (сек). Приоритет над автоматическим значением,
-- вычисляемым по результату участника на ближайшем предыдущем событии.
ALTER TABLE participants
    ADD COLUMN prev_elapsed_time_sec INTEGER NULL CHECK (prev_elapsed_time_sec > 0);

-- +goose Down
ALTER TABLE participants DROP COLUMN IF EXISTS prev_elapsed_time_sec;
