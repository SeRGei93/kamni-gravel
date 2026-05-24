-- +goose Up
CREATE TABLE users (
    id BIGINT PRIMARY KEY NOT NULL,  -- Telegram ID (не autoincrement)
    username VARCHAR(255) DEFAULT '',
    first_name VARCHAR(255) DEFAULT '',
    last_name VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);

-- +goose Down
DROP TABLE users;
