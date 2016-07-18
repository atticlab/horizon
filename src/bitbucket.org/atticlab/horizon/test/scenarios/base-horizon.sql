--
-- PostgreSQL database dump
--

-- Dumped from database version 9.5.0
-- Dumped by pg_dump version 9.5.0

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

SET search_path = public, pg_catalog;

ALTER TABLE IF EXISTS ONLY public.account_traits DROP CONSTRAINT IF EXISTS account_traits_id_fkey;
DROP INDEX IF EXISTS public.trade_effects_by_order_book;
DROP INDEX IF EXISTS public.index_history_transactions_on_id;
DROP INDEX IF EXISTS public.index_history_operations_on_type;
DROP INDEX IF EXISTS public.index_history_operations_on_transaction_id;
DROP INDEX IF EXISTS public.index_history_operations_on_id;
DROP INDEX IF EXISTS public.index_history_ledgers_on_sequence;
DROP INDEX IF EXISTS public.index_history_ledgers_on_previous_ledger_hash;
DROP INDEX IF EXISTS public.index_history_ledgers_on_ledger_hash;
DROP INDEX IF EXISTS public.index_history_ledgers_on_importer_version;
DROP INDEX IF EXISTS public.index_history_ledgers_on_id;
DROP INDEX IF EXISTS public.index_history_ledgers_on_closed_at;
DROP INDEX IF EXISTS public.index_history_effects_on_type;
DROP INDEX IF EXISTS public.index_history_accounts_on_id;
DROP INDEX IF EXISTS public.index_history_accounts_on_address;
DROP INDEX IF EXISTS public.htp_by_htid;
DROP INDEX IF EXISTS public.hs_transaction_by_id;
DROP INDEX IF EXISTS public.hs_ledger_by_id;
DROP INDEX IF EXISTS public.hop_by_hoid;
DROP INDEX IF EXISTS public.hist_tx_p_id;
DROP INDEX IF EXISTS public.hist_op_p_id;
DROP INDEX IF EXISTS public.hist_e_id;
DROP INDEX IF EXISTS public.hist_e_by_order;
DROP INDEX IF EXISTS public.commission_by_hash;
DROP INDEX IF EXISTS public.commission_by_asset;
DROP INDEX IF EXISTS public.commission_by_account_type;
DROP INDEX IF EXISTS public.commission_by_account;
DROP INDEX IF EXISTS public.by_ledger;
DROP INDEX IF EXISTS public.by_hash;
DROP INDEX IF EXISTS public.by_account;
DROP INDEX IF EXISTS public.account_statistics_address_idx;
ALTER TABLE IF EXISTS ONLY public.history_transaction_participants DROP CONSTRAINT IF EXISTS history_transaction_participants_pkey;
ALTER TABLE IF EXISTS ONLY public.history_operation_participants DROP CONSTRAINT IF EXISTS history_operation_participants_pkey;
ALTER TABLE IF EXISTS ONLY public.gorp_migrations DROP CONSTRAINT IF EXISTS gorp_migrations_pkey;
ALTER TABLE IF EXISTS ONLY public.commission DROP CONSTRAINT IF EXISTS commission_pkey;
ALTER TABLE IF EXISTS ONLY public.audit_log DROP CONSTRAINT IF EXISTS audit_log_pkey;
ALTER TABLE IF EXISTS ONLY public.account_traits DROP CONSTRAINT IF EXISTS account_traits_pkey;
ALTER TABLE IF EXISTS ONLY public.account_statistics DROP CONSTRAINT IF EXISTS account_statistics_pkey;
ALTER TABLE IF EXISTS ONLY public.account_limits DROP CONSTRAINT IF EXISTS account_limits_pkey;
ALTER TABLE IF EXISTS public.history_transaction_participants ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.history_operation_participants ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.commission ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.audit_log ALTER COLUMN id DROP DEFAULT;
DROP TABLE IF EXISTS public.history_transactions;
DROP SEQUENCE IF EXISTS public.history_transaction_participants_id_seq;
DROP TABLE IF EXISTS public.history_transaction_participants;
DROP TABLE IF EXISTS public.history_operations;
DROP SEQUENCE IF EXISTS public.history_operation_participants_id_seq;
DROP TABLE IF EXISTS public.history_operation_participants;
DROP TABLE IF EXISTS public.history_ledgers;
DROP TABLE IF EXISTS public.history_effects;
DROP TABLE IF EXISTS public.history_accounts;
DROP TABLE IF EXISTS public.gorp_migrations;
DROP SEQUENCE IF EXISTS public.commission_id_seq;
DROP TABLE IF EXISTS public.commission;
DROP SEQUENCE IF EXISTS public.audit_log_id_seq;
DROP TABLE IF EXISTS public.audit_log;
DROP TABLE IF EXISTS public.account_traits;
DROP TABLE IF EXISTS public.account_statistics;
DROP TABLE IF EXISTS public.account_limits;
DROP EXTENSION IF EXISTS pgcrypto;
DROP EXTENSION IF EXISTS hstore;
DROP EXTENSION IF EXISTS plpgsql;
DROP SCHEMA IF EXISTS public;
--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


--
-- Name: hstore; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA public;


--
-- Name: EXTENSION hstore; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION hstore IS 'data type for storing sets of (key, value) pairs';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


SET search_path = public, pg_catalog;

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: account_limits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_limits (
    address character varying(64) NOT NULL,
    asset_code character varying(12) NOT NULL,
    max_operation_out bigint DEFAULT 0 NOT NULL,
    daily_max_out bigint DEFAULT 0 NOT NULL,
    monthly_max_out bigint DEFAULT 0 NOT NULL,
    max_operation_in bigint DEFAULT '-1'::integer NOT NULL,
    daily_max_in bigint DEFAULT '-1'::integer NOT NULL,
    monthly_max_in bigint DEFAULT '-1'::integer NOT NULL
);


--
-- Name: account_statistics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_statistics (
    address character varying(64) NOT NULL,
    asset_code character varying(12) NOT NULL,
    daily_income bigint DEFAULT 0 NOT NULL,
    daily_outcome bigint DEFAULT 0 NOT NULL,
    weekly_income bigint DEFAULT 0 NOT NULL,
    weekly_outcome bigint DEFAULT 0 NOT NULL,
    monthly_income bigint DEFAULT 0 NOT NULL,
    monthly_outcome bigint DEFAULT 0 NOT NULL,
    annual_income bigint DEFAULT 0 NOT NULL,
    annual_outcome bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    counterparty_type smallint DEFAULT 0 NOT NULL
);


--
-- Name: account_traits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE account_traits (
    id bigint NOT NULL,
    block_incoming_payments boolean DEFAULT false NOT NULL,
    block_outcoming_payments boolean DEFAULT false NOT NULL
);


--
-- Name: audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE audit_log (
    id integer NOT NULL,
    actor character varying(64),
    subject text,
    action text,
    meta text,
    created_at timestamp without time zone DEFAULT timezone('utc'::text, now()) NOT NULL
);


--
-- Name: audit_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE audit_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_log_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE audit_log_id_seq OWNED BY audit_log.id;


--
-- Name: commission; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE commission (
    id bigint NOT NULL,
    key_hash character(64) NOT NULL,
    key_value jsonb NOT NULL,
    flat_fee bigint DEFAULT 0 NOT NULL,
    percent_fee bigint DEFAULT 0 NOT NULL
);


--
-- Name: commission_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE commission_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: commission_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE commission_id_seq OWNED BY commission.id;


--
-- Name: gorp_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE gorp_migrations (
    id text NOT NULL,
    applied_at timestamp with time zone
);


--
-- Name: history_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_accounts (
    id bigint NOT NULL,
    address character varying(64)
);


--
-- Name: history_effects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_effects (
    history_account_id bigint NOT NULL,
    history_operation_id bigint NOT NULL,
    "order" integer NOT NULL,
    type integer NOT NULL,
    details jsonb
);


--
-- Name: history_ledgers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_ledgers (
    sequence integer NOT NULL,
    ledger_hash character varying(64) NOT NULL,
    previous_ledger_hash character varying(64),
    transaction_count integer DEFAULT 0 NOT NULL,
    operation_count integer DEFAULT 0 NOT NULL,
    closed_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    id bigint,
    importer_version integer DEFAULT 1 NOT NULL,
    total_coins bigint NOT NULL,
    fee_pool bigint NOT NULL,
    base_fee integer NOT NULL,
    base_reserve integer NOT NULL,
    max_tx_set_size integer NOT NULL
);


--
-- Name: history_operation_participants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_operation_participants (
    id integer NOT NULL,
    history_operation_id bigint NOT NULL,
    history_account_id bigint NOT NULL
);


--
-- Name: history_operation_participants_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE history_operation_participants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: history_operation_participants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE history_operation_participants_id_seq OWNED BY history_operation_participants.id;


--
-- Name: history_operations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_operations (
    id bigint NOT NULL,
    transaction_id bigint NOT NULL,
    application_order integer NOT NULL,
    type integer NOT NULL,
    details jsonb,
    source_account character varying(64) DEFAULT ''::character varying NOT NULL
);


--
-- Name: history_transaction_participants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_transaction_participants (
    id integer NOT NULL,
    history_transaction_id bigint NOT NULL,
    history_account_id bigint NOT NULL
);


--
-- Name: history_transaction_participants_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE history_transaction_participants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: history_transaction_participants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE history_transaction_participants_id_seq OWNED BY history_transaction_participants.id;


--
-- Name: history_transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_transactions (
    transaction_hash character varying(64) NOT NULL,
    ledger_sequence integer NOT NULL,
    application_order integer NOT NULL,
    account character varying(64) NOT NULL,
    account_sequence bigint NOT NULL,
    fee_paid integer NOT NULL,
    operation_count integer NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    id bigint,
    tx_envelope text NOT NULL,
    tx_result text NOT NULL,
    tx_meta text NOT NULL,
    tx_fee_meta text NOT NULL,
    signatures character varying(96)[] DEFAULT '{}'::character varying[] NOT NULL,
    memo_type character varying DEFAULT 'none'::character varying NOT NULL,
    memo character varying,
    time_bounds int8range
);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY audit_log ALTER COLUMN id SET DEFAULT nextval('audit_log_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY commission ALTER COLUMN id SET DEFAULT nextval('commission_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_operation_participants ALTER COLUMN id SET DEFAULT nextval('history_operation_participants_id_seq'::regclass);


--
-- Name: id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_transaction_participants ALTER COLUMN id SET DEFAULT nextval('history_transaction_participants_id_seq'::regclass);


--
-- Data for Name: account_limits; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: account_statistics; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: account_traits; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO account_traits VALUES (1, false, false);
INSERT INTO account_traits VALUES (2, false, false);


--
-- Data for Name: audit_log; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: audit_log_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('audit_log_id_seq', 1, false);


--
-- Data for Name: commission; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: commission_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('commission_id_seq', 1, false);


--
-- Data for Name: gorp_migrations; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO gorp_migrations VALUES ('1_initial_schema.sql', '2016-07-11 16:56:49.46066+03');
INSERT INTO gorp_migrations VALUES ('2_index_participants_by_toid.sql', '2016-07-11 16:56:49.539775+03');
INSERT INTO gorp_migrations VALUES ('3_aggregate_expenses_for_accounts.sql', '2016-07-11 16:56:49.617108+03');
INSERT INTO gorp_migrations VALUES ('4_account_statistics_updated_at_timezone.sql', '2016-07-11 16:56:49.703979+03');
INSERT INTO gorp_migrations VALUES ('5_account_statistics_account_type.sql', '2016-07-11 16:56:49.861262+03');
INSERT INTO gorp_migrations VALUES ('6_account_traits.sql', '2016-07-11 16:56:49.972082+03');
INSERT INTO gorp_migrations VALUES ('7_account_limits.sql', '2016-07-11 16:56:50.01322+03');
INSERT INTO gorp_migrations VALUES ('8_account_limits_two_way.sql', '2016-07-11 16:56:50.077507+03');
INSERT INTO gorp_migrations VALUES ('9_commission.sql', '2016-07-11 16:56:50.31699+03');


--
-- Data for Name: history_accounts; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_accounts VALUES (1, 'GAJLXJ6AJBYG5IDQZQ45CTDYHJRZ6DI4H4IRJA6CD3W6IIJIKLPAS33R');
INSERT INTO history_accounts VALUES (2, 'GB45Q4BIHV52PK34LB56KKU4YDELVB3NGIZECZ4257H5DDSCUXC6ADGI');


--
-- Data for Name: history_effects; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: history_ledgers; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_ledgers VALUES (1, '0009617b7db56ad0f0fb977c6930e9f6a3bacd489d184457e8fd7ac05f6a2915', NULL, 0, 0, '1970-01-01 00:00:00', '2016-07-11 13:58:34.896634', '2016-07-11 13:58:34.896634', 4294967296, 8, 0, 0, 0, 0, 100);
INSERT INTO history_ledgers VALUES (2, '8f82ea129e775ce58bedae88dbcc03f609f8a75a04b22996b282c93428a1d42f', '0009617b7db56ad0f0fb977c6930e9f6a3bacd489d184457e8fd7ac05f6a2915', 0, 0, '2016-07-11 13:58:28', '2016-07-11 13:58:34.979864', '2016-07-11 13:58:34.979864', 8589934592, 8, 0, 0, 0, 0, 50);
INSERT INTO history_ledgers VALUES (3, '342b37c6bc3fef074065558cfaab377682604099212e884ff1b30575965e3d28', '8f82ea129e775ce58bedae88dbcc03f609f8a75a04b22996b282c93428a1d42f', 0, 0, '2016-07-11 13:58:33', '2016-07-11 13:58:35.003435', '2016-07-11 13:58:35.003435', 12884901888, 8, 0, 0, 0, 0, 50);
INSERT INTO history_ledgers VALUES (4, '9d6778f2d9337c8093057f7df83d13d52d08cfe0ba33e116296f04d6e93f19d6', '342b37c6bc3fef074065558cfaab377682604099212e884ff1b30575965e3d28', 0, 0, '2016-07-11 13:58:38', '2016-07-11 13:58:38.876289', '2016-07-11 13:58:38.876289', 17179869184, 8, 0, 0, 0, 0, 50);


--
-- Data for Name: history_operation_participants; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: history_operation_participants_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_operation_participants_id_seq', 1, false);


--
-- Data for Name: history_operations; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: history_transaction_participants; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: history_transaction_participants_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_transaction_participants_id_seq', 1, false);


--
-- Data for Name: history_transactions; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: account_limits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_limits
    ADD CONSTRAINT account_limits_pkey PRIMARY KEY (address, asset_code);


--
-- Name: account_statistics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_statistics
    ADD CONSTRAINT account_statistics_pkey PRIMARY KEY (address, asset_code, counterparty_type);


--
-- Name: account_traits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_traits
    ADD CONSTRAINT account_traits_pkey PRIMARY KEY (id);


--
-- Name: audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: commission_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY commission
    ADD CONSTRAINT commission_pkey PRIMARY KEY (id);


--
-- Name: gorp_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY gorp_migrations
    ADD CONSTRAINT gorp_migrations_pkey PRIMARY KEY (id);


--
-- Name: history_operation_participants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_operation_participants
    ADD CONSTRAINT history_operation_participants_pkey PRIMARY KEY (id);


--
-- Name: history_transaction_participants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_transaction_participants
    ADD CONSTRAINT history_transaction_participants_pkey PRIMARY KEY (id);


--
-- Name: account_statistics_address_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX account_statistics_address_idx ON account_statistics USING btree (address);


--
-- Name: by_account; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX by_account ON history_transactions USING btree (account, account_sequence);


--
-- Name: by_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX by_hash ON history_transactions USING btree (transaction_hash);


--
-- Name: by_ledger; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX by_ledger ON history_transactions USING btree (ledger_sequence, application_order);


--
-- Name: commission_by_account; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX commission_by_account ON commission USING btree (((key_value ->> 'from'::text)), ((key_value ->> 'to'::text)));


--
-- Name: commission_by_account_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX commission_by_account_type ON commission USING btree ((((key_value ->> 'from_type'::text))::integer), (((key_value ->> 'to_type'::text))::integer));


--
-- Name: commission_by_asset; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX commission_by_asset ON commission USING btree (((key_value ->> 'asset_type'::text)), ((key_value ->> 'asset_code'::text)), ((key_value ->> 'asset_issuer'::text)));


--
-- Name: commission_by_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX commission_by_hash ON commission USING btree (key_hash);


--
-- Name: hist_e_by_order; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_e_by_order ON history_effects USING btree (history_operation_id, "order");


--
-- Name: hist_e_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_e_id ON history_effects USING btree (history_account_id, history_operation_id, "order");


--
-- Name: hist_op_p_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_op_p_id ON history_operation_participants USING btree (history_account_id, history_operation_id);


--
-- Name: hist_tx_p_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_tx_p_id ON history_transaction_participants USING btree (history_account_id, history_transaction_id);


--
-- Name: hop_by_hoid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX hop_by_hoid ON history_operation_participants USING btree (history_operation_id);


--
-- Name: hs_ledger_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hs_ledger_by_id ON history_ledgers USING btree (id);


--
-- Name: hs_transaction_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hs_transaction_by_id ON history_transactions USING btree (id);


--
-- Name: htp_by_htid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX htp_by_htid ON history_transaction_participants USING btree (history_transaction_id);


--
-- Name: index_history_accounts_on_address; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_accounts_on_address ON history_accounts USING btree (address);


--
-- Name: index_history_accounts_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_accounts_on_id ON history_accounts USING btree (id);


--
-- Name: index_history_effects_on_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_effects_on_type ON history_effects USING btree (type);


--
-- Name: index_history_ledgers_on_closed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_ledgers_on_closed_at ON history_ledgers USING btree (closed_at);


--
-- Name: index_history_ledgers_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_id ON history_ledgers USING btree (id);


--
-- Name: index_history_ledgers_on_importer_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_ledgers_on_importer_version ON history_ledgers USING btree (importer_version);


--
-- Name: index_history_ledgers_on_ledger_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_ledger_hash ON history_ledgers USING btree (ledger_hash);


--
-- Name: index_history_ledgers_on_previous_ledger_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_previous_ledger_hash ON history_ledgers USING btree (previous_ledger_hash);


--
-- Name: index_history_ledgers_on_sequence; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_sequence ON history_ledgers USING btree (sequence);


--
-- Name: index_history_operations_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_operations_on_id ON history_operations USING btree (id);


--
-- Name: index_history_operations_on_transaction_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_operations_on_transaction_id ON history_operations USING btree (transaction_id);


--
-- Name: index_history_operations_on_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_operations_on_type ON history_operations USING btree (type);


--
-- Name: index_history_transactions_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_transactions_on_id ON history_transactions USING btree (id);


--
-- Name: trade_effects_by_order_book; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX trade_effects_by_order_book ON history_effects USING btree (((details ->> 'sold_asset_type'::text)), ((details ->> 'sold_asset_code'::text)), ((details ->> 'sold_asset_issuer'::text)), ((details ->> 'bought_asset_type'::text)), ((details ->> 'bought_asset_code'::text)), ((details ->> 'bought_asset_issuer'::text))) WHERE (type = 33);


--
-- Name: account_traits_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY account_traits
    ADD CONSTRAINT account_traits_id_fkey FOREIGN KEY (id) REFERENCES history_accounts(id);


--
-- PostgreSQL database dump complete
--

