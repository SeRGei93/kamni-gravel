-- +goose Up
-- Создаём новую таблицу entity_criteria, если база ещё на старой схеме.
CREATE TABLE IF NOT EXISTS entity_criteria (
    id SERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL CHECK(entity_type IN ('gift', 'result')),
    entity_id INTEGER NOT NULL,
    criteria_id INTEGER NOT NULL,
    FOREIGN KEY (criteria_id) REFERENCES criteria(id) ON DELETE CASCADE,
    UNIQUE(entity_type, entity_id, criteria_id)
);

CREATE INDEX IF NOT EXISTS idx_entity_criteria_entity ON entity_criteria(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_criteria_criteria ON entity_criteria(criteria_id);

-- Мигрируем данные из gift_criteria в entity_criteria только для баз со старой таблицей.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public.gift_criteria') IS NOT NULL THEN
        INSERT INTO entity_criteria (entity_type, entity_id, criteria_id)
        SELECT 'gift', gift_id, criteria_id
        FROM gift_criteria
        ON CONFLICT (entity_type, entity_id, criteria_id) DO NOTHING;
    END IF;
END $$;
-- +goose StatementEnd

-- Удаляем старую таблицу
DROP TABLE IF EXISTS gift_criteria;

-- +goose Down
-- Восстанавливаем gift_criteria
CREATE TABLE gift_criteria (
    id SERIAL PRIMARY KEY,
    gift_id INTEGER NOT NULL,
    criteria_id INTEGER NOT NULL,
    FOREIGN KEY (gift_id) REFERENCES gifts(id) ON DELETE CASCADE,
    FOREIGN KEY (criteria_id) REFERENCES criteria(id) ON DELETE CASCADE,
    UNIQUE(gift_id, criteria_id)
);

CREATE INDEX idx_gift_criteria_gift ON gift_criteria(gift_id);
CREATE INDEX idx_gift_criteria_criteria ON gift_criteria(criteria_id);

-- Мигрируем данные обратно
INSERT INTO gift_criteria (gift_id, criteria_id)
SELECT entity_id, criteria_id
FROM entity_criteria
WHERE entity_type = 'gift';

-- Удаляем entity_criteria
DROP TABLE entity_criteria;
