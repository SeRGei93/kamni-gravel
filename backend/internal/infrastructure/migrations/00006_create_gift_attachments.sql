-- +goose Up
CREATE TABLE gift_attachments (
    id SERIAL PRIMARY KEY,
    gift_id INTEGER NOT NULL,
    telegram_file_id TEXT NOT NULL,
    file_type VARCHAR(50) NOT NULL CHECK(file_type IN ('photo', 'document')),
    FOREIGN KEY (gift_id) REFERENCES gifts(id) ON DELETE CASCADE
);

CREATE INDEX idx_gift_attachments_gift ON gift_attachments(gift_id);

-- +goose Down
DROP TABLE gift_attachments;
