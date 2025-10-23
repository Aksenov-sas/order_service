// Package kafka содержит логику для работы с Apache Kafka
package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"test_service/internal/models"

	"github.com/segmentio/kafka-go"
)

// Consumer для обработки сообщений
type Consumer struct {
	reader *kafka.Reader // Kafka reader для чтения сообщений
}

// NewConsumer создает новый Kafka consumer
func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	// Создаем конфигурацию для Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,     // Список брокеров Kafka
		GroupID:        groupID,     // ID группы потребителей
		Topic:          topic,       // Топик для чтения
		CommitInterval: time.Second, // Интервал коммита сообщений
	})
	return &Consumer{reader: reader}
}

// Consume запускает бесконечный цикл обработки сообщений из Kafka
func (c *Consumer) Consume(ctx context.Context, processFunc func(*models.Order) error) error {
	for {
		select {
		case <-ctx.Done():
			// Контекст выполнен, закрываем reader
			return c.reader.Close()
		default:
			// Получаем сообщение из Kafka
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				// Если контекст отменен, выходим
				select {
				case <-ctx.Done():
					return nil
				default:
					log.Printf("Ошибка при получении сообщения: %v", err)
					continue
				}
			}

			// Декодируем JSON сообщение в структуру заказа
			var order models.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				log.Printf("Ошибка дешифровки сообщения: %v", err)
				// Сообщение невалидно как JSON — подтверждаем, чтобы не зациклиться
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Ошибка commit невалидного сообщения: %v", err)
				}
				continue
			}

			// Валидация полезной нагрузки
			if err := order.Validate(); err != nil {
				log.Printf("Невалидный заказ %v: %v", order.OrderUID, err)
				// Пропускаем сообщение и коммитим, чтобы не зациклиться
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Ошибка commit невалидного сообщения: %v", err)
				}
				continue
			}

			// Обрабатываем заказ через переданную функцию
			if err := processFunc(&order); err != nil {
				log.Printf("Ошибка обработки заказа %s: %v", order.OrderUID, err)
				continue
			}

			// Подтверждаем обработку сообщения
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Ошибка commit сообщения: %v", err)
			}
		}
	}
}

// Close закрывает Kafka reader
func (c *Consumer) Close() error {
	return c.reader.Close()
}
