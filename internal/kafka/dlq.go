// Package kafka содержит логику для работы с Apache Kafka, включая DLQ
package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

// DLQMessage представляет сообщение в DLQ с дополнительной информацией
type DLQMessage struct {
	OriginalMessage json.RawMessage `json:"original_message"` // Оригинальное сообщение
	Error           string          `json:"error"`            // Ошибка, приведшая к отправке в DLQ
	Timestamp       time.Time       `json:"timestamp"`        // Время отправки в DLQ
	Topic           string          `json:"topic"`            // Изначальный топик
	Key             string          `json:"key"`              // Ключ сообщения
	Attempts        int             `json:"attempts"`         // Количество попыток обработки
}

// DLQProducer для отправки сообщений в DLQ
type DLQProducer struct {
	writer  *kafka.Writer
	topic   string
	metrics *KafkaMetrics
}

// NewDLQProducer создает новый DLQ producer
func NewDLQProducer(brokers []string, dlqTopic string) *DLQProducer {
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  dlqTopic,
		Balancer:               &kafka.LeastBytes{},
		WriteTimeout:           10 * time.Second,
		ReadTimeout:            10 * time.Second,
		RequiredAcks:           kafka.RequireAll,
		MaxAttempts:            3,
		AllowAutoTopicCreation: true,
	}
	return &DLQProducer{
		writer:  writer,
		topic:   dlqTopic,
		metrics: NewKafkaMetrics(),
	}
}

// SendToDLQ отправляет сообщение в DLQ
func (d *DLQProducer) SendToDLQ(originalMsg kafka.Message, err error, attempts int) error {
	dlqMsg := DLQMessage{
		OriginalMessage: originalMsg.Value,
		Error:           err.Error(),
		Timestamp:       time.Now(),
		Topic:           originalMsg.Topic,
		Key:             string(originalMsg.Key),
		Attempts:        attempts,
	}

	msgJSON, jsonErr := json.Marshal(dlqMsg)
	if jsonErr != nil {
		return jsonErr
	}

	dlqKafkaMsg := kafka.Message{
		Key:   originalMsg.Key,
		Value: msgJSON,
		Time:  time.Now(),
	}

	sendErr := d.writer.WriteMessages(context.Background(), dlqKafkaMsg)
	if sendErr != nil {
		d.metrics.FailedSendsTotal.Inc()
		return sendErr
	}

	d.metrics.DLQMessagesSentTotal.Inc()
	return nil
}

// Close закрывает DLQ producer
func (d *DLQProducer) Close() error {
	return d.writer.Close()
}
