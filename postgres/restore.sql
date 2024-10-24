--
-- NOTE:
--
-- File paths need to be edited. Search for $$PATH$$ and
-- replace it with the path to the directory containing
-- the extracted data files.
--
--
-- PostgreSQL database dump
--

-- Dumped from database version 12.11 (Ubuntu 12.11-0ubuntu0.20.04.1)
-- Dumped by pg_dump version 12.11 (Ubuntu 12.11-0ubuntu0.20.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

-- DROP DATABASE mydatabase;
--
-- Name: postgres; Type: DATABASE; Schema: -; Owner: postgres
--

-- CREATE DATABASE mydatabase WITH TEMPLATE = template0 ENCODING = 'UTF8' LC_COLLATE = 'en_US.UTF-8' LC_CTYPE = 'en_US.UTF-8';


ALTER DATABASE mydatabase OWNER TO myuser;

\connect mydatabase

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: DATABASE postgres; Type: COMMENT; Schema: -; Owner: postgres
--

COMMENT ON DATABASE mydatabase IS 'default administrative connection database';


--
-- Name: CompletionStatus; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."CompletionStatus" AS ENUM (
    'UNPROCESSED',
    'ACCEPTED',
    'REJECTED',
    'EXCEPTION',
    'MACHINE_HALTED',
    'CYCLE_LIMIT_EXCEEDED',
    'TIME_LIMIT_EXCEEDED',
    'PAYLOAD_LENGTH_LIMIT_EXCEEDED'
);


ALTER TYPE public."CompletionStatus" OWNER TO myuser;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: inputs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.inputs (
    index integer NOT NULL,
    blob bytea NOT NULL,
    status public."CompletionStatus" NOT NULL
);


ALTER TABLE public.inputs OWNER TO myuser;

INSERT INTO public.inputs (index, blob, status) VALUES (1, '\x415bf3630000000000000000000000000000000000000000000000000000000000000001000000000000000000000000ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000004d20000000000000000000000000000000000000000000000000000000000000015000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000a1122334455667788990000000000000000000000000000000000000000000000', 'ACCEPTED');
INSERT INTO public.inputs (index, blob, status) VALUES (2, '\x415bf3630000000000000000000000000000000000000000000000000000000000000001000000000000000000000000ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000004d20000000000000000000000000000000000000000000000000000000000000015000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000a1122334455667788990000000000000000000000000000000000000000000000', 'ACCEPTED');


--
-- Name: outputs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.outputs (
    input_index integer NOT NULL,
    index integer NOT NULL,
    blob bytea NOT NULL
);


ALTER TABLE public.outputs OWNER TO myuser;

--
-- Name: proofs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.proofs (
    input_index integer NOT NULL,
    output_index integer NOT NULL,
    first_input integer NOT NULL,
    last_input integer NOT NULL,
    validity_input_index_within_epoch integer NOT NULL,
    validity_output_index_within_input integer NOT NULL,
    validity_output_hashes_root_hash bytea NOT NULL,
    validity_output_epoch_root_hash bytea NOT NULL,
    validity_machine_state_hash bytea NOT NULL,
    validity_output_hash_in_output_hashes_siblings bytea[] NOT NULL,
    validity_output_hashes_in_epoch_siblings bytea[] NOT NULL
);


ALTER TABLE public.proofs OWNER TO myuser;

--
-- Name: reports; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.reports (
    input_index integer NOT NULL,
    index integer NOT NULL,
    blob bytea NOT NULL
);


ALTER TABLE public.reports OWNER TO myuser;

INSERT INTO public.reports VALUES (1, 1, '\x415bf3630000000000000000000000000000000000000000000000000000000000000001000000000000000000000000ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000004d20000000000000000000000000000000000000000000000000000000000000015000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000a1122334455667788990000000000000000000000000000000000000000000000');
INSERT INTO public.reports VALUES (2, 2, '\x415bf3630000000000000000000000000000000000000000000000000000000000000001000000000000000000000000ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000004d20000000000000000000000000000000000000000000000000000000000000015000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000a1122334455667788990000000000000000000000000000000000000000000000');


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO myuser;

--
-- Data for Name: inputs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.inputs (index, blob, status) FROM stdin;
\.
--COPY public.inputs (index, blob, status) FROM '/docker-entrypoint-initdb.d/data_files/3118.dat';

--
-- Data for Name: outputs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.outputs (input_index, index, blob) FROM stdin;
\.
--COPY public.outputs (input_index, index, blob) FROM '/docker-entrypoint-initdb.d/data_files/3119.dat';

--
-- Data for Name: proofs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.proofs (input_index, output_index, first_input, last_input, validity_input_index_within_epoch, validity_output_index_within_input, validity_output_hashes_root_hash, validity_output_epoch_root_hash, validity_machine_state_hash, validity_output_hash_in_output_hashes_siblings, validity_output_hashes_in_epoch_siblings) FROM stdin;
\.
--COPY public.proofs (input_index, output_index, first_input, last_input, validity_input_index_within_epoch, validity_output_index_within_input, validity_output_hashes_root_hash, validity_output_epoch_root_hash, validity_machine_state_hash, validity_output_hash_in_output_hashes_siblings, validity_output_hashes_in_epoch_siblings) FROM '/docker-entrypoint-initdb.d/data_files/3121.dat';

--
-- Data for Name: reports; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.reports (input_index, index, blob) FROM stdin;
\.
--COPY public.reports (input_index, index, blob) FROM '/docker-entrypoint-initdb.d/data_files/3120.dat';

--
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.schema_migrations (version, dirty) FROM stdin;
\.
--COPY public.schema_migrations (version, dirty) FROM '/docker-entrypoint-initdb.d/data_files/3117.dat';

--
-- Name: inputs inputs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.inputs
    ADD CONSTRAINT inputs_pkey PRIMARY KEY (index);


--
-- Name: outputs outputs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.outputs
    ADD CONSTRAINT outputs_pkey PRIMARY KEY (input_index, index);


--
-- Name: proofs proofs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proofs
    ADD CONSTRAINT proofs_pkey PRIMARY KEY (input_index, output_index);


--
-- Name: reports reports_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reports
    ADD CONSTRAINT reports_pkey PRIMARY KEY (input_index, index);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: outputs outputs_input_index_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.outputs
    ADD CONSTRAINT outputs_input_index_fkey FOREIGN KEY (input_index) REFERENCES public.inputs(index);


--
-- Name: proofs proofs_input_index_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proofs
    ADD CONSTRAINT proofs_input_index_fkey FOREIGN KEY (input_index) REFERENCES public.inputs(index);


--
-- Name: proofs proofs_output_index_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.proofs
    ADD CONSTRAINT proofs_output_index_fkey FOREIGN KEY (input_index, output_index) REFERENCES public.outputs(input_index, index);


--
-- Name: reports reports_input_index_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.reports
    ADD CONSTRAINT reports_input_index_fkey FOREIGN KEY (input_index) REFERENCES public.inputs(index);


CREATE TABLE IF NOT EXISTS public.vouchers (
		destination text,
		payload 	text,
		executed	BOOLEAN,
		input_index  integer,
		output_index integer);

ALTER TABLE public.vouchers OWNER TO myuser;

ALTER TABLE ONLY public.vouchers
    ADD CONSTRAINT vouchers_pkey PRIMARY KEY (input_index, output_index);


INSERT INTO public.vouchers VALUES ('OX1223', 'OX1343', false, 1, 1);
INSERT INTO public.vouchers VALUES ('OX1223', 'OX1343', false, 2, 2);

CREATE TABLE IF NOT EXISTS public.notices (
		payload 		text,
		input_index		integer,
		output_index	integer);

ALTER TABLE public.notices OWNER TO myuser;

ALTER TABLE ONLY public.notices
    ADD CONSTRAINT notices_pkey PRIMARY KEY (input_index, output_index);


INSERT INTO public.notices VALUES ('OX1223', 1, 1);
INSERT INTO public.notices VALUES ('OX1223', 2, 2);

-- CREATE TABLE IF NOT EXISTS public.convenience_inputs (
-- 		id 				integer,
-- 		input_index		integer,
-- 		status	 		text,
-- 		msg_sender	 	text,
-- 		payload			text,
-- 		block_number	integer,
-- 		block_timestamp	integer,
-- 		prev_randao		text,
-- 		exception		text);

-- ALTER TABLE public.convenience_inputs OWNER TO myuser;

-- ALTER TABLE ONLY public.convenience_inputs
--     ADD CONSTRAINT cinputs_pkey PRIMARY KEY (id);

-- ALTER TABLE public.convenience_inputs
-- ADD COLUMN app_contract text;

-- INSERT INTO public.convenience_inputs (id, input_index, status, msg_sender, payload, block_number, block_timestamp, prev_randao, exception, app_contract) VALUES (1, 1, '0', '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266', '\x237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb922660000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000a1122334455667788990000000000000000000000000000000000000000000000', 1, 1, '12345', '', '0xB17e9C1C8D91aA5E4DfCc2f3Ee8167Bf75F54D09');
-- INSERT INTO public.convenience_inputs (id, input_index, status, msg_sender, payload, block_number, block_timestamp, prev_randao, exception, app_contract) VALUES (2, 2, '0', '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266', '\x237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb922660000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000a1122334455667788990000000000000000000000000000000000000000000000', 1, 1, '12345', '', '0xB17e9C1C8D91aA5E4DfCc2f3Ee8167Bf75F54D09');

--
-- PostgreSQL database dump complete
--
