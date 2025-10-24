// Package service содержит бизнес-логику приложения для работы с заказами
package service

import (
	"context"
	"log"
	"sync"
	"time"

	"test_service/internal/cache"
	"test_service/internal/interfaces"
	"test_service/internal/models"
	"test_service/internal/retry"
)

// Service представляет основной сервис для работы с заказами
type Service struct {
	db    interfaces.Database // Подключение к базе данных PostgreSQL
	cache interfaces.Cache    // Кэш для хранения заказов в памяти
	mu    sync.RWMutex        // Мьютекс для безопасного доступа к статистике
	stats struct {
		LastRequestTime     time.Time     // Время последнего запроса
		LastRequestDuration time.Duration // Длительность обработки последнего запроса
	}
	cleanupTicker *time.Ticker  // Тикер для периодической очистки кэша
	stopCleanup   chan struct{} // Канал для остановки очистки
}

// New создает новый экземпляр сервиса с инициализированным кэшем
func New(db interfaces.Database) *Service {
	// Создаем конкретный кэш с TTL
	concreteCache := cache.New(30 * time.Minute) // Создаем новый кэш с TTL 30 минут

	svc := &Service{
		db:            db,
		cache:         concreteCache,                    // Присваиваем кэш интерфейсному полю (автоматическое преобразование)
		cleanupTicker: time.NewTicker(10 * time.Minute), // Очистка каждые 10 минут
		stopCleanup:   make(chan struct{}),              // Канал для остановки очистки
	}

	// Запуск фоновой задачи по очистке кэша
	go svc.runCleanup()

	return svc
}

// NewWithCache создает новый экземпляр сервиса с предоставленным кэшем
func NewWithCache(db interfaces.Database, cache interfaces.Cache) *Service {
	svc := &Service{
		db:            db,
		cache:         cache,
		cleanupTicker: time.NewTicker(10 * time.Minute), // Очистка каждые 10 минут
		stopCleanup:   make(chan struct{}),              // Канал для остановки очистки
	}

	// Запуск фоновой задачи по очистке кэша
	go svc.runCleanup()

	return svc
}

// WarmUpCache загружает все заказы из БД в кэш при старте сервиса.
func (s *Service) WarmUpCache(ctx context.Context) error {
	orders, err := s.db.GetAllOrders(ctx)
	if err != nil {
		return err
	}
	// Загружаем в кэш целиком
	s.cache.LoadFromSlice(orders)
	log.Printf("Кэш прогрет: %d заказов", s.cache.Size())
	return nil
}

// ProcessOrder обрабатывает новый заказ: сохраняет в БД и добавляет в кэш
func (s *Service) ProcessOrder(order *models.Order) error {
	// Создаем контекст с таймаутом 10 секунд
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Если дата создания не установлена, устанавливаем текущее время
	if order.DateCreated.IsZero() {
		order.DateCreated = time.Now()
	}

	// Используем retry механизм для операции сохранения в БД
	retryPolicy := retry.HeavyPolicy() // Используем тяжелую политику для критических операций
	
	err := retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// Сохраняем заказ в базу данных
		return s.db.SaveOrder(ctx, order)
	})
	
	if err != nil {
		return err
	}

	// Добавляем заказ в кэш для быстрого доступа
	s.cache.Set(order)

	log.Printf("Заказ обработан %s", order.OrderUID)
	return nil
}

// GetOrder получает заказ по его UID с использованием кэша и БД
func (s *Service) GetOrder(orderUID string) (*models.Order, error) {
	// Засекаем время начала обработки запроса
	start := time.Now()

	// Обновляем время последнего запроса
	s.mu.Lock()
	s.stats.LastRequestTime = time.Now()
	s.mu.Unlock()

	// Сначала пытаемся найти заказ в кэше
	if order, exists := s.cache.Get(orderUID); exists {
		// Заказ найден в кэше - быстрое получение
		s.mu.Lock()
		s.stats.LastRequestDuration = time.Since(start)
		s.mu.Unlock()
		return order, nil
	}

	// Заказ не найден в кэше, ищем в базе данных
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	order, err := s.db.GetOrder(ctx, orderUID)
	if err != nil {
		// Ошибка при получении из БД
		s.mu.Lock()
		s.stats.LastRequestDuration = time.Since(start)
		s.mu.Unlock()
		return nil, err
	}

	// Добавляем заказ в кэш для будущих запросов
	s.cache.Set(order)

	// Обновляем статистику времени обработки
	s.mu.Lock()
	s.stats.LastRequestDuration = time.Since(start)
	s.mu.Unlock()

	return order, nil
}

// GetCacheStats возвращает статистику работы сервиса
func (s *Service) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"cache_size":            s.cache.Size(),                             // Количество элементов в кэше
		"last_request_time":     s.stats.LastRequestTime,                    // Время последнего запроса
		"last_request_duration": s.stats.LastRequestDuration.Milliseconds(), // Длительность последнего запроса в миллисекундах
		"timestamp":             time.Now().UTC(),                           // Текущее время
	}
}

// runCleanup запускает фоновую задачу по очистке кэша
func (s *Service) runCleanup() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.cache.Cleanup() // Очищаем истекшие элементы
		case <-s.stopCleanup:
			return
		}
	}
}

// Close закрывает соединение с базой данных и останавливает очистку кэша
func (s *Service) Close() {
	// Останавливаем тикер очистки
	s.cleanupTicker.Stop()
	close(s.stopCleanup) // Останавливаем фоновую задачу

	s.db.Close()
}
