package models

import (
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrder_Validate(t *testing.T) {
	// Проверка валидного заказа
	t.Run("ValidOrder", func(t *testing.T) {
		order := &Order{
			OrderUID:        "testorderuid1234567890123456", // 32 буквенно-цифровых символа
			TrackNumber:     "TRACK123",
			Entry:           "EntryTest",
			Locale:          "en",
			CustomerID:      "customer123",
			DeliveryService: "delivery_service",
			ShardKey:        "shard1",
			SMID:            1,
			DateCreated:     time.Now(),
			OOFShard:        "oof_shard",
			Delivery: Delivery{
				Name:    "Test Customer",
				Phone:   "+1234567890",
				Zip:     "12345",
				City:    "Test City",
				Address: "Test Address",
				Region:  "Test Region",
				Email:   "test@example.com",
			},
			Payment: Payment{
				Transaction:  "trans123",
				Currency:     "USD",
				Provider:     "provider_test",
				Amount:       1000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "Test Bank",
				DeliveryCost: 200,
				GoodsTotal:   800,
				CustomFee:    0,
			},
			Items: []Item{
				{
					ChrtID:      1000,
					TrackNumber: "TRACK123",
					Price:       500,
					RID:         "rid123",
					Name:        "Test Item",
					Size:        "M",
					TotalPrice:  500,
					NMID:        5000,
					Brand:       "Test Brand",
				},
			},
		}

		err := order.Validate()
		assert.NoError(t, err, "валидный заказ не должен возвращать ошибки")
	})

	// Проверка заказа с nil значениями
	t.Run("NilOrder", func(t *testing.T) {
		var order *Order
		err := order.Validate()
		assert.Error(t, err, "валидация nil заказа должна вернуть ошибку")
		assert.Contains(t, err.Error(), "order is nil", "ошибка должна содержать 'order is nil'")
	})

	// Проверка заказа с отсутствующими обязательными полями
	t.Run("MissingRequiredFields", func(t *testing.T) {
		testCases := []struct {
			name        string
			modifyOrder func(*Order)
			expectedErr string
		}{
			{
				name: "MissingOrderUID",
				modifyOrder: func(o *Order) {
					o.OrderUID = ""
				},
				expectedErr: "OrderUID",
			},
			{
				name: "MissingTrackNumber",
				modifyOrder: func(o *Order) {
					o.TrackNumber = ""
				},
				expectedErr: "TrackNumber",
			},
			{
				name: "MissingEntry",
				modifyOrder: func(o *Order) {
					o.Entry = ""
				},
				expectedErr: "Entry",
			},
			{
				name: "MissingLocale",
				modifyOrder: func(o *Order) {
					o.Locale = ""
				},
				expectedErr: "Locale",
			},
			{
				name: "MissingCustomerID",
				modifyOrder: func(o *Order) {
					o.CustomerID = ""
				},
				expectedErr: "CustomerID",
			},
			{
				name: "MissingDeliveryService",
				modifyOrder: func(o *Order) {
					o.DeliveryService = ""
				},
				expectedErr: "DeliveryService",
			},
			{
				name: "MissingShardKey",
				modifyOrder: func(o *Order) {
					o.ShardKey = ""
				},
				expectedErr: "ShardKey",
			},
			{
				name: "MissingOOFShard",
				modifyOrder: func(o *Order) {
					o.OOFShard = ""
				},
				expectedErr: "OOFShard",
			},
			{
				name: "ZeroSMID",
				modifyOrder: func(o *Order) {
					o.SMID = 0
				},
				expectedErr: "SMID",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				order := &Order{
					OrderUID:        "testorderuid1234567890123456", // 32 буквенно-цифровых символа
					TrackNumber:     "TRACK123",
					Entry:           "EntryTest",
					Locale:          "en",
					CustomerID:      "customer123",
					DeliveryService: "delivery_service",
					ShardKey:        "shard1",
					SMID:            1,
					DateCreated:     time.Now(),
					OOFShard:        "oof_shard",
					Delivery: Delivery{
						Name:    "Test Customer",
						Phone:   "+1234567890",
						Zip:     "12345",
						City:    "Test City",
						Address: "Test Address",
						Region:  "Test Region",
						Email:   "test@example.com",
					},
					Payment: Payment{
						Transaction:  "trans123",
						Currency:     "USD",
						Provider:     "provider_test",
						Amount:       1000,
						PaymentDT:    time.Now().Unix(),
						Bank:         "Test Bank",
						DeliveryCost: 200,
						GoodsTotal:   800,
						CustomFee:    0,
					},
					Items: []Item{
						{
							ChrtID:      1000,
							TrackNumber: "TRACK123",
							Price:       500,
							RID:         "rid123",
							Name:        "Test Item",
							Size:        "M",
							TotalPrice:  500,
							NMID:        5000,
							Brand:       "Test Brand",
						},
					},
				}

				tc.modifyOrder(order)
				err := order.Validate()
				assert.Error(t, err, "валидация заказа с отсутствующим полем должна вернуть ошибку")
				assert.Contains(t, err.Error(), tc.expectedErr, "ошибка должна содержать ожидаемый текст")
			})
		}
	})

	// Проверка недействительной доставки
	t.Run("InvalidDelivery", func(t *testing.T) {
		order := &Order{
			OrderUID:        "testorderuid1234567890123456", // 32 буквенно-цифровых символа
			TrackNumber:     "TRACK123",
			Entry:           "EntryTest",
			Locale:          "en",
			CustomerID:      "customer123",
			DeliveryService: "delivery_service",
			ShardKey:        "shard1",
			SMID:            1,
			DateCreated:     time.Now(),
			OOFShard:        "oof_shard",
			Delivery: Delivery{
				Name:    "",
				Phone:   "+1234567890",
				Zip:     "12345",
				City:    "Test City",
				Address: "Test Address",
				Region:  "Test Region",
				Email:   "test@example.com",
			},
			Payment: Payment{
				Transaction:  "trans123",
				Currency:     "USD",
				Provider:     "provider_test",
				Amount:       1000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "Test Bank",
				DeliveryCost: 200,
				GoodsTotal:   800,
				CustomFee:    0,
			},
			Items: []Item{
				{
					ChrtID:      1000,
					TrackNumber: "TRACK123",
					Price:       500,
					RID:         "rid123",
					Name:        "Test Item",
					Size:        "M",
					TotalPrice:  500,
					NMID:        5000,
					Brand:       "Test Brand",
				},
			},
		}

		err := order.Validate()
		assert.Error(t, err, "недействительный заказ доставки должен возвращать ошибку")
		assert.Contains(t, err.Error(), "delivery.name", "ошибка должна содержать 'delivery.name'")
	})

	// Проверка недействительного платежа
	t.Run("InvalidPayment", func(t *testing.T) {
		order := &Order{
			OrderUID:        "testorderuid1234567890123456", // 32 буквенно-цифровых символа
			TrackNumber:     "TRACK123",
			Entry:           "EntryTest",
			Locale:          "en",
			CustomerID:      "customer123",
			DeliveryService: "delivery_service",
			ShardKey:        "shard1",
			SMID:            1,
			DateCreated:     time.Now(),
			OOFShard:        "oof_shard",
			Delivery: Delivery{
				Name:    "Test Customer",
				Phone:   "+1234567890",
				Zip:     "12345",
				City:    "Test City",
				Address: "Test Address",
				Region:  "Test Region",
				Email:   "test@example.com",
			},
			Payment: Payment{
				Transaction:  "",
				Currency:     "USD",
				Provider:     "provider_test",
				Amount:       1000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "Test Bank",
				DeliveryCost: 200,
				GoodsTotal:   800,
				CustomFee:    0,
			},
			Items: []Item{
				{
					ChrtID:      1000,
					TrackNumber: "TRACK123",
					Price:       500,
					RID:         "rid123",
					Name:        "Test Item",
					Size:        "M",
					TotalPrice:  500,
					NMID:        5000,
					Brand:       "Test Brand",
				},
			},
		}

		err := order.Validate()
		assert.Error(t, err, "недействительный заказ платежа должен возвращать ошибку")
		assert.Contains(t, err.Error(), "payment.transaction", "ошибка должна содержать 'payment.transaction'")
	})

	// Проверка недействительных товаров
	t.Run("InvalidItems", func(t *testing.T) {
		order := &Order{
			OrderUID:        "testorderuid1234567890123456", // 32 буквенно-цифровых символа
			TrackNumber:     "TRACK123",
			Entry:           "EntryTest",
			Locale:          "en",
			CustomerID:      "customer123",
			DeliveryService: "delivery_service",
			ShardKey:        "shard1",
			SMID:            1,
			DateCreated:     time.Now(),
			OOFShard:        "oof_shard",
			Delivery: Delivery{
				Name:    "Test Customer",
				Phone:   "+1234567890",
				Zip:     "12345",
				City:    "Test City",
				Address: "Test Address",
				Region:  "Test Region",
				Email:   "test@example.com",
			},
			Payment: Payment{
				Transaction:  "trans123",
				Currency:     "USD",
				Provider:     "provider_test",
				Amount:       1000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "Test Bank",
				DeliveryCost: 200,
				GoodsTotal:   800,
				CustomFee:    0,
			},
			Items: []Item{
				{
					ChrtID:      0, // Неверно - должен быть ненулевым
					TrackNumber: "TRACK123",
					Price:       500,
					RID:         "rid123",
					Name:        "Test Item",
					Size:        "M",
					TotalPrice:  500,
					NMID:        5000,
					Brand:       "Test Brand",
				},
			},
		}

		err := order.Validate()
		assert.Error(t, err, "недействительный товар заказа должен возвращать ошибку")
		assert.Contains(t, err.Error(), "chrt_id", "ошибка должна содержать 'chrt_id'")
	})
}

func TestDelivery_Validate(t *testing.T) {
	// Проверка валидной доставки
	t.Run("ValidDelivery", func(t *testing.T) {
		delivery := &Delivery{
			Name:    "Test Customer",
			Phone:   "+1234567890",
			Zip:     "12345",
			City:    "Test City",
			Address: "Test Address",
			Region:  "Test Region",
			Email:   "test@example.com",
		}

		err := delivery.Validate()
		assert.NoError(t, err, "валидная доставка не должна возвращать ошибки")
	})

	// Проверка доставки с отсутствующими обязательными полями
	t.Run("MissingRequiredFields", func(t *testing.T) {
		testCases := []struct {
			name         string
			modifyDelivery func(*Delivery)
			expectedErr  string
		}{
			{
				name: "MissingName",
				modifyDelivery: func(d *Delivery) {
					d.Name = ""
				},
				expectedErr: "delivery.name",
			},
			{
				name: "MissingPhone",
				modifyDelivery: func(d *Delivery) {
					d.Phone = ""
				},
				expectedErr: "delivery.phone",
			},
			{
				name: "MissingZip",
				modifyDelivery: func(d *Delivery) {
					d.Zip = ""
				},
				expectedErr: "delivery.zip",
			},
			{
				name: "MissingCity",
				modifyDelivery: func(d *Delivery) {
					d.City = ""
				},
				expectedErr: "delivery.city",
			},
			{
				name: "MissingAddress",
				modifyDelivery: func(d *Delivery) {
					d.Address = ""
				},
				expectedErr: "delivery.address",
			},
			{
				name: "MissingRegion",
				modifyDelivery: func(d *Delivery) {
					d.Region = ""
				},
				expectedErr: "delivery.region",
			},
			{
				name: "MissingEmail",
				modifyDelivery: func(d *Delivery) {
					d.Email = ""
				},
				expectedErr: "delivery.email",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				delivery := &Delivery{
					Name:    "Test Customer",
					Phone:   "+1234567890",
					Zip:     "12345",
					City:    "Test City",
					Address: "Test Address",
					Region:  "Test Region",
					Email:   "test@example.com",
				}

				tc.modifyDelivery(delivery)
				err := delivery.Validate()
				assert.Error(t, err, "валидация доставки с отсутствующим полем должна вернуть ошибку")
				assert.Contains(t, err.Error(), tc.expectedErr, "ошибка должна содержать ожидаемый текст")
			})
		}
	})
}

func TestPayment_Validate(t *testing.T) {
	// Проверка валидного платежа
	t.Run("ValidPayment", func(t *testing.T) {
		payment := &Payment{
			Transaction:  "trans123",
			Currency:     "USD",
			Provider:     "provider_test",
			Amount:       1000,
			PaymentDT:    time.Now().Unix(),
			Bank:         "Test Bank",
			DeliveryCost: 200,
			GoodsTotal:   800,
			CustomFee:    0,
		}

		err := payment.Validate()
		assert.NoError(t, err, "валидный платеж не должен возвращать ошибки")
	})

	// Проверка платежа с отсутствующими обязательными полями
	t.Run("MissingRequiredFields", func(t *testing.T) {
		testCases := []struct {
			name        string
			modifyPayment func(*Payment)
			expectedErr string
		}{
			{
				name: "MissingTransaction",
				modifyPayment: func(p *Payment) {
					p.Transaction = ""
				},
				expectedErr: "payment.transaction",
			},
			{
				name: "MissingCurrency",
				modifyPayment: func(p *Payment) {
					p.Currency = ""
				},
				expectedErr: "payment.currency",
			},
			{
				name: "MissingProvider",
				modifyPayment: func(p *Payment) {
					p.Provider = ""
				},
				expectedErr: "payment.provider",
			},
			{
				name: "MissingBank",
				modifyPayment: func(p *Payment) {
					p.Bank = ""
				},
				expectedErr: "payment.bank",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				payment := &Payment{
					Transaction:  "trans123",
					Currency:     "USD",
					Provider:     "provider_test",
					Amount:       1000,
					PaymentDT:    time.Now().Unix(),
					Bank:         "Test Bank",
					DeliveryCost: 200,
					GoodsTotal:   800,
					CustomFee:    0,
				}

				tc.modifyPayment(payment)
				err := payment.Validate()
				assert.Error(t, err, "валидация платежа с отсутствующим полем должна вернуть ошибку")
				assert.Contains(t, err.Error(), tc.expectedErr, "ошибка должна содержать ожидаемый текст")
			})
		}
	})

	// Проверка недействительных сумм
	t.Run("InvalidAmounts", func(t *testing.T) {
		testCases := []struct {
			name        string
			modifyPayment func(*Payment)
			expectedErr string
		}{
			{
				name: "NegativeAmount",
				modifyPayment: func(p *Payment) {
					p.Amount = -100
				},
				expectedErr: "amount",
			},
			{
				name: "ZeroPaymentDT",
				modifyPayment: func(p *Payment) {
					p.PaymentDT = 0
				},
				expectedErr: "payment_dt",
			},
			{
				name: "NegativePaymentDT",
				modifyPayment: func(p *Payment) {
					p.PaymentDT = -1
				},
				expectedErr: "payment_dt",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				payment := &Payment{
					Transaction:  "trans123",
					Currency:     "USD",
					Provider:     "provider_test",
					Amount:       1000,
					PaymentDT:    time.Now().Unix(),
					Bank:         "Test Bank",
					DeliveryCost: 200,
					GoodsTotal:   800,
					CustomFee:    0,
				}

				tc.modifyPayment(payment)
				err := payment.Validate()
				assert.Error(t, err, "валидация платежа с недействительной суммой должна вернуть ошибку")
				assert.Contains(t, err.Error(), tc.expectedErr, "ошибка должна содержать ожидаемый текст")
			})
		}
	})
}

func TestItem_Validate(t *testing.T) {
	// Проверка валидного товара
	t.Run("ValidItem", func(t *testing.T) {
		item := &Item{
			ChrtID:      1000,
			TrackNumber: "TRACK123",
			Price:       500,
			RID:         "rid123",
			Name:        "Test Item",
			Size:        "M",
			TotalPrice:  500,
			NMID:        5000,
			Brand:       "Test Brand",
		}

		err := item.Validate()
		assert.NoError(t, err, "валидный товар не должен возвращать ошибки")
	})

	// Проверка товара с отсутствующими обязательными полями
	t.Run("MissingRequiredFields", func(t *testing.T) {
		testCases := []struct {
			name      string
			modifyItem func(*Item)
			expectedErr string
		}{
			{
				name: "MissingTrackNumber",
				modifyItem: func(i *Item) {
					i.TrackNumber = ""
				},
				expectedErr: "track_number",
			},
			{
				name: "MissingRID",
				modifyItem: func(i *Item) {
					i.RID = ""
				},
				expectedErr: "rid",
			},
			{
				name: "MissingName",
				modifyItem: func(i *Item) {
					i.Name = ""
				},
				expectedErr: "name",
			},
			{
				name: "MissingSize",
				modifyItem: func(i *Item) {
					i.Size = ""
				},
				expectedErr: "size",
			},
			{
				name: "MissingBrand",
				modifyItem: func(i *Item) {
					i.Brand = ""
				},
				expectedErr: "brand",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				item := &Item{
					ChrtID:      1000,
					TrackNumber: "TRACK123",
					Price:       500,
					RID:         "rid123",
					Name:        "Test Item",
					Size:        "M",
					TotalPrice:  500,
					NMID:        5000,
					Brand:       "Test Brand",
				}

				tc.modifyItem(item)
				err := item.Validate()
				assert.Error(t, err, "валидация товара с отсутствующим полем должна вернуть ошибку")
				assert.Contains(t, err.Error(), tc.expectedErr, "ошибка должна содержать ожидаемый текст")
			})
		}
	})

	// Проверка недействительных числовых полей
	t.Run("InvalidNumericFields", func(t *testing.T) {
		testCases := []struct {
			name      string
			modifyItem func(*Item)
			expectedErr string
		}{
			{
				name: "ZeroChrtID",
				modifyItem: func(i *Item) {
					i.ChrtID = 0
				},
				expectedErr: "chrt_id",
			},
			{
				name: "ZeroNMID",
				modifyItem: func(i *Item) {
					i.NMID = 0
				},
				expectedErr: "nm_id",
			},
			{
				name: "NegativePrice",
				modifyItem: func(i *Item) {
					i.Price = -100
				},
				expectedErr: "price",
			},
			{
				name: "NegativeTotalPrice",
				modifyItem: func(i *Item) {
					i.TotalPrice = -100
				},
				expectedErr: "total_price",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				item := &Item{
					ChrtID:      1000,
					TrackNumber: "TRACK123",
					Price:       500,
					RID:         "rid123",
					Name:        "Test Item",
					Size:        "M",
					TotalPrice:  500,
					NMID:        5000,
					Brand:       "Test Brand",
				}

				tc.modifyItem(item)
				err := item.Validate()
				assert.Error(t, err, "валидация товара с недействительным числовым полем должна вернуть ошибку")
				assert.Contains(t, err.Error(), tc.expectedErr, "ошибка должна содержать ожидаемый текст")
			})
		}
	})
}

func TestGenerateValidOrders(t *testing.T) {
	// Проверка, что сгенерированные фейкером заказы действительны
	t.Run("FakerGeneratedOrdersAreValid", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			order := &Order{}
			err := faker.FakeData(order)
			require.NoError(t, err, "фейкер не должен возвращать ошибки")

			// Вручную устанавливаем обязательные поля, которые могут не быть установлены фейкером
			if order.OrderUID == "" {
				order.OrderUID = "test-order-" + string(rune('A'+i))
			}
			if order.TrackNumber == "" {
				order.TrackNumber = "TRACK" + string(rune('0'+i))
			}
			if order.Entry == "" {
				order.Entry = "ENTRY"
			}
			if order.Locale == "" {
				order.Locale = "en"
			}
			if order.CustomerID == "" {
				order.CustomerID = "cust" + string(rune('0'+i))
			}
			if order.DeliveryService == "" {
				order.DeliveryService = "delivery"
			}
			if order.ShardKey == "" {
				order.ShardKey = "shard"
			}
			if order.OOFShard == "" {
				order.OOFShard = "oof"
			}
			if order.SMID == 0 {
				order.SMID = i + 1
			}

			// Проверяем заказ
			err = order.Validate()
			assert.NoError(t, err, "Сгенерированный заказ должен быть действительным")
		}
	})
}