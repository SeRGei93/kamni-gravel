-- +goose Up
-- Фикс дрейфа +3ч: метки, вводимые администратором в минской зоне (+03:00),
-- писались в TIMESTAMP без зоны (офсет отбрасывался, хранилось минское
-- wall-время), а при чтении время трактовалось как UTC — каждый цикл
-- «открыл форму → сохранил» сдвигал значения на +3 часа.
-- Конвертируем в TIMESTAMPTZ, трактуя сохранённое wall-время как минское:
-- уже введённые значения при этом не меняются (что вводили — то и останется).
-- Серверные метки (created_at, submitted_at, registered_at и т.п.) пишутся и
-- читаются как UTC согласованно — их не трогаем.
ALTER TABLE results
    ALTER COLUMN started_at TYPE TIMESTAMPTZ USING started_at AT TIME ZONE 'Europe/Minsk',
    ALTER COLUMN finished_at TYPE TIMESTAMPTZ USING finished_at AT TIME ZONE 'Europe/Minsk';

ALTER TABLE events
    ALTER COLUMN start_date TYPE TIMESTAMPTZ USING start_date AT TIME ZONE 'Europe/Minsk',
    ALTER COLUMN end_date TYPE TIMESTAMPTZ USING end_date AT TIME ZONE 'Europe/Minsk';

-- +goose Down
ALTER TABLE results
    ALTER COLUMN started_at TYPE TIMESTAMP USING started_at AT TIME ZONE 'Europe/Minsk',
    ALTER COLUMN finished_at TYPE TIMESTAMP USING finished_at AT TIME ZONE 'Europe/Minsk';

ALTER TABLE events
    ALTER COLUMN start_date TYPE TIMESTAMP USING start_date AT TIME ZONE 'Europe/Minsk',
    ALTER COLUMN end_date TYPE TIMESTAMP USING end_date AT TIME ZONE 'Europe/Minsk';
