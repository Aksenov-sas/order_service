Микросервис для обработки и отображения данных о заказах с использованием Go, PostgreSQL и Kafka.

📋 Функциональность
✅ Прием сообщений из Kafka

✅ Сохранение данных в PostgreSQL

✅ Кэширование в памяти для быстрого доступа

✅ REST API для получения данных о заказах

✅ Веб-интерфейс для просмотра заказов

Архитектура:
test_service/
├── cmd/server/           # Главное приложение
├── internal/             # Внутренние пакеты
│   ├── cache/           # Кэш в памяти
│   ├── database/        # Работа с PostgreSQL
│   ├── handler/         # HTTP обработчики
│   ├── kafka/           # Kafka consumer
│   ├── models/          # Модели данных
│   └── service/         # Бизнес-логика
│── web/static/          # Веб-интерфейс

Старт:
1.git clone <repository-url>
  cd test_service
  go mod download

2.docker-compose up -d

3.go run cmd/server/main.go

База данных инициализирована в init.db, там же создан пользователь и объявлены данные для тестов