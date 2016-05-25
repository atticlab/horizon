-- +migrate Up

ALTER TABLE account_statistics ADD COLUMN  counterparty_type smallint NOT NULL DEFAULT 0;
ALTER TABLE account_statistics DROP CONSTRAINT account_statistics_pkey;
ALTER TABLE account_statistics ADD PRIMARY KEY(address, asset_code, counterparty_type);


-- +migrate Down
ALTER TABLE account_statistics DROP CONSTRAINT account_statistics_pkey;
ALTER TABLE account_statistics DROP COLUMN counterparty_type;
ALTER TABLE account_statistics ADD PRIMARY KEY(address, asset_code);
