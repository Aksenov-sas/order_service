package database

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DBMetrics содержит все метрики, связанные с базой данных
type DBMetrics struct {
	SuccessfulSavesTotal  prometheus.Counter
	FailedSavesTotal      prometheus.Counter
	SuccessfulGetsTotal   prometheus.Counter
	FailedGetsTotal       prometheus.Counter
	SuccessfulGetAllTotal prometheus.Counter
	FailedGetAllTotal     prometheus.Counter

	SaveDuration   prometheus.Histogram
	GetDuration    prometheus.Histogram
	GetAllDuration prometheus.Histogram
	InitDuration   prometheus.Histogram

	ConnectionErrorsTotal  prometheus.Counter
	TransactionErrorsTotal prometheus.Counter
	QueryErrorsTotal       prometheus.Counter

	ConnectionOpen            prometheus.Gauge
	ConnectionAcquireCount    prometheus.Counter
	ConnectionAcquireDuration prometheus.Histogram
	ConnectionMaxOpen         prometheus.Gauge

	QueryDuration *prometheus.HistogramVec
	QueryErrors   *prometheus.CounterVec

	ConnectionEstablishDuration prometheus.Histogram
}

// Global metrics для предотвращения дублирования метрик
var globalDBMetrics *DBMetrics

// NewDBMetrics создает и регистрирует новые метрики БД
func NewDBMetrics() *DBMetrics {
	// Возвращаем глобальный экземпляр, чтобы избежать дублирования метрик
	if globalDBMetrics != nil {
		return globalDBMetrics
	}

	globalDBMetrics = &DBMetrics{
		SuccessfulSavesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_successful_saves_total",
			Help: "Общее количество успешных операций сохранения в БД",
		}),
		FailedSavesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_failed_saves_total",
			Help: "Общее количество неудачных операций сохранения в БД",
		}),
		SuccessfulGetsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_successful_gets_total",
			Help: "Общее количество успешных операций получения из БД",
		}),
		FailedGetsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_failed_gets_total",
			Help: "Общее количество неудачных операций получения из БД",
		}),
		SuccessfulGetAllTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_successful_get_all_total",
			Help: "Общее количество успешных операций получения всех записей из БД",
		}),
		FailedGetAllTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_failed_get_all_total",
			Help: "Общее количество неудачных операций получения всех записей из БД",
		}),
		SaveDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_save_duration_seconds",
			Help:    "Время выполнения операции сохранения в БД в секундах",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}),
		GetDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_get_duration_seconds",
			Help:    "Время выполнения операции получения из БД в секундах",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}),
		GetAllDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_get_all_duration_seconds",
			Help:    "Время выполнения операции получения всех записей из БД в секундах",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}),
		InitDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_init_duration_seconds",
			Help:    "Время выполнения инициализации БД в секундах",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		}),
		ConnectionErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_connection_errors_total",
			Help: "Общее количество ошибок подключения к БД",
		}),
		TransactionErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_transaction_errors_total",
			Help: "Общее количество ошибок транзакций в БД",
		}),
		QueryErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Общее количество ошибок запросов к БД",
		}),
		ConnectionOpen: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Количество открытых соединений с БД",
		}),
		ConnectionAcquireCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_connection_acquire_total",
			Help: "Количество попыток получения соединения из пула",
		}),
		ConnectionAcquireDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_connection_acquire_duration_seconds",
			Help:    "Время ожидания получения соединения из пула в секундах",
			Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		}),
		ConnectionMaxOpen: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_connections_max_open",
			Help: "Максимальное количество открытых соединений в пуле",
		}),
		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Время выполнения SQL-запросов в секундах, разбитое по типу операции",
				Buckets: []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
			},
			[]string{"operation"},
		),
		QueryErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_query_errors_by_operation_total",
				Help: "Количество ошибок SQL-запросов, разбитое по типу операции",
			},
			[]string{"operation"},
		),
		ConnectionEstablishDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_connection_establish_duration_seconds",
			Help:    "Время установления подключения к БД в секундах",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		}),
	}

	return globalDBMetrics
}

// ResetDBMetricsForTest сбрасывает глобальные метрики БД (для использования в тестах)
func ResetDBMetricsForTest() {
	globalDBMetrics = nil
}
