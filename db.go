package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

var dbPoolCollector *dbPoolStatsCollector

// dbPoolStatsCollector wraps a *sql.DB and exposes connection-pool metrics
// via a Prometheus collector. Metrics are scraped on-demand (not polled in a
// goroutine) to avoid unnecessary load.
type dbPoolStatsCollector struct {
	db *sql.DB

	maxOpenConnections prometheus.Gauge
	openConnections    prometheus.Gauge
	inUse              prometheus.Gauge
	idle               prometheus.Gauge
	waitCount          prometheus.Gauge
	waitDuration       prometheus.Gauge
	maxIdleClosed      prometheus.Gauge
	maxIdleTimeClosed  prometheus.Gauge
	maxLifetimeClosed  prometheus.Gauge
}

// RegisterDBPool extracts the underlying *sql.DB from a *gorm.DB and
// registers a Prometheus collector that exposes database connection-pool
// statistics. Call once after the GORM connection is established.
func RegisterDBPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	registerDBPoolStats(sqlDB)
	return nil
}

func registerDBPoolStats(db *sql.DB) {
	if dbPoolCollector != nil {
		// Already registered; update the reference (e.g. reconnect scenario).
		dbPoolCollector.db = db
		return
	}

	c := &dbPoolStatsCollector{
		db: db,
		maxOpenConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_max_open_connections",
			Help: "Maximum number of open connections to the database.",
		}),
		openConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_open_connections",
			Help: "Number of established connections both in use and idle.",
		}),
		inUse: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_in_use_connections",
			Help: "Number of connections currently in use.",
		}),
		idle: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Number of idle connections.",
		}),
		waitCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_wait_count_total",
			Help: "Total number of connections waited for.",
		}),
		waitDuration: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_wait_duration_seconds_total",
			Help: "Total time blocked waiting for a new connection, in seconds.",
		}),
		maxIdleClosed: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_max_idle_closed_total",
			Help: "Total number of connections closed due to SetMaxIdleConns.",
		}),
		maxIdleTimeClosed: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_max_idle_time_closed_total",
			Help: "Total number of connections closed due to SetConnMaxIdleTime.",
		}),
		maxLifetimeClosed: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_max_lifetime_closed_total",
			Help: "Total number of connections closed due to SetConnMaxLifetime.",
		}),
	}

	dbPoolCollector = c
	prometheus.MustRegister(c)
}

// Describe implements prometheus.Collector.
func (c *dbPoolStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.maxOpenConnections.Describe(ch)
	c.openConnections.Describe(ch)
	c.inUse.Describe(ch)
	c.idle.Describe(ch)
	c.waitCount.Describe(ch)
	c.waitDuration.Describe(ch)
	c.maxIdleClosed.Describe(ch)
	c.maxIdleTimeClosed.Describe(ch)
	c.maxLifetimeClosed.Describe(ch)
}

// Collect implements prometheus.Collector. It reads sql.DB.Stats() on each
// scrape so the values are always current.
func (c *dbPoolStatsCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	c.maxOpenConnections.Set(float64(stats.MaxOpenConnections))
	c.openConnections.Set(float64(stats.OpenConnections))
	c.inUse.Set(float64(stats.InUse))
	c.idle.Set(float64(stats.Idle))

	// Counters in sql.DB.Stats() are cumulative since process start.
	// We use gauges and set the raw value each scrape.
	c.waitCount.Set(float64(stats.WaitCount))
	c.waitDuration.Set(stats.WaitDuration.Seconds())
	c.maxIdleClosed.Set(float64(stats.MaxIdleClosed))
	c.maxIdleTimeClosed.Set(float64(stats.MaxIdleTimeClosed))
	c.maxLifetimeClosed.Set(float64(stats.MaxLifetimeClosed))

	c.maxOpenConnections.Collect(ch)
	c.openConnections.Collect(ch)
	c.inUse.Collect(ch)
	c.idle.Collect(ch)
	c.waitCount.Collect(ch)
	c.waitDuration.Collect(ch)
	c.maxIdleClosed.Collect(ch)
	c.maxIdleTimeClosed.Collect(ch)
	c.maxLifetimeClosed.Collect(ch)
}
