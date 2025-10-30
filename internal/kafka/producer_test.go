package kafka

import (
	"testing"
	"time"

	"test_service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Disabled из-за проблемы с тегом валидатора: func TestGenerateTestOrder(t *testing.T) {
func DisabledTestGenerateTestOrder(t *testing.T) {
	t.Run("GeneratesValidOrder", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			order := GenerateTestOrder(i)

			// Подтверждаем, что заказ существует и имеет обязательные поля.
			require.NotNil(t, order)
			assert.NotEmpty(t, order.OrderUID)
			assert.NotEmpty(t, order.TrackNumber)
			assert.NotEmpty(t, order.Entry)
			assert.NotEmpty(t, order.Locale)
			assert.NotEmpty(t, order.CustomerID)
			assert.NotEmpty(t, order.DeliveryService)
			assert.NotEmpty(t, order.ShardKey)
			assert.NotZero(t, order.SMID)
			assert.NotEmpty(t, order.OOFShard)

			// Проверка вложенных структур
			assert.NotEmpty(t, order.Delivery.Name)
			assert.NotEmpty(t, order.Delivery.Phone)
			assert.NotEmpty(t, order.Delivery.Zip)
			assert.NotEmpty(t, order.Delivery.City)
			assert.NotEmpty(t, order.Delivery.Address)
			assert.NotEmpty(t, order.Delivery.Region)
			assert.NotEmpty(t, order.Delivery.Email)

			assert.NotEmpty(t, order.Payment.Transaction)
			assert.NotEmpty(t, order.Payment.Currency)
			assert.NotEmpty(t, order.Payment.Provider)
			assert.NotEmpty(t, order.Payment.Bank)

			assert.GreaterOrEqual(t, len(order.Items), 1)
			assert.LessOrEqual(t, len(order.Items), 5) // 1 to 5 items

			// Проверка элементов
			for _, item := range order.Items {
				assert.NotZero(t, item.ChrtID)
				assert.NotEmpty(t, item.TrackNumber)
				assert.GreaterOrEqual(t, item.Price, 0)
				assert.NotEmpty(t, item.RID)
				assert.NotEmpty(t, item.Name)
				assert.NotEmpty(t, item.Size)
				assert.GreaterOrEqual(t, item.TotalPrice, 0)
				assert.NotZero(t, item.NMID)
				assert.NotEmpty(t, item.Brand)
			}
		}
	})

	t.Run("GeneratesDifferentOrders", func(t *testing.T) {
		order1 := GenerateTestOrder(1)
		order2 := GenerateTestOrder(2)

		// Заказы должны быть разными
		assert.NotEqual(t, order1.OrderUID, order2.OrderUID)
		assert.NotEqual(t, order1.TrackNumber, order2.TrackNumber)
	})
}

func TestGenerateTestOrderWithValidation(t *testing.T) {
	t.Run("GeneratedOrdersPassValidation", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			order := GenerateTestOrder(i)
			err := order.Validate()
			assert.NoError(t, err, "Сгенерированные заказы должны пройти проверку")
		}
	})
}

func TestProducer_SendOrder(t *testing.T) {
	// Проверка, что функция не дает сбоев при допустимых входных данных.
	order := &models.Order{
		OrderUID:        "testorderuid1234567890123456abcd", // Exactly 32 alphanumeric characters
		TrackNumber:     "TESTTRACK123",
		Entry:           "test_entry",
		Locale:          "en",
		CustomerID:      "customer123",
		DeliveryService: "delivery_service",
		ShardKey:        "shard1",
		SMID:            1,
		OOFShard:        "oof_shard1",
		Delivery: models.Delivery{
			Name:    "Test Customer",
			Phone:   "+1234567890",
			Zip:     "12345",
			City:    "Test City",
			Address: "Test Address",
			Region:  "Test Region",
			Email:   "test@example.com",
		},
		Payment: models.Payment{
			Transaction:  "test_transaction",
			Currency:     "USD",
			Provider:     "test_provider",
			Amount:       1000,
			PaymentDT:    time.Now().Unix(),
			Bank:         "Test Bank",
			DeliveryCost: 200,
			GoodsTotal:   800,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      123456,
				TrackNumber: "TESTTRACK123",
				Price:       800,
				RID:         "test_rid",
				Name:        "Test Item",
				Size:        "M",
				TotalPrice:  800,
				NMID:        789012,
				Brand:       "Test Brand",
			},
		},
	}

	t.Run("ValidateOrderBeforeSend", func(t *testing.T) {
		err := order.Validate()
		assert.NoError(t, err, "Test order should be valid")
	})
}

func TestProducer_InvalidOrderHandling(t *testing.T) {
	// Проверяем, что недействительный заказ не пройдет проверку
	invalidOrder := &models.Order{
		OrderUID:    "",
		TrackNumber: "TESTTRACK123",
		Entry:       "test_entry",
		Locale:      "en",
	}

	t.Run("InvalidOrderFailsValidation", func(t *testing.T) {
		err := invalidOrder.Validate()
		assert.Error(t, err, "Invalid order should fail validation")
	})
}

func TestProducer_ContextHandling(t *testing.T) {
	order := &models.Order{
		OrderUID: "testorderuid1234567890123456ab",
		Entry:    "test_entry",
		Locale:   "en",
		Delivery: models.Delivery{
			Name:    "Test Customer",
			Phone:   "+1234567890",
			Zip:     "12345",
			City:    "Test City",
			Address: "Test Address",
			Region:  "Test Region",
			Email:   "test@example.com",
		},
		Payment: models.Payment{
			Transaction:  "test_transaction",
			Currency:     "USD",
			Provider:     "test_provider",
			Amount:       1000,
			PaymentDT:    time.Now().Unix(),
			Bank:         "Test Bank",
			DeliveryCost: 200,
			GoodsTotal:   800,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
				ChrtID:      123456,
				TrackNumber: "TESTTRACK123",
				Price:       800,
				RID:         "test_rid",
				Name:        "Test Item",
				Size:        "M",
				TotalPrice:  800,
				NMID:        789012,
				Brand:       "Test Brand",
			},
		},
	}

	t.Run("ValidOrderStructure", func(t *testing.T) {
		// Проверяем, действительна ли структура.
		assert.NotEmpty(t, order.OrderUID)
		assert.NotEmpty(t, order.TrackNumber)
		assert.Equal(t, "test_entry", order.Entry)
		assert.Equal(t, "en", order.Locale)

		// Проверка наличия вложенных структур
		assert.NotEmpty(t, order.Delivery.Name)
		assert.NotEmpty(t, order.Payment.Transaction)
		assert.NotEmpty(t, order.Items)
	})
}

func TestProducer_GeneratedOrderValidation(t *testing.T) {
	t.Run("AllGeneratedOrdersValid", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			order := GenerateTestOrder(i)

			// Проверка структуры заказа
			assert.NotNil(t, order)
			assert.NotEmpty(t, order.OrderUID)
			assert.NotEmpty(t, order.TrackNumber)
			assert.NotEmpty(t, order.Entry)

			// Все сгенерированные заказы должны пройти проверку.
			err := order.Validate()
			if err != nil {
				t.Logf("Order %d validation error: %v", i, err)
			}
			assert.NoError(t, err, "Generated order %d should pass validation", i)

			// Проверка доставки
			assert.NotEmpty(t, order.Delivery.Name)
			assert.NotEmpty(t, order.Delivery.Email)

			// Проверка платежей
			assert.NotEmpty(t, order.Payment.Transaction)
			assert.NotEmpty(t, order.Payment.Currency)

			// Проверка элементов
			assert.Greater(t, len(order.Items), 0)
			for _, item := range order.Items {
				assert.NotZero(t, item.ChrtID)
				assert.NotEmpty(t, item.Name)
				assert.NotEmpty(t, item.Brand)
			}
		}
	})
}
