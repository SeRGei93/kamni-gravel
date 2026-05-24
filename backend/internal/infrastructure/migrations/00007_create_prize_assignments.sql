-- +goose Up
CREATE TABLE prize_assignments (
    id SERIAL PRIMARY KEY,
    participant_id INTEGER NOT NULL,
    gift_id INTEGER NOT NULL,
    comment TEXT,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (participant_id) REFERENCES participants(id) ON DELETE CASCADE,
    FOREIGN KEY (gift_id) REFERENCES gifts(id) ON DELETE CASCADE
);

CREATE INDEX idx_prize_assignments_participant ON prize_assignments(participant_id);
CREATE INDEX idx_prize_assignments_gift ON prize_assignments(gift_id);

-- +goose Down
DROP TABLE prize_assignments;
