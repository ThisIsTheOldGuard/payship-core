package repository

import "time"

// Интерфейс для инакпсуляции метода с верхнего уровня для записи метрик работы БД.
type DBMetrics interface {
	ObserveQueryDuration(queryType string, duration time.Duration)
}
