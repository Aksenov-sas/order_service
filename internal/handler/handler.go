// Package handler содержит HTTP обработчики для API
package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"test_service/internal/models"
)

// OrderService определяет интерфейс для работы с заказами
type OrderService interface {
	GetOrder(orderUID string) (*models.Order, error) // Получить заказ по UID
	GetCacheStats() map[string]interface{}           // Получить статистику кэша
}

// Handler содержит HTTP обработчики для API
type Handler struct {
	service OrderService // Сервис для работы с заказами
}

// New создает новый экземпляр HTTP обработчика
func New(service OrderService) *Handler {
	return &Handler{service: service}
}

// GetOrder обрабатывает HTTP запрос для получения заказа по UID
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// Извлекаем order_uid из URL пути (убираем префикс "/order/")
	path := strings.TrimPrefix(r.URL.Path, "/order/")
	if path == "" {
		http.Error(w, "Требуется идентификатор заказа", http.StatusBadRequest)
		return
	}

	// Получаем заказ через сервис
	order, err := h.service.GetOrder(path)
	if err != nil {
		http.Error(w, "Заказ не найден", http.StatusNotFound)
		return
	}

	// Возвращаем заказ в формате JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// HealthCheck обрабатывает запрос проверки состояния сервиса
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",        // Статус сервиса
		"timestamp": time.Now().UTC(), // Текущее время
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Stats обрабатывает запрос для получения статистики сервиса
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := h.service.GetCacheStats() // Получаем статистику от сервиса
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} // Возвращаем статистику в формате JSON
}
