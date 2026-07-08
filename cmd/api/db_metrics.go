package main

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type dbPoolCollector struct {
	pool            *pgxpool.Pool
	activeConnsDesc *prometheus.Desc
	waitTotalDesc   *prometheus.Desc
}

var dbQueryDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "PostgreSQL query execution time in seconds.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{"query_type"},
)

func RegisterDBMetrics() {
	prometheus.MustRegister(dbQueryDuration)
}

type PrometheusDBMetric struct{}

func (p PrometheusDBMetric) ObserveQueryDuration(queryType string, duration time.Duration) {
	dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

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

func (c *dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.activeConnsDesc
	ch <- c.waitTotalDesc
}

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
