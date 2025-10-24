// Package database содержит SQL запросы для работы с базой данных
package database

// SQL Queries
const (
	// Создание таблиц
	CreateOrdersTable = `CREATE TABLE IF NOT EXISTS orders (
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
	)`

	CreateDeliveryTable = `CREATE TABLE IF NOT EXISTS delivery (
		order_uid VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
		name VARCHAR(255),
		phone VARCHAR(255),
		zip VARCHAR(255),
		city VARCHAR(255),
		address VARCHAR(255),
		region VARCHAR(255),
		email VARCHAR(255)
	)`

	CreatePaymentTable = `CREATE TABLE IF NOT EXISTS payment (
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
	)`

	CreateItemsTable = `CREATE TABLE IF NOT EXISTS items (
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
	)`

	// Индексы
	CreateOrdersIndex = `CREATE INDEX IF NOT EXISTS idx_orders_track_number ON orders(track_number)`
	CreateItemsIndex = `CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid)`

	// Сохранение заказа (UPSERT)
	SaveOrderQuery = `INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (order_uid) DO UPDATE SET
			track_number = EXCLUDED.track_number,
			entry = EXCLUDED.entry,
			locale = EXCLUDED.locale,
			internal_signature = EXCLUDED.internal_signature,
			customer_id = EXCLUDED.customer_id,
			delivery_service = EXCLUDED.delivery_service,
			shardkey = EXCLUDED.shardkey,
			sm_id = EXCLUDED.sm_id,
			date_created = EXCLUDED.date_created,
			oof_shard = EXCLUDED.oof_shard`

	// Сохранение доставки (UPSERT)
	SaveDeliveryQuery = `INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (order_uid) DO UPDATE SET
			name = EXCLUDED.name,
			phone = EXCLUDED.phone,
			zip = EXCLUDED.zip,
			city = EXCLUDED.city,
			address = EXCLUDED.address,
			region = EXCLUDED.region,
			email = EXCLUDED.email`

	// Сохранение платежа (UPSERT)
	SavePaymentQuery = `INSERT INTO payment (order_uid, transaction, request_id, currency, provider,
			amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (order_uid) DO UPDATE SET
			transaction = EXCLUDED.transaction,
			request_id = EXCLUDED.request_id,
			currency = EXCLUDED.currency,
			provider = EXCLUDED.provider,
			amount = EXCLUDED.amount,
			payment_dt = EXCLUDED.payment_dt,
			bank = EXCLUDED.bank,
			delivery_cost = EXCLUDED.delivery_cost,
			goods_total = EXCLUDED.goods_total,
			custom_fee = EXCLUDED.custom_fee`

	// Удаление товаров заказа
	DeleteItemsQuery = `DELETE FROM items WHERE order_uid = $1`

	// Сохранение товара
	SaveItemQuery = `INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size,
			total_price, nm_id, brand, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	// Получение заказа по UID
	GetOrderByUIDQuery = `SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature,
			o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, 
			p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		JOIN delivery d ON o.order_uid = d.order_uid
		JOIN payment p ON o.order_uid = p.order_uid
		WHERE o.order_uid = $1`

	// Получение товаров заказа
	GetItemsByOrderUIDQuery = `SELECT chrt_id, track_number, price, rid, name, sale, size,
			total_price, nm_id, brand, status
		FROM items 
		WHERE order_uid = $1
		ORDER BY id`

	// Получение всех заказов
	GetAllOrdersQuery = `SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature,
			o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, 
			p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		JOIN delivery d ON o.order_uid = d.order_uid
		JOIN payment p ON o.order_uid = p.order_uid
		ORDER BY o.date_created DESC`
)