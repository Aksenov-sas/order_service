// Пакет cache содержит реализацию кэша для хранения заказов в памяти
package cache

import (
	"sync"

	"test_service/internal/models"
)

// Cache представляет кэш для хранения заказов в памяти
type Cache struct {
	mu     sync.RWMutex             // Мьютекс для безопасного доступа
	orders map[string]*models.Order // Словарь заказов по их UID
}

// New создает новый экземпляр кэша
func New() *Cache {
	return &Cache{
		orders: make(map[string]*models.Order), // Инициализируем пустой словарь
	}
}

// Set добавляет или обновляет заказ в кэше
func (c *Cache) Set(order *models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders[order.OrderUID] = order // Сохраняем заказ по его UID
}

// Get получает заказ из кэша по его UID
func (c *Cache) Get(orderUID string) (*models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, exists := c.orders[orderUID] // Возвращаем заказ и флаг существования
	return order, exists
}

// GetAll возвращает все заказы из кэша
func (c *Cache) GetAll() []*models.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Создаем слайс с предварительно выделенной емкостью
	orders := make([]*models.Order, 0, len(c.orders))
	for _, order := range c.orders {
		orders = append(orders, order)
	}
	return orders
}

// LoadFromSlice загружает заказы из слайса в кэш
func (c *Cache) LoadFromSlice(orders []models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Добавляем все заказы из слайса в кэш
	for i := range orders {
		c.orders[orders[i].OrderUID] = &orders[i]
	}
}

// Size возвращает количество заказов в кэше
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.orders)
}
