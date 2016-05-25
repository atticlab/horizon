-- +migrate Up

CREATE TABLE account_traits (
    id bigint PRIMARY KEY REFERENCES history_accounts(id),
    block_incoming_payments boolean NOT NULL DEFAULT FALSE,
    block_outcoming_payments boolean NOT NULL DEFAULT FALSE
);

CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    invocer character varying(64),
    type text,
    subject text,
    meta text,
    created_at timestamp without time zone NOT NULL default (now() at time zone 'utc')
);

-- +migrate Down

DROP TABLE account_blacklist;
DROP TABLE audit_log;
