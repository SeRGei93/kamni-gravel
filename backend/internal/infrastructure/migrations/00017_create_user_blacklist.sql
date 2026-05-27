-- +goose Up
CREATE TABLE user_blacklist (
    telegram_user_id BIGINT PRIMARY KEY,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_blacklist_created_at ON user_blacklist(created_at);

-- +goose Down
DROP TABLE user_blacklist;
