// Package kafka содержит логику для работы с Apache Kafka
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"test_service/internal/models"
	"test_service/internal/retry"

	"github.com/go-faker/faker/v4"
	"github.com/segmentio/kafka-go"
)

// Producer для отправки сообщений в Kafka
type Producer struct {
	writer *kafka.Writer // Kafka writer для отправки сообщений
	topic  string        // Топик для отправки
}

// NewProducer создает новый Kafka producer
func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...), // Адреса брокеров Kafka
		Topic:                  topic,                 // Топик для отправки
		Balancer:               &kafka.LeastBytes{},   // Балансировщик по наименьшему количеству байт
		WriteTimeout:           10 * time.Second,      // Таймаут на запись
		ReadTimeout:            10 * time.Second,      // Таймаут на чтение
		RequiredAcks:           kafka.RequireAll,      // Подтверждение от всех реплик
		MaxAttempts:            3,                     // Максимальное количество попыток
		AllowAutoTopicCreation: true,                  // Разрешить автосоздание топика
	}
	return &Producer{
		writer: writer,
		topic:  topic,
	}
}

// SendOrder отправляет заказ в Kafka с механизмом повторных попыток
func (p *Producer) SendOrder(order *models.Order) error {
	// Проверяем валидацию заказа перед отправкой
	if err := order.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации заказа перед отправкой в Kafka: %w", err)
	}

	// Сериализуем заказ в JSON
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return err
	}

	// Создаем сообщение для отправки
	msg := kafka.Message{
		Key:   []byte(order.OrderUID), // Используем OrderUID как ключ
		Value: orderJSON,              // Тело сообщения - JSON заказа
		Time:  time.Now(),             // Временная метка
	}

	// Используем retry механизм для отправки сообщения
	retryPolicy := retry.DefaultPolicy()

	return retry.DoWithContext(context.Background(), retryPolicy, func(ctx context.Context) error {
		// Отправляем сообщение в Kafka
		err := p.writer.WriteMessages(ctx, msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения в Kafka (попытка будет повторена): %v", err)
			return err
		}
		return nil
	})
}

// SendOrderWithContext отправляет заказ в Kafka с контекстом и механизмом повторных попыток
func (p *Producer) SendOrderWithContext(ctx context.Context, order *models.Order) error {
	// Проверяем валидацию заказа перед отправкой
	if err := order.Validate(); err != nil {
		return fmt.Errorf("ошибка валидации заказа перед отправкой в Kafka: %w", err)
	}

	// Сериализуем заказ в JSON
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return err
	}

	// Создаем сообщение для отправки
	msg := kafka.Message{
		Key:   []byte(order.OrderUID), // Используем OrderUID как ключ
		Value: orderJSON,              // Тело сообщения - JSON заказа
		Time:  time.Now(),             // Временная метка
	}

	// Используем retry механизм для отправки сообщения с контекстом
	retryPolicy := retry.DefaultPolicy()

	return retry.DoWithContext(ctx, retryPolicy, func(ctx context.Context) error {
		// Отправляем сообщение в Kafka
		err := p.writer.WriteMessages(ctx, msg)
		if err != nil {
			log.Printf("Ошибка отправки сообщения в Kafka с контекстом (попытка будет повторена): %v", err)
			return err
		}
		return nil
	})
}

// Close закрывает Kafka writer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// GenerateTestOrder создает тестовый заказ для демонстрации с использованием фейковых данных
func GenerateTestOrder(index int) *models.Order {
	var delivery models.Delivery
	var payment models.Payment
	var items []models.Item

	// Генерируем фейковые данные для доставки
	_ = faker.FakeData(&delivery)
	// Устанавливаем OrderUID в пустое значение, так как мы его устанавливаем отдельно
	delivery.OrderUID = ""

	// Генерируем фейковые данные для платежа
	_ = faker.FakeData(&payment)
	// Устанавливаем OrderUID в пустое значение, так как мы его устанавливаем отдельно
	payment.OrderUID = ""

	// Создаем фейковые товары (от 1 до 5 товаров)
	numItems := 1 + index%5 // от 1 до 5 товаров
	for i := 0; i < numItems; i++ {
		var item models.Item
		_ = faker.FakeData(&item)
		item.OrderUID = "" // Устанавливаем OrderUID в пустое значение

		// Убедимся, что цены и ID положительные
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
		items = append(items, item)
	}

	// Создаем заказ с фейковыми данными, обеспечивая валидный OrderUID
	orderUID := fmt.Sprintf("testorderuid%019d", index)
	orderUID = fmt.Sprintf("%-32s", orderUID)[:32]
	orderUID = fmt.Sprintf("testorderuid%020d", index)

	order := &models.Order{
		OrderUID:          orderUID,
		TrackNumber:       faker.UUIDDigit(),
		Entry:             faker.Word(),
		Locale:            faker.Word(),
		InternalSignature: "",
		CustomerID:        faker.UUIDDigit(),
		DeliveryService:   faker.Word(),
		ShardKey:          faker.UUIDDigit(),
		SMID:              1 + (index % 999999),
		DateCreated:       time.Now(),
		OOFShard:          faker.UUIDHyphenated(),
		Delivery:          delivery,
		Payment:           payment,
		Items:             items,
	}

	// Убедимся, что важные поля валидны
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

	// Проверяем валидацию сгенерированного заказа
	if err := order.Validate(); err != nil {
		log.Printf("Сгенерированный заказ не прошел валидацию: %v, будет исправлен", err)
	}

	return order
}
