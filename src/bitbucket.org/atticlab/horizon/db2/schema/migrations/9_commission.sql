-- +migrate Up

CREATE TABLE commission (
	id          serial,
    key_hash    character(64) NOT NULL,
    key_value 	jsonb NOT NULL,
    flat_fee 	bigint NOT NULL DEFAULT 0,
    percent_fee bigint NOT NULL DEFAULT 0,
    PRIMARY KEY(id)
);

CREATE INDEX commission_by_hash ON commission USING btree (key_hash);

-- +migrate Down

DROP TABLE commission;
