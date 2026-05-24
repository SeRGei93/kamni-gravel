-- +goose Up
CREATE TABLE criteria (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    criteria_type VARCHAR(50) NOT NULL CHECK(criteria_type IN ('speed', 'photo', 'beer', 'custom')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Предзаполним базовые критерии для ручных номинаций
INSERT INTO criteria (name, description, criteria_type) VALUES
    ('Максимальная скорость', 'Самая высокая зафиксированная скорость', 'speed'),
    ('Больше всего пива', 'Выпил больше всего пива на маршруте', 'beer'),
    ('Лучшее фото', 'Лучшая фотография на маршруте', 'photo'),
    ('Фото у знака', 'Фотография у определенного знака/места', 'photo'),
    ('Семейная пара', 'Номинация для семейной пары', 'custom'),
    ('Командный зачет', 'Номинация для команды', 'custom'),
    ('Самый веселый', 'За позитив и настроение', 'custom'),
    ('Самый стильный', 'За стиль и образ', 'custom');

CREATE INDEX idx_criteria_type ON criteria(criteria_type);

-- +goose Down
DROP TABLE criteria;
