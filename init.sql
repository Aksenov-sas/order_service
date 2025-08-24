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
                                      order_uid TEXT PRIMARY KEY,
                                      track_number TEXT NOT NULL,
                                      entry TEXT NOT NULL,
                                      locale TEXT NOT NULL,
                                      internal_signature TEXT,
                                      customer_id TEXT NOT NULL,
                                      delivery_service TEXT NOT NULL,
                                      shardkey TEXT NOT NULL,
                                      sm_id INTEGER NOT NULL,
                                      date_created TIMESTAMP NOT NULL,
                                      oof_shard TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS delivery (
                                        order_uid TEXT PRIMARY KEY,
                                        name TEXT NOT NULL,
                                        phone TEXT NOT NULL,
                                        zip TEXT NOT NULL,
                                        city TEXT NOT NULL,
                                        address TEXT NOT NULL,
                                        region TEXT NOT NULL,
                                        email TEXT NOT NULL,
                                        FOREIGN KEY (order_uid) REFERENCES orders (order_uid) ON DELETE CASCADE
    );

CREATE TABLE IF NOT EXISTS payment (
                                       transaction TEXT PRIMARY KEY,
                                       order_uid TEXT NOT NULL,
                                       request_id TEXT,
                                       currency TEXT NOT NULL,
                                       provider TEXT NOT NULL,
                                       amount INTEGER NOT NULL,
                                       payment_dt INTEGER NOT NULL,
                                       bank TEXT NOT NULL,
                                       delivery_cost INTEGER NOT NULL,
                                       goods_total INTEGER NOT NULL,
                                       custom_fee INTEGER NOT NULL,
                                       FOREIGN KEY (order_uid) REFERENCES orders (order_uid) ON DELETE CASCADE
    );

CREATE TABLE IF NOT EXISTS items (
                                     id SERIAL PRIMARY KEY,
                                     order_uid TEXT NOT NULL,
                                     chrt_id INTEGER NOT NULL,
                                     track_number TEXT NOT NULL,
                                     price INTEGER NOT NULL,
                                     rid TEXT NOT NULL,
                                     name TEXT NOT NULL,
                                     sale INTEGER NOT NULL,
                                     size TEXT NOT NULL,
                                     total_price INTEGER NOT NULL,
                                     nm_id INTEGER NOT NULL,
                                     brand TEXT NOT NULL,
                                     status INTEGER NOT NULL,
                                     FOREIGN KEY (order_uid) REFERENCES orders (order_uid) ON DELETE CASCADE
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

INSERT INTO payment (transaction, order_uid, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
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

INSERT INTO payment (transaction, order_uid, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
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