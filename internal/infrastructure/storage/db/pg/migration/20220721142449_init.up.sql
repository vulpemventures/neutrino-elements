CREATE TABLE block_header (
    hash varchar(100) PRIMARY KEY,
    height int NOT NULL UNIQUE,
    header_bytes bytea NOT NULL
);

CREATE TABLE filter(
    filter_key   varchar(100) PRIMARY KEY,
    filter_value bytea NOT NULL
);
