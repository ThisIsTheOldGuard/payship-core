package main

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// dbPoolCollector - кастомный коллектор Prometheus для сбора метрик пула соединений pgxpool.
//
// Реализует интерфейс prometheus.Collector (через методы Describe и Collect),
// позволяя динамически получать статистику пула подключений к PostgreSQL
// в момент скрейпинга метрик.
//
// Поля:
//   - pool:            экземпляр пула соединений *pgxpool.Pool.
//   - activeConnsDesc: дескриптор метрики количества активных соединений.
//   - waitTotalDesc:   дескриптор метрики общего количества ожиданий соединения.
type dbPoolCollector struct {
	pool            *pgxpool.Pool
	activeConnsDesc *prometheus.Desc
	waitTotalDesc   *prometheus.Desc
}

// PrometheusDBMetric - адаптер для записи метрик длительности запросов к БД.
//
// Пустая структура, реализующая методы для инкапсуляции работы с глобальной
// переменной dbQueryDuration. Позволяет внедрять зависимость для записи метрик
// в репозитории или сервисы, избегая прямого обращения к глобальным переменным
// и упрощая тестирование (можно заменить на mock-реализацию).
//
// Пример:
//
//	var metricDB PrometheusDBMetric
//	metricDB.ObserveQueryDuration("select_users", 150*time.Millisecond)
type PrometheusDBMetric struct{}

// dbQueryDuration - Prometheus-гистограмма времени выполнения SQL-запросов.
//
// Измеряет латентность запросов к базе данных в секундах.
// Позволяет отслеживать производительность БД и строить графики перцентилей (p50, p95, p99).
//
// Лейблы:
//   - query_type: тип или имя запроса (например, "select", "insert", "get_user").
//
// Бакеты: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
//
// Тип: prometheus.HistogramVec
// Имя метрики: "db_query_duration_seconds"
var dbQueryDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "PostgreSQL query execution time in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{"query_type"},
)

// RegisterDBMetrics регистрирует глобальные метрики базы данных в реестре Prometheus.
//
// Вызывает prometheus.MustRegister для каждой метрики пакета.
func RegisterDBMetrics() {
	prometheus.MustRegister(dbQueryDuration)
}

// newDBPoolCollector создаёт и инициализирует коллектор метрик пула БД.
//
// Конструктор для dbPoolCollector. Создаёт дескрипторы (prometheus.Desc) для двух метрик:
//   - db_pool_active_conns: текущее количество активных (забраных) соединений.
//   - db_pool_wait_total: общее количество раз, когда запрос ждал свободного соединения.
//
// Параметры:
//   - pool: экземпляр пула соединений *pgxpool.Pool, из которого будут браться статистики.
//
// Возвращает:
//   - *dbPoolCollector: готовый к использованию и регистрации коллектор.
func newDBPoolCollector(pool *pgxpool.Pool) *dbPoolCollector {
	return &dbPoolCollector{
		pool: pool,
		activeConnsDesc: prometheus.NewDesc(
			"db_pool_active_conns",
			"Current number of active connections in the pool.",
			nil, nil,
		),
		waitTotalDesc: prometheus.NewDesc(
			"db_pool_wait_total",
			"Total count of times a connection had to wait for the pool.",
			nil, nil,
		),
	}
}

// ObserveQueryDuration записывает время выполнения SQL-запроса в метрику dbQueryDuration.
//
// Преобразует duration в секунды (float64) и добавляет наблюдение в гистограмму
// с указанным лейблом query_type.
//
// Параметры:
//   - queryType: строковый идентификатор типа или имени запроса (например, "select_users").
//   - duration:  фактическое время выполнения запроса (time.Duration).
func (p PrometheusDBMetric) ObserveQueryDuration(queryType string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

// Describe отправляет дескрипторы метрик пула БД в канал Prometheus.
//
// Реализует метод интерфейса prometheus.Collector.
// Вызывается Prometheus при регистрации коллектора для получения списка
// всех метрик, которые данный коллектор будет отдавать.
//
// Параметры:
//   - ch: канал, в который отправляются дескрипторы (*prometheus.Desc).
//
// Отправляемые дескрипторы:
//   - activeConnsDesc (db_pool_active_conns)
//   - waitTotalDesc   (db_pool_wait_total)
func (c *dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.activeConnsDesc
	ch <- c.waitTotalDesc
}

// Collect собирает актуальные статистики пула БД и отправляет их в Prometheus.
//
// Реализует метод интерфейса prometheus.Collector.
// Вызывается каждый раз, когда Prometheus скрейпит эндпоинт /metrics.
// Получает статистику из pgxpool.Pool.Stat() и формирует константные метрики:
//
//   - db_pool_active_conns (Gauge):
//     Текущее количество активных (AcquiredConns) соединений.
//     Отражает текущую нагрузку на пул.
//
//   - db_pool_wait_total (Counter):
//     Общее количество раз (EmptyAcquireCount), когда клиенту пришлось ждать,
//     потому что в пуле не было свободных соединений.
//     Позволяет отслеживать нехватку соединений (connection starvation).
//
// Параметры:
//   - ch: канал, в который отправляются собранные метрики (prometheus.Metric).
func (c *dbPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool != nil {
		stats := c.pool.Stat()

		// Gauge - текущее значение
		ch <- prometheus.MustNewConstMetric(
			c.activeConnsDesc,
			prometheus.GaugeValue,
			float64(stats.AcquiredConns()),
		)

		// Counter: растущее значение
		ch <- prometheus.MustNewConstMetric(
			c.waitTotalDesc,
			prometheus.CounterValue,
			float64(stats.EmptyAcquireCount()),
		)
	}
}
