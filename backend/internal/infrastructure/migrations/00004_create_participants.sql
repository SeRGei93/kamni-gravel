-- +goose Up
CREATE TABLE participants (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    event_id INTEGER NOT NULL,
    bike_type VARCHAR(50) NOT NULL CHECK(bike_type IN ('gravel', 'mtb', 'road', 'single_speed', 'tandem')),
    gender VARCHAR(50) NOT NULL CHECK(gender IN ('male', 'female')),
    notes TEXT,
    registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    UNIQUE(user_id, event_id)
);

CREATE INDEX idx_participants_event ON participants(event_id);
CREATE INDEX idx_participants_user ON participants(user_id);

-- +goose Down
DROP TABLE participants;
