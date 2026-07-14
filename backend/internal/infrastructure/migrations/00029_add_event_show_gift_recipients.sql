-- +goose Up
-- По умолчанию имена получателей скрыты, пока администратор явно не включит
-- их показ на вкладке Mini App «Призы от меня».
ALTER TABLE events
    ADD COLUMN show_gift_recipients BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE events
    DROP COLUMN IF EXISTS show_gift_recipients;
