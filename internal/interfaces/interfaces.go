// Package interfaces содержит интерфейсы для основных сущностей приложения
package interfaces

import (
	"context"

	"test_service/internal/models"
)

// Database интерфейс для работы с базой данных
type Database interface {
	// Init инициализирует базу данных (создает таблицы и т.д.)
	Init(ctx context.Context) error
	
	// SaveOrder сохраняет заказ в базу данных
	SaveOrder(ctx context.Context, order *models.Order) error
	
	// GetOrder получает заказ по его UID из базы данных
	GetOrder(ctx context.Context, orderUID string) (*models.Order, error)
	
	// GetAllOrders получает все заказы из базы данных
	GetAllOrders(ctx context.Context) ([]models.Order, error)
	
	// Close закрывает соединение с базой данных
	Close()
}

// Cache интерфейс для работы с кэшем
type Cache interface {
	// Set добавляет или обновляет заказ в кэше
	Set(order *models.Order)
	
	// Get получает заказ из кэша по его UID
	Get(orderUID string) (*models.Order, bool)
	
	// GetAll возвращает все заказы из кэша
	GetAll() []*models.Order
	
	// LoadFromSlice загружает заказы из слайса в кэш
	LoadFromSlice(orders []models.Order)
	
	// Size возвращает количество заказов в кэше
	Size() int
	
	// Cleanup удаляет истекшие элементы из кэша
	Cleanup()
}

// OrderService интерфейс для сервиса работы с заказами
type OrderService interface {
	// WarmUpCache загружает все заказы из БД в кэш
	WarmUpCache(ctx context.Context) error
	
	// ProcessOrder обрабатывает новый заказ: сохраняет в БД и добавляет в кэш
	ProcessOrder(order *models.Order) error
	
	// GetOrder получает заказ по его UID с использованием кэша и БД
	GetOrder(orderUID string) (*models.Order, error)
	
	// GetCacheStats возвращает статистику работы сервиса
	GetCacheStats() map[string]interface{}
	
	// Close закрывает соединение с базой данных
	Close()
}