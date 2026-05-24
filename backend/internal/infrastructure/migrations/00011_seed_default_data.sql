-- +goose Up
-- Создаём дефолтного админа (пароль: admin123, нужно будет сменить)
-- bcrypt hash для "admin123"
-- Используем ON CONFLICT для безопасности при повторном применении
INSERT INTO admin_users (username, password_hash, role) 
VALUES ('admin', '$2a$10$mXnxTm184PlVYrk37iqLBek8fK6wBh2Z2tF4zCRJb3Isof007T91K', 'admin')
ON CONFLICT (username) DO NOTHING;

-- Создаём дефолтное событие
-- Используем ON CONFLICT для безопасности при повторном применении
INSERT INTO events (name, description, active, start_date, end_date) 
VALUES (
    'kamni_200', 
    'КАМНИ 200 - гравийная гонка на 200 км',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP + INTERVAL '14 days'
)
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DELETE FROM events WHERE name = 'kamni_200';
DELETE FROM admin_users WHERE username = 'admin';
