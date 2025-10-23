package database

import (
	"testing"

	"test_service/internal/models"

	"github.com/stretchr/testify/assert"
)

// Проверяем, что наша структура заказа действительна
func TestOrderStructure(t *testing.T) {
	order := &models.Order{
		OrderUID:        "testorderuid123456789012345678",
		TrackNumber:     "TESTTRACK123",
		Entry:           "test_entry",
		Locale:          "en",
		CustomerID:      "customer_id",
		DeliveryService: "delivery_service",
		ShardKey:        "shard_key",
		SMID:            1,
		OOFShard:        "oof_shard",
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
			PaymentDT:    1678886400,
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
				Sale:        0,
				Size:        "M",
				TotalPrice:  800,
				NMID:        789012,
				Brand:       "Test Brand",
				Status:      1,
			},
		},
	}

	// Проверяем структуру
	assert.NotNil(t, order)
	assert.Equal(t, "testorderuid123456789012345678", order.OrderUID)
	assert.Equal(t, "TESTTRACK123", order.TrackNumber)
	assert.Len(t, order.Items, 1)
	assert.Equal(t, "Test Item", order.Items[0].Name)
}
