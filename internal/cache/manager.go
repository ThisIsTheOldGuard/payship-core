package cache

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// StateChangeHook определяет сигнатуру функции обратного вызова,
// которая вызывается при переключении активного состояния кэша.
//
// Используется для уведомления внешних компонентов (например, Rate Limiter)
// о смене режима работы без создания прямой зависимости менеджера от них.
//
// Параметры:
//   - isFallback: true/false, если система переключилась на резервный режим (NoopCache).
type StateChangeHook func(isFallback bool)

// CacheManager управляет активной реализацией кэша в runtime.
// Он реализует интерфейс Cache, делегируя вызовы текущему активному кэшу.
type CacheManager struct {
	mu            sync.RWMutex
	active        Cache
	fallback      Cache
	onStateChange StateChangeHook
}

// NewCacheManager создаёт менеджер с первичным и резервным кэшем.
func NewCacheManager(primary, fallback Cache, hook StateChangeHook) *CacheManager {
	return &CacheManager{
		active:        primary,
		fallback:      fallback,
		onStateChange: hook,
	}
}

// Get извлекает значение из текущего активного кэша.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор кэшируемого значения.
//
// Возвращает:
//   - string: значение, если ключ найден в активном кэше.
//   - error: ошибка от активной реализации (Redis или NoopCache).
func (cm *CacheManager) Get(ctx context.Context, key string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.active.Get(ctx, key)
}

// Set сохраняет значение в текущий активный кэш.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор кэшируемого значения.
//   - value: сохраняемое строковое значение.
//   - exp: время жизни значения в кэше (TTL).
//
// Возвращает:
//   - error: ошибка записи от активной реализации, или nil при успехе.
func (cm *CacheManager) Set(ctx context.Context, key, value string, exp time.Duration) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.active.Set(ctx, key, value, exp)
}

// Delete удаляет значение из текущего активного кэша.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//   - key: уникальный идентификатор удаляемого значения.
//
// Возвращает:
//   - error: ошибка удаления от активной реализации, или nil при успехе.
func (cm *CacheManager) Delete(ctx context.Context, key string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.active.Delete(ctx, key)
}

// Ping проверяет доступность текущего активного кэша.
//
// Параметры:
//   - ctx: контекст для отмены операции или установки таймаута.
//
// Возвращает:
//   - error: ошибка соединения, или nil, если кэш доступен.
func (cm *CacheManager) Ping(ctx context.Context) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.active.Ping(ctx)
}

// StartHealthCheck запускает фоновую проверку первичного кэша.
// При серии неудач переключает active на fallback. При восстановлении - обратно.
func (cm *CacheManager) StartHealthCheck(ctx context.Context, primary Cache) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	failCount := 0
	const maxFails = 3

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := primary.Ping(ctx)
			if err != nil {
				failCount++
				slog.Warn("Redis ping failed", "error", err, "fail_count", failCount)

				if failCount >= maxFails {
					cm.mu.Lock()
					cm.active = cm.fallback
					cm.mu.Unlock()
					slog.Error("Redis is DOWN. Switched to Fallback (NoopCache)!")

					if cm.onStateChange != nil {
						cm.onStateChange(true)
					}

				}
			} else {
				if failCount > 0 {
					slog.Info("Redis ping recovered", "previous_fails", failCount)
					failCount = 0

					cm.mu.Lock()
					cm.active = primary
					cm.mu.Unlock()
					slog.Info("Switched back to primary RedisCache")

					if cm.onStateChange != nil {
						cm.onStateChange(false)
					}

				}
			}
		}
	}
}
