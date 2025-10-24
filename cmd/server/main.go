// Основной пакет сервера заказов
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"test_service/internal/config"
	"test_service/internal/database"
	"test_service/internal/handler"
	"test_service/internal/kafka"
	"test_service/internal/retry"
	"test_service/internal/service"
)

func main() {
	// Создаем основной контекст
	ctx := context.Background()

	// Загружаем конфигурацию из окружения
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключение к базе данных с retry
	log.Println("Подключение к БД...")
	var db *database.Postgres
	err = retry.DoWithContext(ctx, retry.HeavyPolicy(), func(ctx context.Context) error {
		var dbErr error
		db, dbErr = database.NewPostgres(ctx, cfg.PostgresDSN)
		if dbErr != nil {
			log.Printf("Ошибка подключения к БД (попытка будет повторена): %v", dbErr)
			return dbErr
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Ошибка подключения к БД после всех попыток: %v", err)
	}
	defer db.Close()

	// Инициализация базы данных (создание таблиц) с retry
	err = retry.DoWithContext(ctx, retry.HeavyPolicy(), func(ctx context.Context) error {
		err := db.Init(ctx)
		if err != nil {
			log.Printf("Ошибка инициализации БД (попытка будет повторена): %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Ошибка инициализации БД после всех попыток: %v", err)
	}

	// Создание сервиса для работы с заказами
	svc := service.New(db)

	// Прогрев кэша перед запуском обработчиков с retry
	err = retry.DoWithContext(ctx, retry.DefaultPolicy(), func(ctx context.Context) error {
		err := svc.WarmUpCache(ctx)
		if err != nil {
			log.Printf("Ошибка прогрева кэша (попытка будет повторена): %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Printf("Ошибка прогрева кэша после всех попыток: %v", err)
	}

	// Создание DLQ producer для обработки неудачных сообщений
	dlqTopic := cfg.KafkaTopic + "-dlq" // Используем топик-оригинал с суффиксом DLQ
	dlqProducer := kafka.NewDLQProducer(cfg.KafkaBrokers, dlqTopic)
	defer func() {
		if err := dlqProducer.Close(); err != nil {
			log.Printf("Ошибка при закрытии DLQ producer: %v", err)
		}
	}()

	// Создание Kafka consumer для обработки новых заказов с DLQ
	kafkaConsumer := kafka.NewConsumerWithDLQ(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID, dlqProducer)
	defer func() {
		if err := kafkaConsumer.Close(); err != nil {
			log.Printf("Ошибка при закрытии Kafka consumer: %v", err)
		}
	}()

	// Создание Kafka producer для демонстрации поступления новых заказов
	kafkaProducer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Printf("Ошибка при закрытии Kafka producer: %v", err)
		}
	}()

	// Контекст для управления Kafka consumer
	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	// Запуск Kafka consumer в отдельной горутине
	consumerDone := make(chan struct{})
	go func() {
		log.Printf("Начало работы Kafka consumer для: %s", cfg.KafkaTopic)
		if err := kafkaConsumer.Consume(consumerCtx, svc.ProcessOrder); err != nil {
			log.Printf("Ошибка работы в Kafka consumer: %v", err)
		}
		close(consumerDone)
	}()

	// Запуск Kafka producer в отдельной горутине для демонстрации поступления заказов
	producerCtx, cancelProducer := context.WithCancel(ctx)
	defer cancelProducer()

	producerDone := make(chan struct{})
	go func() {
		log.Printf("Начало отправки тестовых заказов в Kafka: %s", cfg.KafkaTopic)
		ticker := time.NewTicker(5 * time.Second) // Отправляем заказ каждые 5 секунд
		defer ticker.Stop()

		orderCounter := 1
		for {
			select {
			case <-producerCtx.Done():
				close(producerDone)
				return
			case <-ticker.C:
				order := kafka.GenerateTestOrder(orderCounter)
				if err := kafkaProducer.SendOrderWithContext(producerCtx, order); err != nil {
					log.Printf("Ошибка отправки тестового заказа: %v", err)
				} else {
					log.Printf("Отправлен тестовый заказ в Kafka: %s", order.OrderUID)
				}
				orderCounter++
			}
		}
	}()

	// Создание HTTP обработчиков
	h := handler.New(svc)

	// Настройка HTTP маршрутов
	mux := http.NewServeMux()
	mux.HandleFunc("/order/", h.GetOrder)    // API для получения заказа
	mux.HandleFunc("/health", h.HealthCheck) // Проверка состояния сервиса
	mux.HandleFunc("/stats", h.Stats)        // Статистика сервиса

	// Статические файлы и корневая страница
	staticFS := http.Dir(cfg.StaticDir)
	log.Printf("Обслуживание статических файлов из: %s", cfg.StaticDir)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(staticFS)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Если запрос корня — сразу index.html
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "index.html"))
			return
		}
		// Проверяем существование файла в STATIC_DIR безопасно
		candidate := filepath.Clean(filepath.Join(cfg.StaticDir, r.URL.Path))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			http.ServeFile(w, r, candidate)
			return
		}
		// Фоллбэк на index.html
		http.ServeFile(w, r, filepath.Join(cfg.StaticDir, "index.html"))
	})

	// Создание HTTP сервера
	server := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: mux,
	}

	// Запуск HTTP сервера в отдельной горутине
	go func() {
		log.Printf("Сервер запущен на %s", cfg.ServerAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка сервера:%v", err)
		}
	}()

	// Ожидание сигнала для graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Println("Остановка сервера")

	// Graceful shutdown с таймаутом 30 секунд
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("ошибка:%v", err)
	}
	cancelConsumer()
	cancelProducer()
	// Дожидаемся завершения consumer и producer
	select {
	case <-consumerDone:
	case <-time.After(10 * time.Second):
		log.Println("Таймаут ожидания остановки consumer")
	}

	select {
	case <-producerDone:
	case <-time.After(5 * time.Second):
		log.Println("Таймаут ожидания остановки producer")
	}

	log.Println("Сервер остановлен успешно")
}
