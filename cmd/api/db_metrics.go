package main

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type dbPoolCollector struct {
	pool *pgxpool.Pool
	desc *prometheus.Desc
}

func newDBPoolCollector(pool *pgxpool.Pool) *dbPoolCollector {
	return &dbPoolCollector{
		pool: pool,
		desc: prometheus.NewDesc(
			"db_pool_active_conns",
			"Current number of active connections in the pool.",
			nil, nil,
		),
	}
}

func (c *dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *dbPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.pool != nil {
		stats := c.pool.Stat()
		ch <- prometheus.MustNewConstMetric(
			c.desc,
			prometheus.GaugeValue,
			float64(stats.AcquiredConns()),
		)
	}
}
