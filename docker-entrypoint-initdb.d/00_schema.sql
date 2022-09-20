CREATE DOMAIN mail AS TEXT
    CHECK(
            VALUE ~ '^[A-Za-z0-9._+%-]+@[A-Za-z0-9.-]+[.][A-Za-z]+$'
        );

CREATE DOMAIN mobile_phone AS TEXT
    CHECK(
            VALUE ~ '^[0-9]{11}$'
        );

CREATE TABLE clients
(
    id        BIGSERIAL PRIMARY KEY,
    full_name TEXT      NOT NULL,
    phone mobile_phone NOT NULL UNIQUE,
    zip VARCHAR(10) NOT NULL,
    city VARCHAR(25) NOT NULL,
    address VARCHAR(200) NOT NULL,
    region VARCHAR(80) NOT NULL,
    email mail NOT NULL UNIQUE
);

CREATE TABLE payments
(
    id BIGSERIAL PRIMARY KEY,
    transaction TEXT NOT NULL UNIQUE,
    request_id TEXT DEFAULT '',
    currency VARCHAR(3) NOT NULL,
    provider TEXT NOT NULL,
    amount INTEGER NOT NULL,
    payment_dt INTEGER NOT NULL,
    bank VARCHAR(20) NOT NULL,
    delivery_cost INTEGER NOT NULL DEFAULT 0,
    goods_total INTEGER NOT NULL,
    custom_fee INTEGER NOT NULL DEFAULT 0
);


CREATE TABLE items
(
    id BIGSERIAL PRIMARY KEY,
    track_number TEXT NOT NULL,
    price INTEGER NOT NULL,
    rid TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    sale INTEGER,
    size VARCHAR(3) NOT NULL DEFAULT '0',
    total_price INTEGER NOT NULL,
    nm_id INTEGER NOT NULL,
    brand TEXT NOT NULL,
    status INTEGER NOT NULL
);


CREATE TABLE orders
(
    id       BIGSERIAL PRIMARY KEY,
    order_uid   TEXT   NOT NULL UNIQUE,
    track_number  TEXT    NOT NULL,
    entry TEXT NOT NULL,
    delivery BIGINT REFERENCES clients(id) NOT NULL,
    payment BIGINT REFERENCES payments(id) NOT NULL,
    locale VARCHAR(4) NOT NULL,
    internal_signature TEXT DEFAULT '',
    customer_id TEXT NOT NULL,
    delivery_service TEXT NOT NULL,
    shardkey VARCHAR(3) NOT NULL,
    sm_id INTEGER NOT NULL ,
    date_created TIMESTAMP NOT NULL,
    oof_shard VARCHAR(3) NOT NULL
);

CREATE TABLE order_to_items
(
    order_id BIGINT REFERENCES orders(id),
    item_id BIGINT REFERENCES items(id)
);













