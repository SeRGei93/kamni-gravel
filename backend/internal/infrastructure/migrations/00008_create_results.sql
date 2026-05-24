-- +goose Up
CREATE TABLE results (
    id SERIAL PRIMARY KEY,
    participant_id INTEGER NOT NULL,
    result_link TEXT,
    elapsed_time_sec INTEGER,
    moving_time_sec INTEGER,
    is_current BOOLEAN DEFAULT true,
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (participant_id) REFERENCES participants(id) ON DELETE CASCADE
);

CREATE INDEX idx_results_participant ON results(participant_id);
CREATE INDEX idx_results_elapsed_time ON results(elapsed_time_sec);
CREATE INDEX idx_results_moving_time ON results(moving_time_sec);
CREATE INDEX idx_results_current ON results(participant_id, is_current) WHERE is_current = true;

-- +goose Down
DROP TABLE results;
