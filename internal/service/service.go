// Пакет service содержит бизнес-логику приложения для работы с заказами
package service

import (
	"context"
	"log"
	"sync"
	"time"

	"test_service/internal/cache"
	"test_service/internal/database"
	"test_service/internal/models"
)

// Service представляет основной сервис для работы с заказами
type Service struct {
	db    *database.Postgres // Подключение к базе данных PostgreSQL
	cache *cache.Cache       // Кэш для хранения заказов в памяти
	mu    sync.RWMutex       // Мьютекс для безопасного доступа к статистике
	stats struct {
		LastRequestTime     time.Time     // Время последнего запроса
		LastRequestDuration time.Duration // Длительность обработки последнего запроса
	}
}

// New создает новый экземпляр сервиса с инициализированным кэшем
func New(db *database.Postgres) *Service {
	svc := &Service{
		db:    db,
		cache: cache.New(), // Создаем новый кэш
	}
	return svc
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

	// Сохраняем заказ в базу данных
	if err := s.db.SaveOrder(ctx, order); err != nil {
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

// Close закрывает соединение с базой данных
func (s *Service) Close() {
	s.db.Close()
}
