// Package database содержит логику работы с базой данных PostgreSQL
package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"test_service/internal/models"
	"test_service/internal/retry"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres представляет подключение к базе данных PostgreSQL
type Postgres struct {
	pool    *pgxpool.Pool // Пул соединений с базой данных
	metrics *DBMetrics    // Метрики для мониторинга
}

// NewPostgres создает новое подключение к базе данных PostgreSQL
func NewPostgres(ctx context.Context, connectStr string) (*Postgres, error) {
	// Засекаем время установления подключения
	startTime := time.Now()

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
		pool.Close()
		return nil, fmt.Errorf("Ошибка соединения с БД:%v", err)
	}

	// Инициализируем метрики
	metrics := NewDBMetrics()

	// Запускаем сбор метрик пула соединений в отдельной горутине
	go func() {
		ticker := time.NewTicker(15 * time.Second) // Обновляем каждые 15 секунд
		defer ticker.Stop()
		for range ticker.C {
			if pool == nil {
				return // Пул закрыт
			}
			connStats := pool.Stat()
			metrics.ConnectionOpen.Set(float64(connStats.AcquiredConns()))
			metrics.ConnectionMaxOpen.Set(float64(connStats.MaxConns()))
		}
	}()

	// Зафиксируем время установления подключения
	metrics.ConnectionEstablishDuration.Observe(time.Since(startTime).Seconds())

	return &Postgres{
		pool:    pool,
		metrics: metrics, // Инициализируем метрики
	}, nil
}

// Init инициализирует базу данных, создавая необходимые таблицы и индексы
func (p *Postgres) Init(ctx context.Context) error {
	var err error

	startTime := time.Now()

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
			queryStartTime := time.Now()
			_, err := p.pool.Exec(ctx, query)
			p.metrics.QueryDuration.WithLabelValues("init_create_table").Observe(time.Since(queryStartTime).Seconds())
			if err != nil {
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("init_create_table").Inc()
				return fmt.Errorf("Ошибка выполнения запроса %s: %v", query, err)
			}
		}

		// Простейшая миграционная таблица для детерминированных миграций
		queryStartTime := time.Now()
		if _, err := p.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (id TEXT PRIMARY KEY, applied_at TIMESTAMP NOT NULL DEFAULT NOW())`); err != nil {
			p.metrics.QueryDuration.WithLabelValues("init_create_migrations_table").Observe(time.Since(queryStartTime).Seconds())
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("init_create_migrations_table").Inc()
			return fmt.Errorf("Ошибка создания schema_migrations: %v", err)
		} else {
			p.metrics.QueryDuration.WithLabelValues("init_create_migrations_table").Observe(time.Since(queryStartTime).Seconds())
		}

		type migration struct{ id, sql string }
		migrations := []migration{}
		for _, m := range migrations {
			queryStartTime = time.Now()
			var exists bool
			err := p.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE id=$1)`, m.id).Scan(&exists)
			p.metrics.QueryDuration.WithLabelValues("init_check_migration").Observe(time.Since(queryStartTime).Seconds())
			if err != nil {
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("init_check_migration").Inc()
				return fmt.Errorf("Ошибка проверки миграции %s: %v", m.id, err)
			}
			if exists {
				continue
			}
			queryStartTime = time.Now()
			if _, err := p.pool.Exec(ctx, m.sql); err != nil {
				p.metrics.QueryDuration.WithLabelValues("init_apply_migration").Observe(time.Since(queryStartTime).Seconds())
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("init_apply_migration").Inc()
				return fmt.Errorf("Ошибка применения миграции %s: %v", m.id, err)
			} else {
				p.metrics.QueryDuration.WithLabelValues("init_apply_migration").Observe(time.Since(queryStartTime).Seconds())
			}
			queryStartTime = time.Now()
			if _, err := p.pool.Exec(ctx, `INSERT INTO schema_migrations (id) VALUES ($1)`, m.id); err != nil {
				p.metrics.QueryDuration.WithLabelValues("init_record_migration").Observe(time.Since(queryStartTime).Seconds())
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("init_record_migration").Inc()
				return fmt.Errorf("Ошибка записи миграции %s: %v", m.id, err)
			} else {
				p.metrics.QueryDuration.WithLabelValues("init_record_migration").Observe(time.Since(queryStartTime).Seconds())
			}
			log.Printf("Применена миграция: %s", m.id)
		}

		log.Println("БД инициализирована")
		return nil
	})

	if err != nil {
		p.metrics.ConnectionErrorsTotal.Inc()
	} else {
		p.metrics.InitDuration.Observe(time.Since(startTime).Seconds())
	}

	return err
}

// SaveOrder сохраняет заказ в базу данных в рамках транзакции
func (p *Postgres) SaveOrder(ctx context.Context, order *models.Order) error {
	var err error

	startTime := time.Now()

	// Используем retry механизм для операции сохранения
	retryPolicy := retry.HeavyPolicy() // Используем тяжелую политику для критических операций

	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// Начинаем транзакцию
		tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			p.metrics.TransactionErrorsTotal.Inc()
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
		queryStartTime := time.Now()
		_, err = tx.Exec(ctx, SaveOrderQuery, order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
			order.CustomerID, order.DeliveryService, order.ShardKey, order.SMID, order.DateCreated, order.OOFShard)
		p.metrics.QueryDuration.WithLabelValues("save_order").Observe(time.Since(queryStartTime).Seconds())
		if err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("save_order").Inc()
			return fmt.Errorf("Ошибка при записи заказа: %v", err)
		}

		// Сохраняем информацию о доставке (UPSERT)
		queryStartTime = time.Now()
		_, err = tx.Exec(ctx, SaveDeliveryQuery, order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
			order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
		p.metrics.QueryDuration.WithLabelValues("save_delivery").Observe(time.Since(queryStartTime).Seconds())
		if err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("save_delivery").Inc()
			return fmt.Errorf("Ошибка при записи доставки: %v", err)
		}

		// Сохраняем информацию о платеже (UPSERT)
		queryStartTime = time.Now()
		_, err = tx.Exec(ctx, SavePaymentQuery, order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
			order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
			order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
		p.metrics.QueryDuration.WithLabelValues("save_payment").Observe(time.Since(queryStartTime).Seconds())
		if err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("save_payment").Inc()
			return fmt.Errorf("Ошибка при записи payment: %v", err)
		}

		// Удаляем старые товары заказа (для обновления)
		queryStartTime = time.Now()
		_, err = tx.Exec(ctx, DeleteItemsQuery, order.OrderUID)
		p.metrics.QueryDuration.WithLabelValues("delete_items").Observe(time.Since(queryStartTime).Seconds())
		if err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("delete_items").Inc()
			return fmt.Errorf("Ошибка удаления позиций: %v", err)
		}

		// Добавляем новые товары заказа
		for _, items := range order.Items {
			queryStartTime = time.Now()
			_, err = tx.Exec(ctx, SaveItemQuery, order.OrderUID, items.ChrtID, items.TrackNumber, items.Price, items.RID, items.Name,
				items.Sale, items.Size, items.TotalPrice, items.NMID, items.Brand, items.Status)
			p.metrics.QueryDuration.WithLabelValues("save_item").Observe(time.Since(queryStartTime).Seconds())
			if err != nil {
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("save_item").Inc()
				return fmt.Errorf("Ошибка добавления позиции: %v", err)
			}
		}

		// Коммитим транзакцию
		queryStartTime = time.Now()
		if err := tx.Commit(ctx); err != nil {
			p.metrics.QueryDuration.WithLabelValues("commit_transaction").Observe(time.Since(queryStartTime).Seconds())
			p.metrics.TransactionErrorsTotal.Inc()
			return fmt.Errorf("Ошибка коммита транзакции: %v", err)
		} else {
			p.metrics.QueryDuration.WithLabelValues("commit_transaction").Observe(time.Since(queryStartTime).Seconds())
		}

		// Успешно закоммиченная транзакция не нуждается в откате
		shouldRollback = false
		return nil
	})

	if err != nil {
		p.metrics.FailedSavesTotal.Inc()
	} else {
		p.metrics.SuccessfulSavesTotal.Inc()
		p.metrics.SaveDuration.Observe(time.Since(startTime).Seconds())
	}

	return err
}

// GetOrder получает заказ из базы данных по его UID
func (p *Postgres) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	var order *models.Order
	var err error

	startTime := time.Now()

	// Используем retry механизм для операции получения заказа
	retryPolicy := retry.DefaultPolicy() // Используем стандартную политику для операций чтения

	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		var tempOrder models.Order

		// Получаем все данные заказа за один запрос
		queryStartTime := time.Now()
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
		p.metrics.QueryDuration.WithLabelValues("get_order_by_uid").Observe(time.Since(queryStartTime).Seconds())
		if err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("get_order_by_uid").Inc()
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("Заказ не найден: %v", err) // Не возвращаем как ошибку для повторных попыток
			}
			return fmt.Errorf("Ошибка получения заказа: %v", err)
		}

		// Получаем список товаров заказа
		queryStartTime = time.Now()
		rows, err := p.pool.Query(ctx, GetItemsByOrderUIDQuery, orderUID)
		p.metrics.QueryDuration.WithLabelValues("get_items_by_order_uid").Observe(time.Since(queryStartTime).Seconds())
		if err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("get_items_by_order_uid").Inc()
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
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("get_items_by_order_uid").Inc()
				return fmt.Errorf("Ошибка при чтении items:%v", err)
			}
			tempOrder.Items = append(tempOrder.Items, item)
		}

		// Проверяем ошибки при итерации
		if err := rows.Err(); err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("get_items_by_order_uid").Inc()
			return fmt.Errorf("Ошибка при переборе items: %v", err)
		}

		order = &tempOrder
		return nil
	})

	if err != nil {
		p.metrics.FailedGetsTotal.Inc()
	} else {
		p.metrics.SuccessfulGetsTotal.Inc()
		p.metrics.GetDuration.Observe(time.Since(startTime).Seconds())
	}

	if err != nil {
		return nil, err
	}

	return order, nil
}

// GetAllOrders получает все заказы из базы данных
func (p *Postgres) GetAllOrders(ctx context.Context) ([]models.Order, error) {
	var orders []models.Order
	var err error

	startTime := time.Now()

	// Используем retry механизм для операции получения всех заказов
	retryPolicy := retry.DefaultPolicy() // Используем стандартную политику для операций чтения

	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// Получаем все данные всех заказов за один запрос
		queryStartTime := time.Now()
		rows, err := p.pool.Query(ctx, GetAllOrdersQuery)
		p.metrics.QueryDuration.WithLabelValues("get_all_orders").Observe(time.Since(queryStartTime).Seconds())
		if err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("get_all_orders").Inc()
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
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("get_all_orders").Inc()
				return fmt.Errorf("Ошибка при чтении заказа: %v", err)
			}

			orderMap[order.OrderUID] = &order
			orders = append(orders, order)
		}

		if err := rows.Err(); err != nil {
			p.metrics.QueryErrorsTotal.Inc()
			p.metrics.QueryErrors.WithLabelValues("get_all_orders").Inc()
			return fmt.Errorf("Ошибка перебора заказов: %v", err)
		}

		for i := range orders {
			order := &orders[i]
			queryStartTime = time.Now()
			itemsRows, err := p.pool.Query(ctx, GetItemsByOrderUIDQuery, order.OrderUID)
			p.metrics.QueryDuration.WithLabelValues("get_items_by_order_uid").Observe(time.Since(queryStartTime).Seconds())
			if err != nil {
				p.metrics.QueryErrorsTotal.Inc()
				p.metrics.QueryErrors.WithLabelValues("get_items_by_order_uid").Inc()
				log.Printf("Ошибка при запросе товаров для заказа %s: %v", order.OrderUID, err)
				continue
			}

			// Обрабатываем результаты запроса товаров
			for itemsRows.Next() {
				var item models.Item
				err := itemsRows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name, &item.Sale,
					&item.Size, &item.TotalPrice, &item.NMID, &item.Brand, &item.Status)
				if err != nil {
					p.metrics.QueryErrorsTotal.Inc()
					p.metrics.QueryErrors.WithLabelValues("get_items_by_order_uid").Inc()
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
		p.metrics.FailedGetAllTotal.Inc()
	} else {
		p.metrics.SuccessfulGetAllTotal.Inc()
		p.metrics.GetAllDuration.Observe(time.Since(startTime).Seconds())
	}

	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Close закрывает соединение с базой данных
func (p *Postgres) Close() {
	p.pool.Close()
	// Сбрасываем метрики соединений при закрытии
	p.metrics.ConnectionOpen.Set(0)
}
