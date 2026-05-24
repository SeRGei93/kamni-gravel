-- +goose Up
CREATE TABLE gifts (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    event_id INTEGER NOT NULL,
    description TEXT NOT NULL,
    gender_filter VARCHAR(50) CHECK(gender_filter IN ('all', 'male', 'female')),
    bike_type_filter VARCHAR(50) CHECK(bike_type_filter IN ('all', 'gravel', 'mtb', 'road', 'single_speed', 'tandem')),
    place INTEGER CHECK(place > 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
);

CREATE INDEX idx_gifts_user ON gifts(user_id);
CREATE INDEX idx_gifts_event ON gifts(event_id);
CREATE INDEX idx_gifts_place ON gifts(place);
CREATE INDEX idx_gifts_filters ON gifts(gender_filter, bike_type_filter);

-- +goose Down
DROP TABLE gifts;
