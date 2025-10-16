package config

import (
	"errors"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config содержит конфигурацию сервиса, считанную из переменных окружения
type Config struct {
	ServerAddr   string   // Адрес HTTP сервера, например :8081
	PostgresDSN  string   // Строка подключения к PostgreSQL
	KafkaBrokers []string // Список брокеров Kafka
	KafkaTopic   string   // Топик Kafka
	KafkaGroupID string   // Группа консюмера Kafka
	StaticDir    string   // Путь к статическим файлам
}

// LoadFromEnv загружает конфигурацию из переменных окружения
func LoadFromEnv() (*Config, error) {
	// Автозагрузка .env, если файл есть в рабочей директории
	_ = godotenv.Load()

	cfg := &Config{}

	// HTTP сервер
	if v := strings.TrimSpace(os.Getenv("SERVER_ADDR")); v != "" {
		cfg.ServerAddr = v
	} else {
		cfg.ServerAddr = ":8081"
	}

	//Postgres DSN (секреты из окружения)
	if v := strings.TrimSpace(os.Getenv("POSTGRES_DSN")); v != "" {
		cfg.PostgresDSN = v
	} else {
		cfg.PostgresDSN = "host=localhost port=5433 user=postgres password=postgres dbname=order_db sslmode=disable"
	}

	// Kafka brokers
	if v := strings.TrimSpace(os.Getenv("KAFKA_BROKERS")); v != "" {
		// Разрешаем пробелы после запятой
		parts := strings.Split(v, ",")
		brokers := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				brokers = append(brokers, p)
			}
		}
		cfg.KafkaBrokers = brokers
	} else {
		cfg.KafkaBrokers = []string{"localhost:9092"}
	}

	// Kafka topic
	if v := strings.TrimSpace(os.Getenv("KAFKA_TOPIC")); v != "" {
		cfg.KafkaTopic = v
	} else {
		cfg.KafkaTopic = "orders"
	}

	// Kafka group id
	if v := strings.TrimSpace(os.Getenv("KAFKA_GROUP_ID")); v != "" {
		cfg.KafkaGroupID = v
	} else {
		cfg.KafkaGroupID = "order-service-group"
	}

	// Static dir
	if v := strings.TrimSpace(os.Getenv("STATIC_DIR")); v != "" {
		cfg.StaticDir = v
	} else {
		cfg.StaticDir = "./web/static"
	}

	// Валидация
	if len(cfg.KafkaBrokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS must not be empty")
	}
	if strings.TrimSpace(cfg.KafkaTopic) == "" {
		return nil, errors.New("KAFKA_TOPIC must not be empty")
	}
	if strings.TrimSpace(cfg.KafkaGroupID) == "" {
		return nil, errors.New("KAFKA_GROUP_ID must not be empty")
	}

	return cfg, nil
}
