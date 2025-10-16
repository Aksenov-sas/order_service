Микросервис для приема, хранения и отображения заказов (Go + PostgreSQL + Kafka).

📋 Функциональность

- Прием сообщений из Kafka и валидация полезной нагрузки
- Транзакционное сохранение в PostgreSQL
- Кэш в памяти и прогрев на старте
- REST API и веб-интерфейс
- Грейсфул шатдаун HTTP-сервера и Kafka consumer

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
│   ├── kafka/            # Kafka consumer
│   ├── models/           # Модели и валидация
│   └── service/          # Бизнес-логика и кэш-операции
└── web/static/           # Веб UI (index.html, script.js)

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
- GET / — веб-интерфейс, статика на /static/

Миграции и данные
- При старте выполняется инициализация схемы и таблица schema_migrations
- Начальные данные и пользователь в init.sql (монтируется в контейнер Postgres)

Типичные проблемы и решения
- 404 на / — задайте STATIC_DIR на каталог с index.html (например, ./web/static)
- Dial error к БД — поднимите postgres: docker compose up -d postgres
- Dial error к Kafka — поднимите zookeeper и kafka: docker compose up -d zookeeper kafka
