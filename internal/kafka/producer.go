// Package kafka содержит логику для работы с Apache Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"time"

	"test_service/internal/models"
	"test_service/internal/retry"

	"github.com/go-faker/faker/v4"
	"github.com/segmentio/kafka-go"
)

// Producer для отправки сообщений в Kafka
type Producer struct {
	writer  *kafka.Writer // Kafka writer для отправки сообщений
	topic   string        // Топик для отправки
	metrics *KafkaMetrics // Метрики для мониторинга
}

// NewProducer создает нового Kafka продюсера
func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...), // Адреса брокеров Kafka
		Topic:                  topic,                 // Топик для отправки
		Balancer:               &kafka.LeastBytes{},   // Балансировщик по наименьшему количеству байт
		WriteTimeout:           10 * time.Second,      // Таймаут на запись
		ReadTimeout:            10 * time.Second,      // Таймаут на чтение
		RequiredAcks:           kafka.RequireAll,      // Требовать подтверждения от всех реплик
		MaxAttempts:            3,                     // Максимальное количество попыток
		AllowAutoTopicCreation: true,                  // Разрешить автоматическое создание топика
	}
	return &Producer{
		writer:  writer,
		topic:   topic,
		metrics: NewKafkaMetrics(), // Инициализировать метрики
	}
}

// SendOrder отправляет заказ в Kafka с механизмом повторных попыток
func (p *Producer) SendOrder(order *models.Order) error {
	// Валидация заказа перед отправкой
	if err := order.Validate(); err != nil {
		p.metrics.ProcessingErrorsTotal.Inc()
		return fmt.Errorf("ошибка валидации заказа перед отправкой в Kafka: %w", err)
	}

	// Сериализация заказа в JSON
	orderJSON, err := json.Marshal(order)
	if err != nil {
		p.metrics.ProcessingErrorsTotal.Inc()
		return err
	}

	// Создание сообщения для отправки
	msg := kafka.Message{
		Key:   []byte(order.OrderUID), // Использовать OrderUID в качестве ключа
		Value: orderJSON,              // Тело сообщения - JSON заказа
		Time:  time.Now(),             // Временная метка
	}

	// Использовать механизм повторных попыток для отправки сообщения
	retryPolicy := retry.DefaultPolicy()

	err = retry.DoWithContext(context.Background(), retryPolicy, func(ctx context.Context) error {
		// Отправить сообщение в Kafka
		err := p.writer.WriteMessages(ctx, msg)
		if err != nil {
			p.metrics.FailedSendsTotal.Inc()
			p.metrics.RetryAttemptsTotal.Inc()
			log.Printf("Ошибка отправки сообщения в Kafka (будет повторная попытка): %v", err)
			return err
		}
		p.metrics.MessagesSentTotal.Inc()
		return nil
	})

	if err != nil {
		p.metrics.ProcessingErrorsTotal.Inc()
	}

	return err
}

// SendOrderWithContext отправляет заказ в Kafka с контекстом и механизмом повторных попыток
func (p *Producer) SendOrderWithContext(ctx context.Context, order *models.Order) error {
	// Валидация заказа перед отправкой
	if err := order.Validate(); err != nil {
		p.metrics.ProcessingErrorsTotal.Inc()
		return fmt.Errorf("ошибка валидации заказа перед отправкой в Kafka: %w", err)
	}

	// Сериализация заказа в JSON
	orderJSON, err := json.Marshal(order)
	if err != nil {
		p.metrics.ProcessingErrorsTotal.Inc()
		return err
	}

	// Создание сообщения для отправки
	msg := kafka.Message{
		Key:   []byte(order.OrderUID), // Использовать OrderUID в качестве ключа
		Value: orderJSON,              // Тело сообщения - JSON заказа
		Time:  time.Now(),             // Временная метка
	}

	// Использовать механизм повторных попыток для отправки сообщения с контекстом
	retryPolicy := retry.DefaultPolicy()

	err = retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// Отправить сообщение в Kafka
		err := p.writer.WriteMessages(ctx, msg)
		if err != nil {
			p.metrics.FailedSendsTotal.Inc()
			p.metrics.RetryAttemptsTotal.Inc()
			log.Printf("Ошибка отправки сообщения в Kafka с контекстом (будет повторная попытка): %v", err)
			return err
		}
		p.metrics.MessagesSentTotal.Inc()
		return nil
	})

	if err != nil {
		p.metrics.ProcessingErrorsTotal.Inc()
	}

	return err
}

// Close закрывает writer Kafka
func (p *Producer) Close() error {
	return p.writer.Close()
}

// GenerateTestOrder создает тестовый заказ для демонстрации с использованием фейковых данных
func GenerateTestOrder(index int) *models.Order {
	var delivery models.Delivery
	var payment models.Payment
	var items []models.Item

	// Генерация фейковых данных для доставки
	_ = faker.FakeData(&delivery)
	// Установить OrderUID в пустое значение, так как мы устанавливаем его отдельно
	delivery.OrderUID = ""
	// Обеспечить валидность email
	if delivery.Email == "" || !isValidEmail(delivery.Email) {
		delivery.Email = fmt.Sprintf("test%d@example.com", index)
	}

	// Обеспечить, чтобы строковые поля не превышали ограничения базы данных
	if len(delivery.Name) > 255 {
		delivery.Name = delivery.Name[:255]
	}
	if len(delivery.Phone) > 255 {
		delivery.Phone = delivery.Phone[:255]
	}
	if len(delivery.Zip) > 255 {
		delivery.Zip = delivery.Zip[:255]
	}
	if len(delivery.City) > 255 {
		delivery.City = delivery.City[:255]
	}
	if len(delivery.Address) > 255 {
		delivery.Address = delivery.Address[:255]
	}
	if len(delivery.Region) > 255 {
		delivery.Region = delivery.Region[:255]
	}
	if len(delivery.Email) > 255 {
		delivery.Email = delivery.Email[:255]
	}

	// Генерация фейковых данных для оплаты
	_ = faker.FakeData(&payment)
	// Установить OrderUID в пустое значение, так как мы устанавливаем его отдельно
	payment.OrderUID = ""
	// Обеспечить, чтобы PaymentDT было больше 0
	if payment.PaymentDT <= 0 {
		payment.PaymentDT = time.Now().Unix()
	}

	// Обеспечить, чтобы строковые поля не превышали ограничения базы данных
	if len(payment.Currency) > 10 {
		payment.Currency = payment.Currency[:10]
	}
	if len(payment.Provider) > 255 {
		payment.Provider = payment.Provider[:255]
	}
	if len(payment.Bank) > 255 {
		payment.Bank = payment.Bank[:255]
	}
	if len(payment.Transaction) > 255 {
		payment.Transaction = payment.Transaction[:255]
	}
	if len(payment.RequestID) > 255 {
		payment.RequestID = payment.RequestID[:255]
	}

	// Создание фейковых товаров (от 1 до 5 товаров)
	numItems := 1 + index%5 // от 1 до 5 товаров
	for i := 0; i < numItems; i++ {
		var item models.Item
		_ = faker.FakeData(&item)
		item.OrderUID = "" // Установить OrderUID в пустое значение

		// Обеспечить, чтобы цены и ID были положительными
		if item.Price <= 0 {
			item.Price = 100 + (index*10+i*5)%1000
		}
		if item.TotalPrice <= 0 {
			item.TotalPrice = item.Price + (index*5+i*3)%500
		}
		if item.ChrtID <= 0 {
			item.ChrtID = 1000000 + (index*100+i*10)%8000000
		}
		if item.NMID <= 0 {
			item.NMID = 100000000 + (index*1000+i*100)%800000000
		}

		// Обеспечить, чтобы строковые поля не превышали ограничения базы данных
		if len(item.TrackNumber) > 255 {
			item.TrackNumber = item.TrackNumber[:255]
		}
		if len(item.RID) > 255 {
			item.RID = item.RID[:255]
		}
		if len(item.Name) > 255 {
			item.Name = item.Name[:255]
		}
		if len(item.Size) > 255 {
			item.Size = item.Size[:255]
		}
		if len(item.Brand) > 255 {
			item.Brand = item.Brand[:255]
		}

		items = append(items, item)
	}

	// Создание заказа с фейковыми данными, обеспечивая валидный OrderUID (32 буквенно-цифровых символа)
	orderUID := fmt.Sprintf("testorderuid%020d", index)
	orderUID = orderUID[:32] // Обеспечить ровно 32 символа
	// Обеспечить, чтобы строка была буквенно-цифровой
	orderUID = fmt.Sprintf("testorderuid%020d", index)[:32]

	// Генерация фейковых данных для основной структуры заказа
	var order models.Order
	_ = faker.FakeData(&order)

	// Установка конкретных значений, которые должны соответствовать требованиям
	order.OrderUID = orderUID
	order.TrackNumber = fmt.Sprintf("TRACK%010d", index) // Обеспечить, чтобы не было пустым
	order.Entry = "TestEntry"                            // Обеспечить, чтобы не было пустым
	order.Locale = "en"                                  // Обеспечить, чтобы не было пустым и в рамках ограничения длины
	order.InternalSignature = ""
	order.CustomerID = fmt.Sprintf("customer_%d", index) // Обеспечить, чтобы не было пустым
	order.DeliveryService = "delivery_service"           // Обеспечить, чтобы не было пустым
	order.ShardKey = fmt.Sprintf("shard_%d", index)      // Обеспечить, чтобы не было пустым
	order.SMID = 1 + (index % 999999)                    // Обеспечить, чтобы было > 0
	order.DateCreated = time.Now()
	order.OOFShard = fmt.Sprintf("oof_shard_%d", index) // Обеспечить, чтобы не было пустым

	// Назначение связанных структур
	order.Delivery = delivery
	order.Payment = payment
	order.Items = items

	// Обеспечить, чтобы все необходимые поля оплаты были заполнены
	if order.Payment.Transaction == "" {
		order.Payment.Transaction = fmt.Sprintf("trans_%d", index)
	}
	if order.Payment.Currency == "" {
		order.Payment.Currency = "USD"
	}
	if order.Payment.Provider == "" {
		order.Payment.Provider = "provider_test"
	}
	if order.Payment.Bank == "" {
		order.Payment.Bank = "TestBank"
	}
	if order.Payment.PaymentDT <= 0 {
		order.Payment.PaymentDT = time.Now().Unix()
	}

	// Обеспечить валидность важных полей
	if order.Payment.Amount <= 0 {
		order.Payment.Amount = 100 + (index*10)%10000
	}
	if order.Payment.DeliveryCost <= 0 {
		order.Payment.DeliveryCost = 20 + (index*2)%500
	}
	if order.Payment.GoodsTotal <= 0 {
		order.Payment.GoodsTotal = order.Payment.Amount - order.Payment.DeliveryCost
		if order.Payment.GoodsTotal <= 0 {
			order.Payment.GoodsTotal = order.Payment.Amount - 50
		}
	}

	// Валидация сгенерированного заказа
	if err := order.Validate(); err != nil {
		log.Printf("Сгенерированный заказ не прошел валидацию: %v, будет исправлен", err)
	}

	return &order
}

// isValidEmail проверяет, является ли строка валидным email адресом
func isValidEmail(email string) bool {
	if len(email) <= 0 {
		return false
	}

	// Использовать регулярное выражение для валидации email
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
