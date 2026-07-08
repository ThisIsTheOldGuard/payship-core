package api

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
