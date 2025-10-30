package kafka

import (
	"testing"
)

// TestMain обеспечивает правильную инициализацию перед запуском тестов
func TestMain(m *testing.M) {
	// Сброс метрик перед запуском тестов
	ResetMetricsForTest()
	
	// Запуск всех тестов
	m.Run()
}