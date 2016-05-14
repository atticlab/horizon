-- +migrate Up

ALTER TABLE history_accounts ADD daily_income 		decimal NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD daily_outcome 		decimal NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD weekly_income 		decimal	NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD weekly_outcome		decimal NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD monthly_income		decimal NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD monthly_outcome 	decimal NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD annual_income		decimal NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD annual_outcome		decimal NOT NULL DEFAULT 0;
ALTER TABLE history_accounts ADD updated_at 		timestamp without time zone;

-- +migrate Down

ALTER TABLE history_accounts DROP	daily_income;
ALTER TABLE history_accounts DROP 	daily_outcome;
ALTER TABLE history_accounts DROP 	weekly_income;
ALTER TABLE history_accounts DROP 	weekly_outcome;
ALTER TABLE history_accounts DROP 	monthly_income;
ALTER TABLE history_accounts DROP 	monthly_outcome;
ALTER TABLE history_accounts DROP 	annual_income;
ALTER TABLE history_accounts DROP 	annual_outcome;
ALTER TABLE history_accounts DROP   updated_at;
