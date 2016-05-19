-- +migrate Up

CREATE TABLE account_statistics (
    address             character varying(64) NOT NULL,
    asset_code          character varying(12) NOT NULL,   
    daily_income 		decimal NOT NULL DEFAULT 0,
    daily_outcome 		decimal NOT NULL DEFAULT 0,
    weekly_income 		decimal	NOT NULL DEFAULT 0,
    weekly_outcome		decimal NOT NULL DEFAULT 0,
    monthly_income		decimal NOT NULL DEFAULT 0,
    monthly_outcome 	decimal NOT NULL DEFAULT 0,
    annual_income		decimal NOT NULL DEFAULT 0,
    annual_outcome		decimal NOT NULL DEFAULT 0,
    updated_at 		    timestamp without time zone NOT NULL,
    PRIMARY KEY(address, asset_code)
);

CREATE INDEX account_statistics_address_idx ON account_statistics (address);

-- +migrate Down

DROP INDEX account_statistics_address_idx;
DROP TABLE account_statistics;
