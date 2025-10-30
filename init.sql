-- Создаем пользователя если он не существует
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'order_user') THEN
        CREATE USER order_user WITH PASSWORD 'order_password';
    END IF;
END
$$;

-- Создаем базу данных если она не существует
CREATE DATABASE order_db OWNER order_user;

-- Подключаемся к новой базе данных
\c order_db;

-- Создаем необходимое расширение в базе данных order_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Создание таблиц
CREATE TABLE IF NOT EXISTS orders (
    order_uid VARCHAR(255) PRIMARY KEY,
    track_number VARCHAR(255),
    entry VARCHAR(255),
    locale VARCHAR(10),
    internal_signature VARCHAR(255),
    customer_id VARCHAR(255),
    delivery_service VARCHAR(255),
    shardkey VARCHAR(255),
    sm_id INTEGER,
    date_created TIMESTAMP,
    oof_shard VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS delivery (
    order_uid VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    name VARCHAR(255),
    phone VARCHAR(255),
    zip VARCHAR(255),
    city VARCHAR(255),
    address VARCHAR(255),
    region VARCHAR(255),
    email VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS payment (
    order_uid VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    transaction VARCHAR(255),
    request_id VARCHAR(255),
    currency VARCHAR(10),
    provider VARCHAR(255),
    amount INTEGER,
    payment_dt BIGINT,
    bank VARCHAR(255),
    delivery_cost INTEGER,
    goods_total INTEGER,
    custom_fee INTEGER
);

CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(255) REFERENCES orders(order_uid) ON DELETE CASCADE,
    chrt_id INTEGER,
    track_number VARCHAR(255),
    price INTEGER,
    rid VARCHAR(255),
    name VARCHAR(255),
    sale INTEGER,
    size VARCHAR(255),
    total_price INTEGER,
    nm_id INTEGER,
    brand VARCHAR(255),
    status INTEGER
);

-- Вставка тестовых данных
INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
VALUES (
           'b563feb7b2b84b6test',
           'WBILMTESTTRACK',
           'WBIL',
           'en',
           '',
           'test',
           'meest',
           '9',
           99,
           '2021-11-26 06:22:19',
           '1'
       );

INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
VALUES (
           'b563feb7b2b84b6test',
           'Test Testov',
           '+9720000000',
           '2639809',
           'Kiryat Mozkin',
           'Ploshad Mira 15',
           'Kraiot',
           'test@gmail.com'
       );

INSERT INTO payment (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
VALUES (
           'b563feb7b2b84b6test',
           'b563feb7b2b84b6test',
           '',
           'USD',
           'wbpay',
           1817,
           1637907727,
           'alpha',
           1500,
           317,
           0
       );

INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
VALUES (
           'b563feb7b2b84b6test',
           9934930,
           'WBILMTESTTRACK',
           453,
           'ab4219087a764ae0btest',
           'Mascaras',
           30,
           '0',
           317,
           2389212,
           'Vivienne Sabo',
           202
       );

-- Дополнительные тестовые данные
INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
VALUES (
           'test1234567890abcd',
           'TRACK123456',
           'TEST',
           'ru',
           'signature123',
           'customer2',
           'dhl',
           '5',
           88,
           '2021-12-01 10:30:00',
           '2'
       );

INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
VALUES (
           'test1234567890abcd',
           'Ivan Ivanov',
           '+79161234567',
           '123456',
           'Moscow',
           'Tverskaya 10',
           'Moscow Oblast',
           'ivanov@mail.ru'
       );

INSERT INTO payment (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
VALUES (
           'test1234567890abcd',
           'test1234567890abcd',
           'req123',
           'RUB',
           'sberpay',
           5000,
           1638345600,
           'sberbank',
           300,
           4700,
           100
       );

INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
VALUES
    (
        'test1234567890abcd',
        1234567,
        'TRACK123456',
        1000,
        'rid1234567890',
        'Smartphone',
        0,
        '1',
        1000,
        1111111,
        'Samsung',
        200
    ),
    (
        'test1234567890abcd',
        7654321,
        'TRACK123456',
        500,
        'rid0987654321',
        'Case',
        10,
        'M',
        450,
        2222222,
        'Spigen',
        200
    );
-- Даем все права пользователю на базу данных
GRANT ALL PRIVILEGES ON DATABASE order_db TO order_user;