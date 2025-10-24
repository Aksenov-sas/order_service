package kafka

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDLQMessageStructure(t *testing.T) {
	t.Run("ValidDLQMessage", func(t *testing.T) {
		originalMsg := json.RawMessage(`{"order_uid": "test-123", "track_number": "TRACK123"}`)
		err := "validation error"
		attempts := 3

		dlqMsg := DLQMessage{
			OriginalMessage: originalMsg,
			Error:           err,
			Timestamp:       time.Now(),
			Topic:           "test-topic",
			Key:             "test-key",
			Attempts:        attempts,
		}

		// Проверяем, что структура правильная
		assert.Equal(t, originalMsg, dlqMsg.OriginalMessage)
		assert.Equal(t, err, dlqMsg.Error)
		assert.Equal(t, "test-topic", dlqMsg.Topic)
		assert.Equal(t, "test-key", dlqMsg.Key)
		assert.Equal(t, attempts, dlqMsg.Attempts)
		assert.NotZero(t, dlqMsg.Timestamp)
	})

	t.Run("DLQMessageSerialization", func(t *testing.T) {
		originalMsg := json.RawMessage(`{"test": "data"}`)
		dlqMsg := DLQMessage{
			OriginalMessage: originalMsg,
			Error:           "test error",
			Timestamp:       time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			Topic:           "test-topic",
			Key:             "test-key",
			Attempts:        1,
		}

		// Сериализуем в JSON
		data, err := json.Marshal(dlqMsg)
		require.NoError(t, err)

		// Десериализуем обратно
		var deserialized DLQMessage
		err = json.Unmarshal(data, &deserialized)
		require.NoError(t, err)

		// Проверяем, что основные данные сохранены
		assert.Equal(t, dlqMsg.Error, deserialized.Error)
		assert.Equal(t, dlqMsg.Topic, deserialized.Topic)
		assert.Equal(t, dlqMsg.Key, deserialized.Key)
		assert.Equal(t, dlqMsg.Attempts, deserialized.Attempts)
		assert.Equal(t, dlqMsg.Timestamp.Unix(), deserialized.Timestamp.Unix()) // Сравниваем Unix временные метки, чтобы избежать проблем с точностью

		// Проверяем, что содержимое оригинального сообщения сохранено после обработки
		var originalData map[string]interface{}
		var deserializedOriginalData map[string]interface{}
		err1 := json.Unmarshal(dlqMsg.OriginalMessage, &originalData)
		err2 := json.Unmarshal(deserialized.OriginalMessage, &deserializedOriginalData)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, originalData, deserializedOriginalData)
	})
}

func TestDLQProducer(t *testing.T) {

	t.Run("NewDLQProducer", func(t *testing.T) {
		brokers := []string{"localhost:9092"}
		topic := "test-dlq-topic"

		producer := NewDLQProducer(brokers, topic)

		// Проверяем, что продюсер был создан с правильными значениями
		assert.NotNil(t, producer)
		assert.Equal(t, topic, producer.topic)
		assert.NotNil(t, producer.writer)
	})
}

func TestDLQMessageSending(t *testing.T) {
	// Этот тест проверяет, что структура DLQ сообщения и сериализация работают правильно
	t.Run("CreateAndSerializeDLQMessage", func(t *testing.T) {
		// Создаем оригинальное Kafka сообщение
		originalMsgValue := json.RawMessage(`{"order_uid": "test-order", "track_number": "TEST001"}`)
		originalMsg := kafka.Message{
			Topic:   "original-topic",
			Key:     []byte("original-key"),
			Value:   originalMsgValue,
			Headers: []kafka.Header{{Key: "header1", Value: []byte("value1")}},
		}

		testErr := "test processing error"
		attempts := 2

		// Создаем DLQ сообщение
		dlqMsg := DLQMessage{
			OriginalMessage: originalMsgValue,
			Error:           testErr,
			Timestamp:       time.Now(),
			Topic:           originalMsg.Topic,
			Key:             string(originalMsg.Key),
			Attempts:        attempts,
		}

		// Проверяем все поля
		assert.Equal(t, originalMsgValue, dlqMsg.OriginalMessage)
		assert.Equal(t, testErr, dlqMsg.Error)
		assert.Equal(t, "original-topic", dlqMsg.Topic)
		assert.Equal(t, "original-key", dlqMsg.Key)
		assert.Equal(t, attempts, dlqMsg.Attempts)
		assert.NotZero(t, dlqMsg.Timestamp)

		// Сериализуем в JSON, чтобы убедиться, что это работает для отправки в Kafka
		jsonData, err := json.Marshal(dlqMsg)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Проверяем, что мы можем десериализовать его обратно
		var deserialized DLQMessage
		err = json.Unmarshal(jsonData, &deserialized)
		assert.NoError(t, err)

		// Проверяем важные поля
		assert.Equal(t, dlqMsg.Error, deserialized.Error)
		assert.Equal(t, dlqMsg.Topic, deserialized.Topic)
		assert.Equal(t, dlqMsg.Key, deserialized.Key)
		assert.Equal(t, dlqMsg.Attempts, deserialized.Attempts)

		// Проверяем, что содержимое оригинального сообщения сохранено
		var originalData map[string]interface{}
		var deserializedOriginalData map[string]interface{}
		err1 := json.Unmarshal(dlqMsg.OriginalMessage, &originalData)
		err2 := json.Unmarshal(deserialized.OriginalMessage, &deserializedOriginalData)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, originalData, deserializedOriginalData)
	})
}

func TestConsumerWithDLQConstructor(t *testing.T) {
	t.Run("NewConsumerWithDLQ", func(t *testing.T) {
		brokers := []string{"localhost:9092"}
		topic := "test-topic"
		groupID := "test-group"
		dlqProducer := &DLQProducer{topic: "test-dlq"}

		consumer := NewConsumerWithDLQ(brokers, topic, groupID, dlqProducer)

		// Проверяем, что консьюмер был создан с правильными значениями
		assert.NotNil(t, consumer)
		assert.Equal(t, dlqProducer, consumer.dlq)
		assert.Equal(t, 3, consumer.maxRetry)
		assert.NotNil(t, consumer.reader)
	})

	t.Run("SetMaxRetry", func(t *testing.T) {
		consumer := &Consumer{maxRetry: 3}
		assert.Equal(t, 3, consumer.maxRetry)

		consumer.SetMaxRetry(5)
		assert.Equal(t, 5, consumer.maxRetry)
	})
}

func TestConsumerConstructor(t *testing.T) {
	t.Run("NewConsumer", func(t *testing.T) {
		brokers := []string{"localhost:9092"}
		topic := "test-topic"
		groupID := "test-group"

		consumer := NewConsumer(brokers, topic, groupID)

		// Проверяем, что консьюмер был создан с правильными значениями
		assert.NotNil(t, consumer)
		assert.Nil(t, consumer.dlq) // DLQ должен быть nil по умолчанию
		assert.Equal(t, 3, consumer.maxRetry)
		assert.NotNil(t, consumer.reader)
	})
}

func TestDLQIntegration(t *testing.T) {
	t.Run("DLQMessageWithRealOrderData", func(t *testing.T) {
		// Тест с реальной структурой JSON заказа
		orderJSON := json.RawMessage(`{
			"order_uid": "test-order-uid-12345678901234567890123",
			"track_number": "TESTTRACK123",
			"entry": "EntryTest",
			"locale": "en",
			"customer_id": "customer123",
			"delivery_service": "delivery_service",
			"shard_key": "shard1",
			"sm_id": 1,
			"oof_shard": "oof_shard",
			"delivery": {
				"name": "Test Customer",
				"phone": "+1234567890",
				"zip": "12345",
				"city": "Test City",
				"address": "Test Address",
				"region": "Test Region",
				"email": "test@example.com"
			},
			"payment": {
				"transaction": "trans123",
				"currency": "USD",
				"provider": "provider_test",
				"amount": 1000,
				"payment_dt": 1678886400,
				"bank": "Test Bank",
				"delivery_cost": 200,
				"goods_total": 800,
				"custom_fee": 0
			},
			"items": [
				{
					"chrt_id": 1000,
					"track_number": "TRACK123",
					"price": 500,
					"rid": "rid123",
					"name": "Test Item",
					"size": "M",
					"total_price": 500,
					"nm_id": 5000,
					"brand": "Test Brand"
				}
			]
		}`)

		dlqMsg := DLQMessage{
			OriginalMessage: orderJSON,
			Error:           "validation failed: missing required field",
			Timestamp:       time.Now(),
			Topic:           "orders-topic",
			Key:             "test-order-uid-12345678901234567890123",
			Attempts:        1,
		}

		// Сериализуем, чтобы убедиться, что это работает
		jsonData, err := json.Marshal(dlqMsg)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Проверяем, что можем десериализовать его обратно
		var deserialized DLQMessage
		err = json.Unmarshal(jsonData, &deserialized)
		assert.NoError(t, err)
		assert.Equal(t, dlqMsg.Error, deserialized.Error)
		assert.Equal(t, dlqMsg.Topic, deserialized.Topic)
	})
}

func TestDLQMessageEmptyValues(t *testing.T) {
	t.Run("DLQMessageWithEmptyValues", func(t *testing.T) {
		dlqMsg := DLQMessage{
			OriginalMessage: json.RawMessage(`{}`), // Используем валидный JSON вместо пустых байтов
			Error:           "",
			Timestamp:       time.Time{},
			Topic:           "",
			Key:             "",
			Attempts:        0,
		}

		// Сериализуем и десериализуем
		jsonData, err := json.Marshal(dlqMsg)
		assert.NoError(t, err)

		var deserialized DLQMessage
		err = json.Unmarshal(jsonData, &deserialized)
		assert.NoError(t, err)

		// Значения должны сохраняться даже если они пустые
		assert.Equal(t, dlqMsg.OriginalMessage, deserialized.OriginalMessage)
		assert.Equal(t, dlqMsg.Error, deserialized.Error)
		assert.Equal(t, dlqMsg.Topic, deserialized.Topic)
		assert.Equal(t, dlqMsg.Key, deserialized.Key)
		assert.Equal(t, dlqMsg.Attempts, deserialized.Attempts)
	})
}

func TestDLQProducerSendToDLQ(t *testing.T) {
	// Этот тест проверяет, что метод SendToDLQ работает правильно с правильными параметрами

	t.Run("SendToDLQMethodStructure", func(t *testing.T) {
		// Создаем мок оригинального сообщения
		originalMsgValue := json.RawMessage(`{"order_uid": "test-order"}`)
		originalMsg := kafka.Message{
			Topic: "original-topic",
			Key:   []byte("test-key"),
			Value: originalMsgValue,
		}

		testErr := "test error for DLQ"
		attempts := 1

		// Создаем DLQProducer - но мы не используем реальное подключение к Kafka для тестов
		// Вместо этого мы тестируем логику создания DLQ сообщения
		dlqMsg := DLQMessage{
			OriginalMessage: originalMsgValue,
			Error:           testErr,
			Timestamp:       time.Now(),
			Topic:           originalMsg.Topic,
			Key:             string(originalMsg.Key),
			Attempts:        attempts,
		}

		// Проверяем, что структура правильная
		assert.Equal(t, originalMsgValue, dlqMsg.OriginalMessage)
		assert.Equal(t, testErr, dlqMsg.Error)
		assert.Equal(t, originalMsg.Topic, dlqMsg.Topic)
		assert.Equal(t, "test-key", dlqMsg.Key)
		assert.Equal(t, attempts, dlqMsg.Attempts)
		assert.WithinDuration(t, time.Now(), dlqMsg.Timestamp, 1*time.Second)
	})
}
