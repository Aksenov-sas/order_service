// Основной пакет сервера заказов
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"test_service/internal/database"
	"test_service/internal/handler"
	"test_service/internal/kafka"
	"test_service/internal/service"
)

func main() {
	// Создаем основной контекст
	ctx := context.Background()

	// Настройки подключения к базе данных PostgreSQL
	postgresConnStr := "host=localhost port=5433 user=postgres password=postgres dbname=order_db sslmode=disable"

	// Настройки Kafka
	kafkaBrokers := []string{"localhost:9092"}
	kafkaTopic := "orders"
	kafkaGroupID := "order-service-group"

	// Подключение к базе данных
	log.Println("Подключение к БД...")
	db, err := database.NewPostgres(ctx, postgresConnStr)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.Close()

	// Инициализация базы данных (создание таблиц)
	if err := db.Init(ctx); err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}

	// Создание сервиса для работы с заказами
	svc := service.New(db)

	// Создание Kafka consumer для обработки новых заказов
	kafkaConsumer := kafka.NewConsumer(kafkaBrokers, kafkaTopic, kafkaGroupID)
	defer kafkaConsumer.Close()

	// Контекст для управления Kafka consumer
	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	// Запуск Kafka consumer в отдельной горутине
	go func() {
		log.Printf("Начало работы Kafka consumer для: %s", kafkaTopic)
		if err := kafkaConsumer.Consume(consumerCtx, svc.ProcessOrder); err != nil {
			log.Printf("Ошибка работы в Kafka consumer: %v", err)
		}
	}()

	// Создание HTTP обработчиков
	h := handler.New(svc)

	// Настройка HTTP маршрутов
	mux := http.NewServeMux()
	mux.HandleFunc("/order/", h.GetOrder)    // API для получения заказа
	mux.HandleFunc("/health", h.HealthCheck) // Проверка состояния сервиса
	mux.HandleFunc("/stats", h.Stats)        // Статистика сервиса

	// Статические файлы для веб-интерфейса
	fs := http.FileServer(http.Dir("./web/static"))
	mux.Handle("/", fs)

	// Создание HTTP сервера
	server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	// Запуск HTTP сервера в отдельной горутине
	go func() {
		log.Printf("Сервер запущен на порте 8081")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
	log.Println("Сервер остановлен успешно")
}
