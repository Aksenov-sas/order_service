Микросервис для приема, хранения и отображения заказов (Go + PostgreSQL + Kafka).

📋 Функциональность

- Прием сообщений из Kafka и валидация полезной нагрузки
- Транзакционное сохранение в PostgreSQL
- Кэш в памяти и прогрев на старте
- REST API и веб-интерфейс
- Грейсфул шатдаун HTTP-сервера и Kafka consumer
- Мониторинг и метрики в формате Prometheus
- Поддержка повторных попыток (retry) для критических операций
- Обработка DLQ (Dead Letter Queue) для неудачных сообщений

Требования
- Go 1.21+
- Docker + Docker Compose

Архитектура
test_service/
├── cmd/server/           # Точка входа HTTP + запуск consumer
├── internal/
│   ├── cache/            # Кэш заказов
│   ├── config/           # Загрузка конфигурации и .env
│   ├── database/         # Подключение к PostgreSQL, миграции, CRUD
│   ├── handler/          # HTTP обработчики
│   ├── interfaces/       # Интерфейсы для инъекции зависимостей
│   ├── kafka/            # Kafka consumer/producer и DLQ
│   ├── models/           # Модели и валидация
│   ├── retry/            # Механизмы повторных попыток
│   └── service/          # Бизнес-логика и кэш-операции
└── web/static/           # Веб UI (index.html, script.js)

Инфраструктура
- PostgreSQL 13 (порт 5433)
- Apache Kafka с Zookeeper (порт 9092)
- Kafka UI для мониторинга (порт 8080)
- Prometheus метрики (порт как у основного сервиса, эндпоинт /metrics)

Переменные окружения
- SERVER_ADDR — адрес HTTP сервера, по умолчанию :8081
- POSTGRES_DSN — строка подключения к БД
- KAFKA_BROKERS — список брокеров, например localhost:9092
- KAFKA_TOPIC — топик Kafka (orders)
- KAFKA_GROUP_ID — группа consumer
- STATIC_DIR — путь к статике (по умолчанию ./web/static)

Пример .env
SERVER_ADDR=:8081
POSTGRES_DSN=host=localhost port=5433 user=postgres password=postgres dbname=order_db sslmode=disable
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=orders
KAFKA_GROUP_ID=order-service-group
STATIC_DIR=./web/static

Запуск инфраструктуры
docker-compose up -d
go run cmd/server/main.go

HTTP эндпоинты
- GET /order/{order_uid} — получить заказ
- GET /health — проверка здоровья
- GET /stats — статистика работы сервиса
- GET /metrics — метрики Prometheus
- GET / — веб-интерфейс, статика на /static/

Метрики
Следующие метрики экспортируются на эндпоинте /metrics:
- db_successful_saves_total - общее количество успешных операций сохранения в БД
- db_failed_saves_total - общее количество неудачных операций сохранения в БД
- db_successful_gets_total - общее количество успешных операций получения из БД
- db_failed_gets_total - общее количество неудачных операций получения из БД
- db_successful_get_all_total - общее количество успешных операций получения всех записей из БД
- db_failed_get_all_total - общее количество неудачных операций получения всех записей из БД
- db_save_duration_seconds - время выполнения операции сохранения в БД
- db_get_duration_seconds - время выполнения операции получения из БД
- db_get_all_duration_seconds - время выполнения операции получения всех записей из БД
- db_init_duration_seconds - время выполнения инициализации БД
- db_connection_errors_total - общее количество ошибок подключения к БД
- db_transaction_errors_total - общее количество ошибок транзакций в БД
- db_query_errors_total - общее количество ошибок запросов к БД
- db_connections_open - количество открытых соединений с БД
- db_connection_acquire_total - количество попыток получения соединения из пула
- db_connection_acquire_duration_seconds - время ожидания получения соединения из пула
- db_connections_max_open - максимальное количество открытых соединений в пуле
- db_query_duration_seconds - время выполнения SQL-запросов, разбитое по типу операции
- db_query_errors_by_operation_total - количество ошибок SQL-запросов, разбитое по типу операции
- db_connection_establish_duration_seconds - время установления подключения к БД
- kafka_messages_sent_total - общее количество отправленных сообщений в Kafka
- kafka_messages_received_total - общее количество полученных сообщений из Kafka
- kafka_failed_sends_total - общее количество неудачных отправок в Kafka
- kafka_failed_receives_total - общее количество неудачных получений из Kafka
- kafka_processing_errors_total - общее количество ошибок обработки Kafka сообщений
- kafka_dlq_messages_sent_total - общее количество сообщений, отправленных в DLQ
- kafka_retry_attempts_total - общее количество попыток повторной отправки Kafka

Миграции и данные
- При старте выполняется инициализация схемы и таблица schema_migrations
- Начальные данные и пользователь в init.sql (монтируется в контейнер Postgres)
- Создается пользователь `order_user` и база данных `order_db`
- Автоматическая инициализация таблиц заказов, доставки, платежей и товаров

Типичные проблемы и решения
- 404 на / — задайте STATIC_DIR на каталог с index.html (например, ./web/static)
- Dial error к БД — поднимите postgres: docker compose up -d postgres
- Dial error к Kafka — поднимите zookeeper и kafka: docker compose up -d zookeeper kafka
