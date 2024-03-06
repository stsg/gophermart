-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS users (
    uid uuid UNIQUE NOT NULL PRIMARY KEY default gen_random_uuid(), 
    login text UNIQUE NOT NULL,
    password text NOT NULL,
    jwt text DEFAULT NULL,
    createrd_at timestamptz NOT NULL DEFAULT NOW(),
    deleted boolean NOT NULL default FALSE
);

CREATE TABLE IF NOT EXISTS orders (
    id text NOT NULL PRIMARY KEY,
    uid uuid NOT NULL,
    amount int DEFAULT 0,
    status text NOT NULL DEFAULT 'NEW',
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    deleted boolean NOT NULL default FALSE
);

CREATE TABLE IF NOT EXISTS withdrawals (
    order_id text UNIQUE NOT NULL PRIMARY KEY,
    uid uuid NOT NULL,
    amount int DEFAULT 0,
    processed_at timestamptz NOT NULL DEFAULT NOW(),
    deleted boolean NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS balances (
    uid uuid UNIQUE NOT NULL PRIMARY KEY,
    current_balance float NOT NULL DEFAULT 0,
    withdrawn float NOT NULL DEFAULT 0
);

INSERT INTO
    users (login, password)
        VALUES 
            ('stas', '$2a$10$k4/iXqhXQg/mK/fsDXbF5Ocq50yPzkaw4l4Elg37A38fYmtw7oxAm'),
            ('nata', '$2a$10$7ixg.hUXcUF4YTHZfgrU.ePgOhvAZhu5sIaOa4TTTwgIfxIhVnMry');

-- +goose Down
DROP TABLE withdrawals;
DROP TABLE balances;
DROP TABLE orders;
DROP TABLE users;
DROP EXTENSION IF EXISTS "uuid-ossp";