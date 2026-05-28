-- +goose Up
CREATE TABLE gift_place_rules (
    gift_id INTEGER PRIMARY KEY REFERENCES gifts(id) ON DELETE CASCADE,
    rule_type VARCHAR(20) NOT NULL CHECK(rule_type IN ('places', 'last_n')),
    last_count INTEGER CHECK(last_count IS NULL OR last_count > 0),
    CHECK(
        (rule_type = 'places' AND last_count IS NULL)
        OR
        (rule_type = 'last_n' AND last_count IS NOT NULL)
    )
);

CREATE TABLE gift_place_rule_places (
    gift_id INTEGER NOT NULL REFERENCES gift_place_rules(gift_id) ON DELETE CASCADE,
    place INTEGER NOT NULL CHECK(place > 0),
    PRIMARY KEY (gift_id, place)
);

CREATE INDEX idx_gift_place_rules_gift ON gift_place_rules(gift_id);
CREATE INDEX idx_gift_place_rule_places_gift ON gift_place_rule_places(gift_id);

INSERT INTO gift_place_rules (gift_id, rule_type, last_count)
SELECT id, 'places', NULL
FROM gifts
WHERE place IS NOT NULL
ON CONFLICT (gift_id) DO NOTHING;

INSERT INTO gift_place_rule_places (gift_id, place)
SELECT id, place
FROM gifts
WHERE place IS NOT NULL
ON CONFLICT (gift_id, place) DO NOTHING;

-- +goose Down
DROP TABLE gift_place_rule_places;
DROP TABLE gift_place_rules;
