package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Debug_DB_Slow создаёт HTTP-обработчик для отладки и тестирования поведения пула соединений БД.
//
// Возвращаемая функция имитирует медленный запрос к БД.
// В случае ошибки при захвате соединения возвращает HTTP 500.
//
// Этот эндпоинт полезен для стресс-тестов и проверки наблюдаемости:
//   - Позволяет визуально отследить рост метрики db_pool_active_conns.
//   - Помогает проверить, как система реагирует на исчерпание пула (connection starvation)
//     и рост метрики db_pool_wait_total при одновременных запросах.
//   - Даёт возможность протестировать таймауты на уровне приложения или балансировщика.
//
// Параметры:
//   - pool: экземпляр пула соединений *pgxpool.Pool, из которого будет забираться соединение.
func Debug_DB_Slow(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := pool.Acquire(r.Context())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer conn.Release()
		time.Sleep(20 * time.Second)
		w.Write([]byte("done"))
	}
}
