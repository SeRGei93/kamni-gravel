-- +goose Up
ALTER TABLE criteria DROP CONSTRAINT IF EXISTS criteria_criteria_type_check;

ALTER TABLE criteria
    ADD CONSTRAINT criteria_criteria_type_check
    CHECK (criteria_type IN ('speed', 'photo', 'beer', 'random', 'custom'));

-- +goose Down
ALTER TABLE criteria DROP CONSTRAINT IF EXISTS criteria_criteria_type_check;

UPDATE criteria
SET criteria_type = 'custom'
WHERE criteria_type = 'random';

ALTER TABLE criteria
    ADD CONSTRAINT criteria_criteria_type_check
    CHECK (criteria_type IN ('speed', 'photo', 'beer', 'custom'));
