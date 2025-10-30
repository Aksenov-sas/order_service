// Package database содержит логику работы с базой данных PostgreSQL
package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"test_service/internal/models"
	"test_service/internal/retry"

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
	var err error
	
	// Используем retry механизм для инициализации базы данных
	retryPolicy := retry.HeavyPolicy() // Используем тяжелую политику для критических операций инициализации
	
	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// SQL запросы для создания таблиц и индексов
		queries := []string{
			// Таблица заказов
			CreateOrdersTable,

			// Таблица доставки
			CreateDeliveryTable,

			// Таблица платежей
			CreatePaymentTable,

			// Таблица товаров
			CreateItemsTable,

			// Индексы для оптимизации запросов
			CreateItemsIndex,
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
	})
	
	return err
}

// SaveOrder сохраняет заказ в базу данных в рамках транзакции
func (p *Postgres) SaveOrder(ctx context.Context, order *models.Order) error {
	var err error

	// Используем retry механизм для операции сохранения
	retryPolicy := retry.HeavyPolicy() // Используем тяжелую политику для критических операций

	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// Начинаем транзакцию
		tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return fmt.Errorf("Ошибка начала транзакции: %v", err)
		}
		
		// Откатываем транзакцию только в случае ошибки
		shouldRollback := true
		defer func() {
			if shouldRollback {
				if err := tx.Rollback(ctx); err != nil {
					log.Printf("Ошибка при откате транзакции: %v", err)
				}
			}
		}()
		
		// Сохраняем основную информацию о заказе (UPSERT)
		_, err = tx.Exec(ctx, SaveOrderQuery, order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
			order.CustomerID, order.DeliveryService, order.ShardKey, order.SMID, order.DateCreated, order.OOFShard)
		if err != nil {
			return fmt.Errorf("Ошибка при записи заказа: %v", err)
		}

		// Сохраняем информацию о доставке (UPSERT)
		_, err = tx.Exec(ctx, SaveDeliveryQuery, order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
			order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
		if err != nil {
			return fmt.Errorf("Ошибка при записи доставки: %v", err)
		}

		// Сохраняем информацию о платеже (UPSERT)
		_, err = tx.Exec(ctx, SavePaymentQuery, order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
			order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
			order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
		if err != nil {
			return fmt.Errorf("Ошибка при записи payment: %v", err)
		}

		// Удаляем старые товары заказа (для обновления)
		_, err = tx.Exec(ctx, DeleteItemsQuery, order.OrderUID)
		if err != nil {
			return fmt.Errorf("Ошибка удаления позиций: %v", err)
		}

		// Добавляем новые товары заказа
		for _, items := range order.Items {
			_, err = tx.Exec(ctx, SaveItemQuery, order.OrderUID, items.ChrtID, items.TrackNumber, items.Price, items.RID, items.Name,
				items.Sale, items.Size, items.TotalPrice, items.NMID, items.Brand, items.Status)
			if err != nil {
				return fmt.Errorf("Ошибка добавления позиции: %v", err)
			}
		}

		// Коммитим транзакцию
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("Ошибка коммита транзакции: %v", err)
		}
		
		// Успешно закоммиченная транзакция не нуждается в откате
		shouldRollback = false
		return nil
	})

	return err
}

// GetOrder получает заказ из базы данных по его UID
func (p *Postgres) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	var order *models.Order
	var err error

	// Используем retry механизм для операции получения заказа
	retryPolicy := retry.DefaultPolicy() // Используем стандартную политику для операций чтения

	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		var tempOrder models.Order

		// Получаем все данные заказа за один запрос
		row := p.pool.QueryRow(ctx, GetOrderByUIDQuery, orderUID)
		err := row.Scan(
			&tempOrder.OrderUID, &tempOrder.TrackNumber, &tempOrder.Entry, &tempOrder.Locale, &tempOrder.InternalSignature,
			&tempOrder.CustomerID, &tempOrder.DeliveryService, &tempOrder.ShardKey, &tempOrder.SMID, &tempOrder.DateCreated, &tempOrder.OOFShard,
			&tempOrder.Delivery.Name, &tempOrder.Delivery.Phone, &tempOrder.Delivery.Zip, &tempOrder.Delivery.City,
			&tempOrder.Delivery.Address, &tempOrder.Delivery.Region, &tempOrder.Delivery.Email,
			&tempOrder.Payment.Transaction, &tempOrder.Payment.RequestID, &tempOrder.Payment.Currency, &tempOrder.Payment.Provider,
			&tempOrder.Payment.Amount, &tempOrder.Payment.PaymentDT, &tempOrder.Payment.Bank, &tempOrder.Payment.DeliveryCost,
			&tempOrder.Payment.GoodsTotal, &tempOrder.Payment.CustomFee,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("Заказ не найден: %v", err) // Не возвращаем как ошибку для повторных попыток
			}
			return fmt.Errorf("Ошибка получения заказа: %v", err)
		}

		// Получаем список товаров заказа
		rows, err := p.pool.Query(ctx, GetItemsByOrderUIDQuery, orderUID)
		if err != nil {
			return fmt.Errorf("Не удалось запросить items: %v", err)
		}
		defer rows.Close()

		// Обрабатываем результаты запроса
		tempOrder.Items = []models.Item{}
		for rows.Next() {
			var item models.Item
			err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name, &item.Sale,
				&item.Size, &item.TotalPrice, &item.NMID, &item.Brand, &item.Status)
			if err != nil {
				return fmt.Errorf("Ошибка при чтении items:%v", err)
			}
			tempOrder.Items = append(tempOrder.Items, item)
		}

		// Проверяем ошибки при итерации
		if err := rows.Err(); err != nil {
			return fmt.Errorf("Ошибка при переборе items: %v", err)
		}

		order = &tempOrder
		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

// GetAllOrders получает все заказы из базы данных
func (p *Postgres) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	var orders []models.Order
	var err error

	// Используем retry механизм для операции получения всех заказов
	retryPolicy := retry.DefaultPolicy() // Используем стандартную политику для операций чтения

	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// Получаем все данные всех заказов за один запрос
		rows, err := p.pool.Query(ctx, GetAllOrdersQuery)
		if err != nil {
			return fmt.Errorf("Ошибка при запросе заказов: %v", err)
		}
		defer rows.Close()

		// Обрабатываем результаты запроса
		orders = make([]models.Order, 0)           // Инициализируем слайс
		orderMap := make(map[string]*models.Order) // To group orders by UID

		for rows.Next() {
			var order models.Order
			err := rows.Scan(
				&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
				&order.CustomerID, &order.DeliveryService, &order.ShardKey, &order.SMID, &order.DateCreated, &order.OOFShard,
				&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
				&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
				&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider,
				&order.Payment.Amount, &order.Payment.PaymentDT, &order.Payment.Bank, &order.Payment.DeliveryCost,
				&order.Payment.GoodsTotal, &order.Payment.CustomFee,
			)
			if err != nil {
				return fmt.Errorf("Ошибка при чтении заказа: %v", err)
			}

			orderMap[order.OrderUID] = &order
			orders = append(orders, order)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("Ошибка перебора заказов: %v", err)
		}

		for i := range orders {
			order := &orders[i]
			itemsRows, err := p.pool.Query(ctx, GetItemsByOrderUIDQuery, order.OrderUID)
			if err != nil {
				log.Printf("Ошибка при запросе товаров для заказа %s: %v", order.OrderUID, err)
				continue
			}

			// Обрабатываем результаты запроса товаров
			for itemsRows.Next() {
				var item models.Item
				err := itemsRows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name, &item.Sale,
					&item.Size, &item.TotalPrice, &item.NMID, &item.Brand, &item.Status)
				if err != nil {
					log.Printf("Ошибка при чтении товара для заказа %s: %v", order.OrderUID, err)
					itemsRows.Close()
					break
				}
				order.Items = append(order.Items, item)
			}
			itemsRows.Close()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Close закрывает соединение с базой данных
func (p *Postgres) Close() {
	p.pool.Close()
}
