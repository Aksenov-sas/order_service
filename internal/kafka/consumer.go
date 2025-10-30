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
	reader   *kafka.Reader // Kafka reader для чтения сообщений
	dlq      *DLQProducer  // DLQ producer для отправки неудачных сообщений
	maxRetry int           // Максимальное количество попыток обработки
	metrics  *KafkaMetrics // Метрики для мониторинга
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
	return &Consumer{
		reader:   reader,
		maxRetry: 3,                 // Максимальное количество попыток
		metrics:  NewKafkaMetrics(), // Инициализировать метрики
	}
}

// NewConsumerWithDLQ создает новый Kafka consumer с DLQ
func NewConsumerWithDLQ(brokers []string, topic string, groupID string, dlqProducer *DLQProducer) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,     // Список брокеров Kafka
		GroupID:        groupID,     // ID группы потребителей
		Topic:          topic,       // Топик для чтения
		CommitInterval: time.Second, // Интервал коммита сообщений
	})
	return &Consumer{
		reader:   reader,
		dlq:      dlqProducer,
		maxRetry: 3,                 // Максимальное количество попыток по умолчанию
		metrics:  NewKafkaMetrics(), // Инициализировать метрики
	}
}

// SetMaxRetry устанавливает максимальное количество попыток обработки
func (c *Consumer) SetMaxRetry(maxRetry int) {
	c.maxRetry = maxRetry
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
					c.metrics.FailedReceivesTotal.Inc()
					log.Printf("Ошибка при получении сообщения: %v", err)
					continue
				}
			}

			c.metrics.MessagesReceivedTotal.Inc()

			// Декодируем JSON сообщение в структуру заказа
			var order models.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				c.metrics.ProcessingErrorsTotal.Inc()
				log.Printf("Ошибка дешифровки сообщения: %v", err)
				// Отправляем сообщение в DLQ, если DLQ настроена
				if c.dlq != nil {
					dlqMsg := kafka.Message{
						Topic: c.reader.Config().Topic,
						Key:   msg.Key,
						Value: msg.Value,
					}
					if dlqErr := c.dlq.SendToDLQ(dlqMsg, err, 1); dlqErr != nil {
						log.Printf("Ошибка отправки в DLQ: %v", dlqErr)
					} else {
						c.metrics.DLQMessagesSentTotal.Inc()
						log.Printf("Сообщение отправлено в DLQ из-за ошибки JSON: %s", order.OrderUID)
					}
				}
				// Подтверждаем сообщение, чтобы не зациклиться
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Ошибка commit невалидного сообщения: %v", err)
				}
				continue
			}

			// Валидация полезной нагрузки
			if err := order.Validate(); err != nil {
				c.metrics.ProcessingErrorsTotal.Inc()
				log.Printf("Невалидный заказ %v: %v", order.OrderUID, err)
				// Отправляем сообщение в DLQ
				if c.dlq != nil {
					dlqMsg := kafka.Message{
						Topic: c.reader.Config().Topic,
						Key:   msg.Key,
						Value: msg.Value,
					}
					if dlqErr := c.dlq.SendToDLQ(dlqMsg, err, 1); dlqErr != nil {
						log.Printf("Ошибка отправки в DLQ: %v", dlqErr)
					} else {
						c.metrics.DLQMessagesSentTotal.Inc()
						log.Printf("Сообщение отправлено в DLQ из-за ошибки валидации: %s", order.OrderUID)
					}
				}
				// Подтверждаем сообщение, чтобы не зациклиться
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Ошибка commit невалидного сообщения: %v", err)
				}
				continue
			}

			// Обрабатываем заказ через переданную функцию
			startTime := time.Now()
			if err := processFunc(&order); err != nil {
				c.metrics.ProcessingErrorsTotal.Inc()
				c.metrics.MessageProcessingTime.Observe(time.Since(startTime).Seconds())
				log.Printf("Ошибка обработки заказа %s: %v", order.OrderUID, err)
				// Отправляем сообщение в DLQ
				if c.dlq != nil {
					dlqMsg := kafka.Message{
						Topic: c.reader.Config().Topic,
						Key:   msg.Key,
						Value: msg.Value,
					}
					if dlqErr := c.dlq.SendToDLQ(dlqMsg, err, 1); dlqErr != nil {
						log.Printf("Ошибка отправки в DLQ: %v", dlqErr)
					} else {
						c.metrics.DLQMessagesSentTotal.Inc()
						log.Printf("Сообщение отправлено в DLQ из-за ошибки обработки: %s", order.OrderUID)
					}
				}
				// Подтверждаем сообщение, чтобы не зациклиться
				if err := c.reader.CommitMessages(ctx, msg); err != nil {
					log.Printf("Ошибка commit сообщения: %v", err)
				}
				continue
			}
			c.metrics.MessageProcessingTime.Observe(time.Since(startTime).Seconds())

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
