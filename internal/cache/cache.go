// Package cache предоставляет абстракции и реализации для кэширования данных.
//
// Этот пакет отвечает за:
//   - Определение единого контракта операций с кэшем (Get, Set, Delete, Ping).
//   - Интеграцию с Redis в качестве основного хранилища кэша.
//   - Сбор метрик наблюдаемости (hits, misses, latency) через паттерн Декоратор.
//   - Обеспечение отказоустойчивости (graceful degradation) при недоступности Redis.
//
// Реализации кэша не зависят от бизнес-логики или транспортного слоя (HTTP)
// и внедряются в хендлеры через Dependency Injection (структура Dependencies).
//
// Пример использования:
//
//	redisCache := cache.NewRedisCache("localhost:6379", 10)
//	cached := cache.NewMetricsCache(redisCache) // Сбор метрик
//
//	val, err := cached.Get(ctx, "order:42")
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache определяет контракт для операций с кэшем.
//
// Реализации этого интерфейса (например, RedisCache или NoopCache)
// позволяют абстрагировать бизнес-логику от конкретной технологии хранения.
//
// Методы:
//   - Get: получает значение по ключу.
//   - Set: сохраняет значение с заданным временем жизни (TTL).
//   - Delete: удаляет значение по ключу.
//   - Ping: проверяет доступность кэша.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Ping(ctx context.Context) error
}

// RedisCache реализация интерфейса Cache
// на базе Redis.
//
// Поля:
//   - client: клиент для взаимодействия с сервером Redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache создаёт и инициализирует новый экземпляр RedisCache.
//
// Функция настраивает пул соединений к Redis с указанными параметрами.
//
// Параметры:
//   - addr: адрес Redis-сервера (например, "localhost:6379").
//   - poolSize: максимальное количество соединений в пуле.
//
// Возвращает:
//   - *RedisCache: экземпляр кэша.
func NewRedisCache(addr string, poolSize int) *RedisCache {

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		PoolSize: poolSize,
	})

	return &RedisCache{client: client}
}

// Get извлекает значение из кэша по указанному ключу.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор кэшируемого значения.
//
// Возвращает:
//   - string: значение, если ключ найден.
//   - error: ошибка, если ключ не найден (redis.Nil) или произошла ошибка сети.
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Set сохраняет значение в кэше с заданным временем жизни (TTL).
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор кэшируемого значения.
//   - value: сохраняемое строковое значение.
//   - expiration: время жизни значения в кэше (0 означает отсутствие TTL).
//
// Возвращает:
//   - error: ошибка при записи в Redis, или nil при успехе.
func (c *RedisCache) Set(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Delete удаляет значение из кэша по указанному ключу.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор удаляемого значения.
//
// Возвращает:
//   - error: ошибка при удалении, или nil при успехе (даже если ключ не существовал).
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Ping проверяет доступность и работоспособность соединения с Redis.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//
// Возвращает:
//   - error: ошибка соединения, или nil, если сервер отвечает.
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
