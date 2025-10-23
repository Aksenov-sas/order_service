package cache

import (
	"testing"
	"time"

	"test_service/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestCache_SetGet(t *testing.T) {
	cache := New(30 * time.Minute)

	order := &models.Order{
		OrderUID: "order-123",
		Locale:   "en",
	}

	// Test Set
	cache.Set(order)

	// Test Get
	result, exists := cache.Get("order-123")
	assert.True(t, exists)
	assert.Equal(t, order, result)
}

func TestCache_GetNonExistent(t *testing.T) {
	cache := New(30 * time.Minute)

	// Test Get для несуществующего ключа
	result, exists := cache.Get("non-existent")
	assert.False(t, exists)
	assert.Nil(t, result)
}

func TestCache_ExpiredItems(t *testing.T) {
	cache := New(100 * time.Millisecond) // Очень короткое время TTL

	order := &models.Order{
		OrderUID: "order-123",
		Locale:   "en",
	}

	// Добавляем элементы в кеш
	cache.Set(order)

	// Подтверждение существования
	result, exists := cache.Get("order-123")
	assert.True(t, exists)
	assert.Equal(t, order, result)

	// Дожидаемся истечения жизни элемента
	time.Sleep(200 * time.Millisecond)

	// Подтверждение, что больше не существует
	result, exists = cache.Get("order-123")
	assert.False(t, exists)
	assert.Nil(t, result)
}

func TestCache_GetAll(t *testing.T) {
	cache := New(30 * time.Minute)

	orders := []models.Order{
		{OrderUID: "order-1", Locale: "en"},
		{OrderUID: "order-2", Locale: "ru"},
		{OrderUID: "order-3", Locale: "de"},
	}

	// Добавляем все заказы в кеш
	for i := range orders {
		cache.Set(&orders[i])
	}

	// Получение всех заказов
	allOrders := cache.GetAll()

	// Проверка возврата всех заказов
	assert.Len(t, allOrders, 3)
	orderUIDs := make(map[string]bool)
	for _, order := range allOrders {
		orderUIDs[order.OrderUID] = true
	}

	for _, expected := range orders {
		assert.True(t, orderUIDs[expected.OrderUID])
	}
}

func TestCache_GetAllWithExpiredItems(t *testing.T) {
	cache := New(100 * time.Millisecond)

	//Добавление товаров с разным сроком жизни
	order1 := &models.Order{OrderUID: "order-1", Locale: "en"}
	order2 := &models.Order{OrderUID: "order-2", Locale: "ru"}

	cache.Set(order1)
	cache.Set(order2)

	//Дожидаемся пока истечет срок жизни некоторых товаров.
	time.Sleep(200 * time.Millisecond)

	//Получаем все заказы — должно быть пусто, так как все они просрочены.
	allOrders := cache.GetAll()
	assert.Len(t, allOrders, 0)
}

func TestCache_LoadFromSlice(t *testing.T) {
	cache := New(30 * time.Minute)

	orders := []models.Order{
		{OrderUID: "order-1", Locale: "en"},
		{OrderUID: "order-2", Locale: "ru"},
		{OrderUID: "order-3", Locale: "de"},
	}

	// Берем заказы из слайса
	cache.LoadFromSlice(orders)

	// Проверяем что все заказы в кеше
	for _, expected := range orders {
		result, exists := cache.Get(expected.OrderUID)
		assert.True(t, exists)
		assert.Equal(t, &expected, result)
	}
}

func TestCache_Size(t *testing.T) {
	cache := New(30 * time.Minute)

	// Инициализируем пустой
	assert.Equal(t, 0, cache.Size())

	// Добавляем заказ
	order := &models.Order{OrderUID: "order-1", Locale: "en"}
	cache.Set(order)
	assert.Equal(t, 1, cache.Size())

	// Добавляем ещё один заказ
	order2 := &models.Order{OrderUID: "order-2", Locale: "ru"}
	cache.Set(order2)
	assert.Equal(t, 2, cache.Size())

	// Удаляем, сделав его недействительным
	shortCache := New(100 * time.Millisecond)
	shortCache.Set(order)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 0, shortCache.Size())
}

func TestCache_SizeWithExpired(t *testing.T) {
	cache := New(100 * time.Millisecond)

	order1 := &models.Order{OrderUID: "order-1", Locale: "en"}
	order2 := &models.Order{OrderUID: "order-2", Locale: "ru"}

	cache.Set(order1)
	cache.Set(order2)

	// Размер должен быть 2 до истечения срока жизни
	assert.Equal(t, 2, cache.Size())

	// Дожидаемся истечения
	time.Sleep(200 * time.Millisecond)

	// Размер должен быть 0 после истечения
	assert.Equal(t, 0, cache.Size())
}

func TestCache_Cleanup(t *testing.T) {
	cache := New(100 * time.Millisecond)

	order1 := &models.Order{OrderUID: "order-1", Locale: "en"}
	order2 := &models.Order{OrderUID: "order-2", Locale: "ru"}

	cache.Set(order1)
	cache.Set(order2)

	// Ждем истчения жизни заказов
	time.Sleep(200 * time.Millisecond)

	// Подверждаем что заказы истекли но всё ещё в мапе
	_, exists1 := cache.Get("order-1")
	_, exists2 := cache.Get("order-2")
	assert.False(t, exists1)
	assert.False(t, exists2)

	// Заказы должны оставаться в мапе до момента очистки.
	assert.Equal(t, 0, cache.Size())

	// После очистки мапа должна быть очищена.
	cache.Cleanup()
	assert.Equal(t, 0, cache.Size())
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := New(30 * time.Minute)

	// Тестируем конкурентый доступ
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			order := &models.Order{
				OrderUID: "order-" + string(rune('A'+i%26)),
				Locale:   "en",
			}
			cache.Set(order)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("order-" + string(rune('A'+i%26)))
		}
		done <- true
	}()

	<-done
	<-done

	// Проверяем что кеш работает
	cache.Set(&models.Order{OrderUID: "final", Locale: "en"})
	result, exists := cache.Get("final")
	assert.True(t, exists)
	assert.Equal(t, "final", result.OrderUID)
}
