-- +goose Up
-- Актуальный ростер публичного чата. Строка существует, только пока участник в
-- чате: бот делает upsert при входе/повышении и удаляет строку при выходе/кике
-- (по Telegram chat_member апдейтам). Первичное заполнение — импортом CSV из
-- админки. Истории выхода не храним.
CREATE TABLE chat_members (
    telegram_user_id BIGINT PRIMARY KEY,
    username TEXT NOT NULL DEFAULT '',
    first_name TEXT NOT NULL DEFAULT '',
    last_name TEXT NOT NULL DEFAULT '',
    is_bot BOOLEAN NOT NULL DEFAULT false,
    is_admin BOOLEAN NOT NULL DEFAULT false,
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE chat_members;
