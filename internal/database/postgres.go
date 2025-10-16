// Пакет database содержит логику работы с базой данных PostgreSQL
package database

import (
	"context"
	"fmt"
	"log"
	"test_service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres представляет подключение к базе данных PostgreSQL
type Postgres struct {
	pool *pgxpool.Pool // Пул соединений с базой данных
}

// NewPostgres создает новое подключение к базе данных PostgreSQL
func NewPostgres(ctx context.Context, connectStr string) (*Postgres, error) {
	// Парсим строку подключения
	config, err := pgxpool.ParseConfig(connectStr)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при анализе строки для подключения:%v", err)
	}

	// Создаем пул соединений
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при создании подключения:%v", err)
	}

	// Проверяем соединение с базой данных
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Ошибка соединения с БД:%v", err)
	}

	return &Postgres{pool: pool}, nil
}

// Init инициализирует базу данных, создавая необходимые таблицы и индексы
func (p *Postgres) Init(ctx context.Context) error {
	// SQL запросы для создания таблиц и индексов
	queries := []string{
		// Таблица заказов
		`CREATE TABLE IF NOT EXISTS orders (
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
		)`,

		// Таблица доставки
		`CREATE TABLE IF NOT EXISTS delivery (
			order_uid VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
			name VARCHAR(255),
			phone VARCHAR(255),
			zip VARCHAR(255),
			city VARCHAR(255),
			address VARCHAR(255),
			region VARCHAR(255),
			email VARCHAR(255)
		)`,

		// Таблица платежей
		`CREATE TABLE IF NOT EXISTS payment (
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
		)`,

		// Таблица товаров
		`CREATE TABLE IF NOT EXISTS items (
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
		)`,

		// Индексы для оптимизации запросов
		`CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created)`,
	}

	// Выполняем все SQL запросы
	for _, query := range queries {
		_, err := p.pool.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("Ошибка выполнения запроса %s: %v", query, err)
		}
	}

	// Простейшая миграционная таблица для детерминированных миграций
	if _, err := p.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (id TEXT PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT NOW())`); err != nil {
		return fmt.Errorf("Ошибка создания schema_migrations: %v", err)
	}

	type migration struct{ id, sql string }
	migrations := []migration{}
	for _, m := range migrations {
		var exists bool
		err := p.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE id=$1)`, m.id).Scan(&exists)
		if err != nil {
			return fmt.Errorf("Ошибка проверки миграции %s: %v", m.id, err)
		}
		if exists {
			continue
		}
		if _, err := p.pool.Exec(ctx, m.sql); err != nil {
			return fmt.Errorf("Ошибка применения миграции %s: %v", m.id, err)
		}
		if _, err := p.pool.Exec(ctx, `INSERT INTO schema_migrations (id) VALUES ($1)`, m.id); err != nil {
			return fmt.Errorf("Ошибка записи миграции %s: %v", m.id, err)
		}
		log.Printf("Применена миграция: %s", m.id)
	}

	log.Println("БД инициализирована")
	return nil
}

// SaveOrder сохраняет заказ в базу данных в рамках транзакции
func (p *Postgres) SaveOrder(ctx context.Context, order *models.Order) error {
	// Начинаем транзакцию
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("Ошибка начала транзакции: %v", err)
	}
	defer tx.Rollback(ctx) // Откатываем транзакцию в случае ошибки

	// Сохраняем основную информацию о заказе (UPSERT)
	_, err = tx.Exec(ctx, `INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, 
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
			oof_shard = EXCLUDED.oof_shard
	`, order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.ShardKey, order.SMID, order.DateCreated, order.OOFShard)
	if err != nil {
		return fmt.Errorf("Ошибка при записи заказа: %v", err)
	}

	// Сохраняем информацию о доставке (UPSERT)
	_, err = tx.Exec(ctx, `
		INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (order_uid) DO UPDATE SET
			name = EXCLUDED.name,
			phone = EXCLUDED.phone,
			zip = EXCLUDED.zip,
			city = EXCLUDED.city,
			address = EXCLUDED.address,
			region = EXCLUDED.region,
			email = EXCLUDED.email
	`, order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return fmt.Errorf("Ошибка при записи доставки: %v", err)
	}

	// Сохраняем информацию о платеже (UPSERT)
	_, err = tx.Exec(ctx, `
		INSERT INTO payment (order_uid, transaction, request_id, currency, provider, 
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
			custom_fee = EXCLUDED.custom_fee
	`, order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("Ошибка при записи payment: %v", err)
	}

	// Удаляем старые товары заказа (для обновления)
	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid = $1`,
		order.OrderUID)
	if err != nil {
		return fmt.Errorf("Ошибка удаления позиций: %v", err)
	}

	// Добавляем новые товары заказа
	for _, items := range order.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, 
				sale, size, total_price, nm_id, brand, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, order.OrderUID, items.ChrtID, items.TrackNumber, items.Price, items.RID, items.Name,
			items.Sale, items.Size, items.TotalPrice, items.NMID, items.Brand, items.Status)
		if err != nil {
			return fmt.Errorf("Ошибка добавления позиции: %v", err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("Ошибка коммита транзакции: %v", err)
	}
	return nil
}

// GetOrder получает заказ из базы данных по его UID
func (p *Postgres) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	var order models.Order

	// Получаем основную информацию о заказе
	err := p.pool.QueryRow(ctx, `SELECT order_uid, track_number, entry, locale, internal_signature, 
			customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid = $1
	`, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerID, &order.DeliveryService, &order.ShardKey, &order.SMID, &order.DateCreated, &order.OOFShard)
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения заказа: %v", err)
	}

	// Получаем информацию о доставке
	err = p.pool.QueryRow(ctx, `SELECT name, phone, zip, city, address, region, email
		FROM delivery WHERE order_uid = $1
	`, orderUID).Scan(
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
		&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
	)
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения данных о доставке: %v", err)
	}

	// Получаем информацию о платеже
	err = p.pool.QueryRow(ctx, `
		SELECT transaction, request_id, currency, provider, amount, payment_dt, 
			bank, delivery_cost, goods_total, custom_fee
		FROM payment WHERE order_uid = $1
	`, orderUID).Scan(
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider,
		&order.Payment.Amount, &order.Payment.PaymentDT, &order.Payment.Bank, &order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal, &order.Payment.CustomFee)
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения данных о платёжных средствах: %v", err)
	}

	// Получаем список товаров заказа
	rows, err := p.pool.Query(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, 
			total_price, nm_id, brand, status
		FROM items WHERE order_uid = $1
	`, orderUID)
	if err != nil {
		return nil, fmt.Errorf("Не удалось запросить items: %v", err)
	}
	defer rows.Close()

	// Обрабатываем результаты запроса
	order.Items = []models.Item{}
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name, &item.Sale,
			&item.Size, &item.TotalPrice, &item.NMID, &item.Brand, &item.Status)
		if err != nil {
			return nil, fmt.Errorf("Ошибка при чтении items:%v", err)
		}
		order.Items = append(order.Items, item)
	}

	// Проверяем ошибки при итерации
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Ошибка при переборе items: %v", err)
	}

	return &order, nil
}

// GetAllOrders получает все заказы из базы данных
func (p *Postgres) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	// Получаем список всех UID заказов
	rows, err := p.pool.Query(ctx, "SELECT order_uid FROM orders")
	if err != nil {
		return nil, fmt.Errorf("Ошибка при запросе заказов: %v", err)
	}
	defer rows.Close()

	// Получаем полную информацию о каждом заказе
	var orders []models.Order
	for rows.Next() {
		var OrderUID string
		if err := rows.Scan(&OrderUID); err != nil {
			return nil, fmt.Errorf("Ошибка в UID заказа:%v", err)
		}
		// Получаем полную информацию о заказе
		order, err := p.GetOrder(ctx, OrderUID)
		if err != nil {
			log.Printf("Ошибка при получении заказа %s: %v", OrderUID, err)
			continue // Пропускаем проблемный заказ
		}
		orders = append(orders, *order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Ошибка перебора заказов: %v", err)
	}
	return orders, nil
}

// Close закрывает соединение с базой данных
func (p *Postgres) Close() {
	p.pool.Close()
}
