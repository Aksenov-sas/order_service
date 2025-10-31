package kafka

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// KafkaMetrics содержит все метрики, связанные с Kafka
type KafkaMetrics struct {
	// Messages
	MessagesSentTotal     prometheus.Counter
	MessagesReceivedTotal prometheus.Counter
	MessageProcessingTime prometheus.Histogram
	FailedSendsTotal      prometheus.Counter
	FailedReceivesTotal   prometheus.Counter

	// Retries
	RetryAttemptsTotal prometheus.Counter

	// DLQ
	DLQMessagesSentTotal prometheus.Counter

	// Errors
	ProcessingErrorsTotal prometheus.Counter
}

// Global registry для предотвращения дублирования метрик
var globalKafkaMetrics *KafkaMetrics

// NewKafkaMetrics создает и регистрирует новые метрики Kafka
func NewKafkaMetrics() *KafkaMetrics {
	// Возвращаем глобальный экземпляр, чтобы избежать дублирования метрик
	if globalKafkaMetrics != nil {
		return globalKafkaMetrics
	}

	globalKafkaMetrics = &KafkaMetrics{
		MessagesSentTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_sent_total",
			Help: "Общее количество отправленных сообщений в Kafka",
		}),
		MessagesReceivedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kafka_messages_received_total",
			Help: "Общее количество полученных сообщений из Kafka",
		}),
		MessageProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "kafka_message_processing_duration_seconds",
			Help:    "Время обработки сообщения Kafka в секундах",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}),
		FailedSendsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kafka_failed_sends_total",
			Help: "Общее количество неудачных попыток отправки сообщений в Kafka",
		}),
		FailedReceivesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kafka_failed_receives_total",
			Help: "Общее количество неудачных попыток получения сообщений из Kafka",
		}),
		RetryAttemptsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kafka_retry_attempts_total",
			Help: "Общее количество попыток повторной отправки/получения сообщений",
		}),
		DLQMessagesSentTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kafka_dlq_messages_sent_total",
			Help: "Общее количество сообщений, отправленных в DLQ",
		}),
		ProcessingErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kafka_processing_errors_total",
			Help: "Общее количество ошибок обработки сообщений",
		}),
	}

	return globalKafkaMetrics
}

// ResetMetricsForTest сбрасывает глобальные метрики (для использования в тестах)
func ResetMetricsForTest() {
	globalKafkaMetrics = nil
}
