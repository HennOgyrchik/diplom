CREATE TABLE if not exists funds (
                              id serial4 NOT NULL,
                              tag varchar(15) NOT NULL,
                              balance float8 NOT NULL,
                              CONSTRAINT funds_pkey PRIMARY KEY (id)
);

CREATE TABLE if not exists members (
                                id serial4 NOT NULL,
                                tag varchar(15) NOT NULL,
                                member_id int8 NOT NULL,
                                "admin" bool DEFAULT false,
                                login varchar,
                                "name" text,
                                CONSTRAINT members_pkey PRIMARY KEY (id)
);

CREATE TABLE if not exists cash_collections (
                                         id serial4 NOT NULL,
                                         tag varchar(15) NOT NULL,
                                         sum float8 NOT NULL,
                                         status varchar(10) NOT NULL,
                                         "comment" text,
                                         create_date date NOT NULL,
                                         close_date date,
                                         purpose text NOT NULL
);

CREATE TABLE if not exists transactions (
                                     id serial4 NOT NULL,
                                     cash_collection_id int4 NOT NULL,
                                     sum float8 NOT NULL,
                                     "type" varchar(15) NOT NULL,
                                     status varchar(25) NOT NULL,
                                     receipt text,
                                     member_id int8 NOT NULL,
                                     "date" date NOT NULL
);