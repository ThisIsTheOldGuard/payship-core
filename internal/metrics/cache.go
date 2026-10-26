package metrics

import (
	"context"
	"errors"
	"time"

	"github.com/ThisIsTheOldGuard/payship-core/internal/cache"
	"github.com/ThisIsTheOldGuard/payship-core/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// Глобальные метрики для мониторинга производительности и доступности кэша.
var (
	// cacheHitsTotal считает количество успешных обращений к кэшу (данные найдены).
	cacheHitsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of successful cache hits."},
	)
	// cacheMissesTotal считает количество промахов (ключ не найден или кэш отключен).
	// Это ожидаемое поведение, ведущее к запросу в БД, а не сбой.
	cacheMissesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses (key not found)."},
	)
	// cacheErrorsTotal считает только реальные сбои инфраструктуры (таймауты, разрыв сети).
	cacheErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_errors_total",
			Help: "Total number of cache infrastructure errors (e.g., Redis timeout).",
		},
	)
	// cacheOperationDuration измеряет время выполнения операций кэширования.
	// Разбита по типам операций (get, set, delete) через лейблы.
	cacheOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Duration of cache operations in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		}, []string{"operation"},
	)
)

// RegisterCacheMetrics регистрирует все метрики кэша в глобальном реестре Prometheus.
func RegisterCacheMetrics() {
	prometheus.MustRegister(cacheHitsTotal, cacheMissesTotal, cacheErrorsTotal, cacheOperationDuration)
}

// MetricsCache реализует Декоратор для интерфейса Cache.
// Поля:
//   - inner: базовая реализация интерфейса Cache
type MetricsCache struct {
	inner cache.Cache
}

// NewMetricsCache создаёт новый экземпляр декоратора метрик.
//
// Параметры:
//   - inner: любая реализация интерфейса Cache (например, *RedisCache).
//
// Возвращает:
//   - *MetricsCache: обёрнутый экземпляр, готовый к использованию.
func NewMetricsCache(inner cache.Cache) *MetricsCache {
	return &MetricsCache{inner: inner}
}

// Get извлекает значение, измеряет время выполнения и обновляет счётчики hits/misses/errors.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор кэшируемого значения.
//
// Возвращает:
//   - string: значение, если ключ найден.
//   - error: ошибка, если ключ не найден (redis.Nil, domain.ErrCacheDisabled)
//     или произошла реальная ошибка инфраструктуры.
func (m *MetricsCache) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	val, err := m.inner.Get(ctx, key)
	duration := time.Since(start).Seconds()

	cacheOperationDuration.WithLabelValues("get").Observe(duration)

	if err != nil {
		if errors.Is(err, domain.ErrCacheDisabled) || errors.Is(err, redis.Nil) {
			cacheMissesTotal.Inc()
		} else {
			cacheErrorsTotal.Inc()
		}
	} else {
		cacheHitsTotal.Inc()
	}

	return val, err
}

// Set сохраняет значение, измеряет время выполнения и инкрементирует счётчик ошибок при сбое.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор кэшируемого значения.
//   - value: сохраняемое строковое значение.
//   - exp: время жизни значения в кэше (TTL).
//
// Возвращает:
//   - error: ошибка записи, или nil при успехе (или при намеренном no-op в NoopCache).
func (m *MetricsCache) Set(ctx context.Context, key, value string, exp time.Duration) error {
	start := time.Now()
	err := m.inner.Set(ctx, key, value, exp)
	cacheOperationDuration.WithLabelValues("set").Observe(time.Since(start).Seconds())

	if err != nil {
		cacheErrorsTotal.Inc()
	}
	return err
}

// Delete удаляет значение, измеряет время выполнения и инкрементирует счётчик ошибок при сбое.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор удаляемого значения.
//
// Возвращает:
//   - error: ошибка удаления, или nil при успехе.
func (m *MetricsCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	err := m.inner.Delete(ctx, key)
	cacheOperationDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())

	if err != nil {
		cacheErrorsTotal.Inc()
	}
	return err
}

// Ping делегирует запрос проверки работоспособности внутреннему кэшу.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//
// Возвращает:
//   - error: ошибка соединения, или nil, если сервер отвечает.
//     (Метрики для Ping не собираются, так как он используется только при старте).
func (m *MetricsCache) Ping(ctx context.Context) error {
	return m.inner.Ping(ctx)
}
