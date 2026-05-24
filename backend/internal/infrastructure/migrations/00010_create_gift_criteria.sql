-- +goose Up
CREATE TABLE entity_criteria (
    id SERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL CHECK(entity_type IN ('gift', 'result')),
    entity_id INTEGER NOT NULL,
    criteria_id INTEGER NOT NULL,
    FOREIGN KEY (criteria_id) REFERENCES criteria(id) ON DELETE CASCADE,
    UNIQUE(entity_type, entity_id, criteria_id)
);

CREATE INDEX idx_entity_criteria_entity ON entity_criteria(entity_type, entity_id);
CREATE INDEX idx_entity_criteria_criteria ON entity_criteria(criteria_id);

-- +goose Down
DROP TABLE IF EXISTS entity_criteria;
-- Восстанавливаем старую таблицу для совместимости с downgrade
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
