CREATE TABLE users (
    uuid SERIAL PRIMARY KEY,
    login text NOT NULL,
    password text NOT NULL,
    bonus int NOT NULL default 0,
    spentbonus int NOT NULL default 0,
    deleted boolean NOT NULL default false
);

ALTER TABLE users
    ADD CONSTRAINT login
        UNIQUE (login);

INSERT INTO
    users (login, password)
        VALUES ('stas', 'q1w2e3');
