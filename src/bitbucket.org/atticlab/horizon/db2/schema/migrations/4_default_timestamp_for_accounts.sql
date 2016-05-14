-- +migrate Up

ALTER TABLE history_accounts ALTER COLUMN updated_at SET DEFAULT (now() at time zone 'utc');

-- +migrate Down

ALTER TABLE history_accounts ALTER COLUMN updated_at DROP DEFAULT;
