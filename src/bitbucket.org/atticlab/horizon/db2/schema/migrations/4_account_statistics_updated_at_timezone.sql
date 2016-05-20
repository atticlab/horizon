-- +migrate Up

ALTER TABLE account_statistics ALTER COLUMN  updated_at TYPE timestamp with time zone;

-- +migrate Down

ALTER TABLE account_statistics ALTER COLUMN  updated_at TYPE timestamp without time zone;
