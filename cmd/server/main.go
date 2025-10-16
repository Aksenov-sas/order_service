// Основной пакет сервера заказов
package main

import (
	"context"
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

	// Подключение к базе данных
	log.Println("Подключение к БД...")
	db, err := database.NewPostgres(ctx, cfg.PostgresDSN)
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

	// Прогрев кэша перед запуском обработчиков
	if err := svc.WarmUpCache(ctx); err != nil {
		log.Printf("Ошибка прогрева кэша: %v", err)
	}

	// Создание Kafka consumer для обработки новых заказов
	kafkaConsumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	defer kafkaConsumer.Close()

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
	// Дожидаемся завершения consumer
	select {
	case <-consumerDone:
	case <-time.After(10 * time.Second):
		log.Println("Таймаут ожидания остановки consumer")
	}
	log.Println("Сервер остановлен успешно")
}
