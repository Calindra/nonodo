--
-- PostgreSQL database dump
--

-- Dumped from database version 16.4
-- Dumped by pg_dump version 17.0

-- Started on 2024-10-11 11:28:57 -03

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 3524 (class 1262 OID 16384)
-- Name: rollupsdb; Type: DATABASE; Schema: -; Owner: postgres
--

CREATE DATABASE rollupsdb WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE rollupsdb OWNER TO postgres;

\connect rollupsdb

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 4 (class 2615 OID 2200)
-- Name: public; Type: SCHEMA; Schema: -; Owner: pg_database_owner
--

CREATE SCHEMA public;


ALTER SCHEMA public OWNER TO pg_database_owner;

--
-- TOC entry 3525 (class 0 OID 0)
-- Dependencies: 4
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: pg_database_owner
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- TOC entry 863 (class 1247 OID 16399)
-- Name: ApplicationStatus; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."ApplicationStatus" AS ENUM (
    'RUNNING',
    'NOT RUNNING'
);


ALTER TYPE public."ApplicationStatus" OWNER TO postgres;

--
-- TOC entry 869 (class 1247 OID 16422)
-- Name: DefaultBlock; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."DefaultBlock" AS ENUM (
    'FINALIZED',
    'LATEST',
    'PENDING',
    'SAFE'
);


ALTER TYPE public."DefaultBlock" OWNER TO postgres;

--
-- TOC entry 872 (class 1247 OID 16432)
-- Name: EpochStatus; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."EpochStatus" AS ENUM (
    'OPEN',
    'CLOSED',
    'PROCESSED_ALL_INPUTS',
    'CLAIM_COMPUTED',
    'CLAIM_SUBMITTED',
    'CLAIM_ACCEPTED',
    'CLAIM_REJECTED'
);


ALTER TYPE public."EpochStatus" OWNER TO postgres;

--
-- TOC entry 866 (class 1247 OID 16404)
-- Name: InputCompletionStatus; Type: TYPE; Schema: public; Owner: postgres
--

CREATE TYPE public."InputCompletionStatus" AS ENUM (
    'NONE',
    'ACCEPTED',
    'REJECTED',
    'EXCEPTION',
    'MACHINE_HALTED',
    'CYCLE_LIMIT_EXCEEDED',
    'TIME_LIMIT_EXCEEDED',
    'PAYLOAD_LENGTH_LIMIT_EXCEEDED'
);


ALTER TYPE public."InputCompletionStatus" OWNER TO postgres;

--
-- TOC entry 235 (class 1255 OID 16447)
-- Name: f_maxuint64(); Type: FUNCTION; Schema: public; Owner: postgres
--

CREATE FUNCTION public.f_maxuint64() RETURNS numeric
    LANGUAGE sql IMMUTABLE PARALLEL SAFE
    AS $$SELECT 18446744073709551615$$;


ALTER FUNCTION public.f_maxuint64() OWNER TO postgres;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 218 (class 1259 OID 16449)
-- Name: application; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.application (
    id integer NOT NULL,
    contract_address bytea NOT NULL,
    template_hash bytea NOT NULL,
    template_uri character varying(4096) NOT NULL,
    last_processed_block numeric(20,0) NOT NULL,
    status public."ApplicationStatus" NOT NULL,
    iconsensus_address bytea NOT NULL,
    last_claim_check_block numeric(20,0) NOT NULL,
    last_output_check_block numeric(20,0) NOT NULL,
    CONSTRAINT application_last_claim_check_block_check CHECK (((last_claim_check_block >= (0)::numeric) AND (last_claim_check_block <= public.f_maxuint64()))),
    CONSTRAINT application_last_output_check_block_check CHECK (((last_output_check_block >= (0)::numeric) AND (last_output_check_block <= public.f_maxuint64()))),
    CONSTRAINT application_last_processed_block_check CHECK (((last_processed_block >= (0)::numeric) AND (last_processed_block <= public.f_maxuint64())))
);


ALTER TABLE public.application OWNER TO postgres;

--
-- TOC entry 220 (class 1259 OID 16463)
-- Name: epoch; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.epoch (
    id bigint NOT NULL,
    application_address bytea NOT NULL,
    index bigint NOT NULL,
    first_block numeric(20,0) NOT NULL,
    last_block numeric(20,0) NOT NULL,
    claim_hash bytea,
    transaction_hash bytea,
    status public."EpochStatus" NOT NULL,
    CONSTRAINT epoch_first_block_check CHECK (((first_block >= (0)::numeric) AND (first_block <= public.f_maxuint64()))),
    CONSTRAINT epoch_last_block_check CHECK (((last_block >= (0)::numeric) AND (last_block <= public.f_maxuint64())))
);


ALTER TABLE public.epoch OWNER TO postgres;

--
-- TOC entry 222 (class 1259 OID 16483)
-- Name: input; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.input (
    id bigint NOT NULL,
    index numeric(20,0) NOT NULL,
    raw_data bytea NOT NULL,
    block_number numeric(20,0) NOT NULL,
    status public."InputCompletionStatus" NOT NULL,
    machine_hash bytea,
    outputs_hash bytea,
    application_address bytea NOT NULL,
    epoch_id bigint NOT NULL,
    CONSTRAINT input_block_number_check CHECK (((block_number >= (0)::numeric) AND (block_number <= public.f_maxuint64()))),
    CONSTRAINT input_index_check CHECK (((index >= (0)::numeric) AND (index <= public.f_maxuint64())))
);


ALTER TABLE public.input OWNER TO postgres;

--
-- TOC entry 224 (class 1259 OID 16507)
-- Name: output; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.output (
    id bigint NOT NULL,
    index numeric(20,0) NOT NULL,
    raw_data bytea NOT NULL,
    hash bytea,
    output_hashes_siblings bytea[],
    input_id bigint NOT NULL,
    transaction_hash bytea,
    CONSTRAINT output_index_check CHECK (((index >= (0)::numeric) AND (index <= public.f_maxuint64())))
);


ALTER TABLE public.output OWNER TO postgres;

--
-- TOC entry 226 (class 1259 OID 16523)
-- Name: report; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.report (
    id bigint NOT NULL,
    index numeric(20,0) NOT NULL,
    raw_data bytea NOT NULL,
    input_id bigint NOT NULL,
    CONSTRAINT report_index_check CHECK (((index >= (0)::numeric) AND (index <= public.f_maxuint64())))
);


ALTER TABLE public.report OWNER TO postgres;

--
-- TOC entry 217 (class 1259 OID 16448)
-- Name: application_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.application_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.application_id_seq OWNER TO postgres;

--
-- TOC entry 3526 (class 0 OID 0)
-- Dependencies: 217
-- Name: application_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.application_id_seq OWNED BY public.application.id;


--
-- TOC entry 219 (class 1259 OID 16462)
-- Name: epoch_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.epoch_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.epoch_id_seq OWNER TO postgres;

--
-- TOC entry 3527 (class 0 OID 0)
-- Dependencies: 219
-- Name: epoch_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.epoch_id_seq OWNED BY public.epoch.id;


--
-- TOC entry 221 (class 1259 OID 16482)
-- Name: input_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.input_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.input_id_seq OWNER TO postgres;

--
-- TOC entry 3528 (class 0 OID 0)
-- Dependencies: 221
-- Name: input_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.input_id_seq OWNED BY public.input.id;


--
-- TOC entry 229 (class 1259 OID 16559)
-- Name: node_config; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.node_config (
    default_block public."DefaultBlock" NOT NULL,
    input_box_deployment_block integer NOT NULL,
    input_box_address bytea NOT NULL,
    chain_id integer NOT NULL
);


ALTER TABLE public.node_config OWNER TO postgres;

--
-- TOC entry 223 (class 1259 OID 16506)
-- Name: output_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.output_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.output_id_seq OWNER TO postgres;

--
-- TOC entry 3529 (class 0 OID 0)
-- Dependencies: 223
-- Name: output_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.output_id_seq OWNED BY public.output.id;


--
-- TOC entry 225 (class 1259 OID 16522)
-- Name: report_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.report_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.report_id_seq OWNER TO postgres;

--
-- TOC entry 3530 (class 0 OID 0)
-- Dependencies: 225
-- Name: report_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.report_id_seq OWNED BY public.report.id;


--
-- TOC entry 216 (class 1259 OID 16391)
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


ALTER TABLE public.schema_migrations OWNER TO postgres;

--
-- TOC entry 228 (class 1259 OID 16539)
-- Name: snapshot; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.snapshot (
    id bigint NOT NULL,
    input_id bigint NOT NULL,
    application_address bytea NOT NULL,
    uri character varying(4096) NOT NULL
);


ALTER TABLE public.snapshot OWNER TO postgres;

--
-- TOC entry 227 (class 1259 OID 16538)
-- Name: snapshot_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.snapshot_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.snapshot_id_seq OWNER TO postgres;

--
-- TOC entry 3531 (class 0 OID 0)
-- Dependencies: 227
-- Name: snapshot_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.snapshot_id_seq OWNED BY public.snapshot.id;


--
-- TOC entry 3308 (class 2604 OID 16452)
-- Name: application id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.application ALTER COLUMN id SET DEFAULT nextval('public.application_id_seq'::regclass);


--
-- TOC entry 3309 (class 2604 OID 16466)
-- Name: epoch id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.epoch ALTER COLUMN id SET DEFAULT nextval('public.epoch_id_seq'::regclass);


--
-- TOC entry 3310 (class 2604 OID 16486)
-- Name: input id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input ALTER COLUMN id SET DEFAULT nextval('public.input_id_seq'::regclass);


--
-- TOC entry 3311 (class 2604 OID 16510)
-- Name: output id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.output ALTER COLUMN id SET DEFAULT nextval('public.output_id_seq'::regclass);


--
-- TOC entry 3312 (class 2604 OID 16526)
-- Name: report id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.report ALTER COLUMN id SET DEFAULT nextval('public.report_id_seq'::regclass);


--
-- TOC entry 3313 (class 2604 OID 16542)
-- Name: snapshot id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.snapshot ALTER COLUMN id SET DEFAULT nextval('public.snapshot_id_seq'::regclass);


--
-- TOC entry 3507 (class 0 OID 16449)
-- Dependencies: 218
-- Data for Name: application; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.application VALUES (1, '\x5112cf49f2511ac7b13a032c4c62a48410fc28fb', '\xd61ce5095e54ea2ad6e826dc6dee990b76b0a71bf0cf806b5be562ae7cd7a74b', 'applications/echo-dapp', 1188, 'RUNNING', '\x3fd5dc9dcf5df3c7002c0628eb9ad3bb5e2ce257', 1188, 1188);


--
-- TOC entry 3509 (class 0 OID 16463)
-- Dependencies: 220
-- Data for Name: epoch; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.epoch VALUES (1, '\x5112cf49f2511ac7b13a032c4c62a48410fc28fb', 112, 1120, 1129, NULL, NULL, 'PROCESSED_ALL_INPUTS');
INSERT INTO public.epoch VALUES (2, '\x5112cf49f2511ac7b13a032c4c62a48410fc28fb', 115, 1150, 1159, NULL, NULL, 'PROCESSED_ALL_INPUTS');


--
-- TOC entry 3511 (class 0 OID 16483)
-- Dependencies: 222
-- Data for Name: input; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.input VALUES (1, 0, '\x415bf3630000000000000000000000000000000000000000000000000000000000007a690000000000000000000000005112cf49f2511ac7b13a032c4c62a48410fc28fb000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000046900000000000000000000000000000000000000000000000000000000670931c70a06511d13afecb37c88e47c1a7357e42205ac4b8e49fcd4632373e036261e26000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000', 1129, 'ACCEPTED', '\x0de13656d1f133a93069f1fc2962814ed3193a35dee028da211165c3a3b74cc1', '\xdd27bf753d0b372c08536587772b3c68aa6fdf5114c47dccae824de63e95e8f8', '\x5112cf49f2511ac7b13a032c4c62a48410fc28fb', 1);
INSERT INTO public.input VALUES (2, 1, '\x415bf3630000000000000000000000000000000000000000000000000000000000007a690000000000000000000000005112cf49f2511ac7b13a032c4c62a48410fc28fb000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266000000000000000000000000000000000000000000000000000000000000048000000000000000000000000000000000000000000000000000000000670931defa7c5819a4f71bd1175c27329a1a67c62bf76702fac35445d248dbda23777ee6000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000', 1152, 'ACCEPTED', '\x8f050fb0095daa8543fb8010dbc732b12bab73883e05f5ec746652b62d6416cd', '\x1432c1d4354e2133372cad6153bf08a953bb450147bb48c5145cca1a07ed0594', '\x5112cf49f2511ac7b13a032c4c62a48410fc28fb', 2);


--
-- TOC entry 3518 (class 0 OID 16559)
-- Dependencies: 229
-- Data for Name: node_config; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.node_config VALUES ('FINALIZED', 10, '\x593e5bcf894d6829dd26d0810da7f064406aebb6', 31337);


--
-- TOC entry 3513 (class 0 OID 16507)
-- Dependencies: 224
-- Data for Name: output; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.output VALUES (1, 0, '\x237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000deadbeef00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000', NULL, NULL, 1, NULL);
INSERT INTO public.output VALUES (2, 1, '\xc258d6e500000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000', NULL, NULL, 1, NULL);
INSERT INTO public.output VALUES (3, 2, '\x237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000deadbeef00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000', NULL, NULL, 2, NULL);
INSERT INTO public.output VALUES (4, 3, '\xc258d6e500000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000005deadbeef11000000000000000000000000000000000000000000000000000000', NULL, NULL, 2, NULL);


--
-- TOC entry 3515 (class 0 OID 16523)
-- Dependencies: 226
-- Data for Name: report; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.report VALUES (1, 0, '\xdeadbeef11', 1);
INSERT INTO public.report VALUES (2, 1, '\xdeadbeef11', 2);


--
-- TOC entry 3505 (class 0 OID 16391)
-- Dependencies: 216
-- Data for Name: schema_migrations; Type: TABLE DATA; Schema: public; Owner: postgres
--

INSERT INTO public.schema_migrations VALUES (2, false);


--
-- TOC entry 3517 (class 0 OID 16539)
-- Dependencies: 228
-- Data for Name: snapshot; Type: TABLE DATA; Schema: public; Owner: postgres
--



--
-- TOC entry 3532 (class 0 OID 0)
-- Dependencies: 217
-- Name: application_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.application_id_seq', 1, true);


--
-- TOC entry 3533 (class 0 OID 0)
-- Dependencies: 219
-- Name: epoch_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.epoch_id_seq', 3, true);


--
-- TOC entry 3534 (class 0 OID 0)
-- Dependencies: 221
-- Name: input_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.input_id_seq', 2, true);


--
-- TOC entry 3535 (class 0 OID 0)
-- Dependencies: 223
-- Name: output_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.output_id_seq', 4, true);


--
-- TOC entry 3536 (class 0 OID 0)
-- Dependencies: 225
-- Name: report_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.report_id_seq', 2, true);


--
-- TOC entry 3537 (class 0 OID 0)
-- Dependencies: 227
-- Name: snapshot_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.snapshot_id_seq', 1, false);


--
-- TOC entry 3326 (class 2606 OID 16461)
-- Name: application application_contract_address_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.application
    ADD CONSTRAINT application_contract_address_key UNIQUE (contract_address);


--
-- TOC entry 3328 (class 2606 OID 16459)
-- Name: application application_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.application
    ADD CONSTRAINT application_pkey PRIMARY KEY (id);


--
-- TOC entry 3331 (class 2606 OID 16474)
-- Name: epoch epoch_index_application_address_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.epoch
    ADD CONSTRAINT epoch_index_application_address_key UNIQUE (index, application_address);


--
-- TOC entry 3334 (class 2606 OID 16472)
-- Name: epoch epoch_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.epoch
    ADD CONSTRAINT epoch_pkey PRIMARY KEY (id);


--
-- TOC entry 3337 (class 2606 OID 16494)
-- Name: input input_index_application_address_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input
    ADD CONSTRAINT input_index_application_address_key UNIQUE (index, application_address);


--
-- TOC entry 3339 (class 2606 OID 16492)
-- Name: input input_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input
    ADD CONSTRAINT input_pkey PRIMARY KEY (id);


--
-- TOC entry 3342 (class 2606 OID 16515)
-- Name: output output_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.output
    ADD CONSTRAINT output_pkey PRIMARY KEY (id);


--
-- TOC entry 3345 (class 2606 OID 16531)
-- Name: report report_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.report
    ADD CONSTRAINT report_pkey PRIMARY KEY (id);


--
-- TOC entry 3324 (class 2606 OID 16395)
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- TOC entry 3347 (class 2606 OID 16548)
-- Name: snapshot snapshot_input_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.snapshot
    ADD CONSTRAINT snapshot_input_id_key UNIQUE (input_id);


--
-- TOC entry 3349 (class 2606 OID 16546)
-- Name: snapshot snapshot_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.snapshot
    ADD CONSTRAINT snapshot_pkey PRIMARY KEY (id);


--
-- TOC entry 3329 (class 1259 OID 16480)
-- Name: epoch_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX epoch_idx ON public.epoch USING btree (index);


--
-- TOC entry 3332 (class 1259 OID 16481)
-- Name: epoch_last_block_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX epoch_last_block_idx ON public.epoch USING btree (last_block);


--
-- TOC entry 3335 (class 1259 OID 16505)
-- Name: input_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX input_idx ON public.input USING btree (block_number);


--
-- TOC entry 3340 (class 1259 OID 16521)
-- Name: output_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX output_idx ON public.output USING btree (index);


--
-- TOC entry 3343 (class 1259 OID 16537)
-- Name: report_idx; Type: INDEX; Schema: public; Owner: postgres
--

CREATE INDEX report_idx ON public.report USING btree (index);


--
-- TOC entry 3350 (class 2606 OID 16475)
-- Name: epoch epoch_application_address_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.epoch
    ADD CONSTRAINT epoch_application_address_fkey FOREIGN KEY (application_address) REFERENCES public.application(contract_address);


--
-- TOC entry 3351 (class 2606 OID 16495)
-- Name: input input_application_address_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input
    ADD CONSTRAINT input_application_address_fkey FOREIGN KEY (application_address) REFERENCES public.application(contract_address);


--
-- TOC entry 3352 (class 2606 OID 16500)
-- Name: input input_epoch_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.input
    ADD CONSTRAINT input_epoch_fkey FOREIGN KEY (epoch_id) REFERENCES public.epoch(id);


--
-- TOC entry 3353 (class 2606 OID 16516)
-- Name: output output_input_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.output
    ADD CONSTRAINT output_input_id_fkey FOREIGN KEY (input_id) REFERENCES public.input(id);


--
-- TOC entry 3354 (class 2606 OID 16532)
-- Name: report report_input_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.report
    ADD CONSTRAINT report_input_id_fkey FOREIGN KEY (input_id) REFERENCES public.input(id);


--
-- TOC entry 3355 (class 2606 OID 16554)
-- Name: snapshot snapshot_application_address_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.snapshot
    ADD CONSTRAINT snapshot_application_address_fkey FOREIGN KEY (application_address) REFERENCES public.application(contract_address);


--
-- TOC entry 3356 (class 2606 OID 16549)
-- Name: snapshot snapshot_input_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.snapshot
    ADD CONSTRAINT snapshot_input_id_fkey FOREIGN KEY (input_id) REFERENCES public.input(id);


-- Completed on 2024-10-11 11:28:57 -03

--
-- PostgreSQL database dump complete
--

