package repository

import "time"

type DBMetrics interface {
	ObserveQueryDuration(queryType string, duration time.Duration)
}
