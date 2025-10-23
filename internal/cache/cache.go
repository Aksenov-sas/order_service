// Package cache содержит реализацию кэша для хранения заказов в памяти
package cache

import (
	"sync"
	"time"

	"test_service/internal/models"
)

// CachedOrderItem кэшированный заказ со сроком жизни
type CachedOrderItem struct {
	order      *models.Order
	expireTime time.Time
}

// Cache представляет кэш для хранения заказов в памяти
type Cache struct {
	mu     sync.RWMutex                // Мьютекс для безопасного доступа
	orders map[string]*CachedOrderItem // Словарь заказов по их UID с временем истечения
	ttl    time.Duration               // Время жизни элемента кэша
}

// New создает новый экземпляр кэша
func New(ttl time.Duration) *Cache {
	return &Cache{
		orders: make(map[string]*CachedOrderItem), // Инициализируем пустой словарь
		ttl:    ttl,                               // Устанавливаем время жизни
	}
}

// Set добавляет или обновляет заказ в кэше
func (c *Cache) Set(order *models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders[order.OrderUID] = &CachedOrderItem{
		order:      order,
		expireTime: time.Now().Add(c.ttl), // Устанавливаем время истечения
	} // Сохраняем заказ по его UID
}

// Get получает заказ из кэша по его UID
func (c *Cache) Get(orderUID string) (*models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.orders[orderUID] // Проверяем наличие элемента
	if !exists {
		return nil, false
	}

	// Проверяем, не истекло ли время жизни
	if time.Now().After(item.expireTime) {
		return nil, false // Элемент истек, считаем что не существует
	}

	return item.order, true
}

// GetAll возвращает все заказы из кэша
func (c *Cache) GetAll() []*models.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Создаем слайс с предварительно выделенной емкостью
	orders := make([]*models.Order, 0, len(c.orders))
	now := time.Now()
	for _, item := range c.orders {
		// Пропускаем истекшие элементы
		if now.After(item.expireTime) {
			continue
		}
		orders = append(orders, item.order)
	}
	return orders
}

// LoadFromSlice загружает заказы из слайса в кэш
func (c *Cache) LoadFromSlice(orders []models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Добавляем все заказы из слайса в кэш
	for i := range orders {
		c.orders[orders[i].OrderUID] = &CachedOrderItem{
			order:      &orders[i],
			expireTime: time.Now().Add(c.ttl), // Устанавливаем время истечения
		}
	}
}

// Size возвращает количество заказов в кэше
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	count := 0
	for _, item := range c.orders {
		if now.After(item.expireTime) {
			continue // Пропускаем истекшие элементы
		}
		count++
	}
	return count
}

// Cleanup удаляет истекшие элементы из кэша
func (c *Cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.orders {
		if now.After(item.expireTime) {
			delete(c.orders, key)
		}
	}
}
