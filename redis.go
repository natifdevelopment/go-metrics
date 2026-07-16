package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var redisMetrics struct {
	totalConns    prometheus.Gauge
	idleConns     prometheus.Gauge
	staleConns    prometheus.Gauge
	hitsTotal     prometheus.Gauge
	missesTotal   prometheus.Gauge
	timeoutsTotal prometheus.Gauge
}

// RegisterRedisStats registers Prometheus metrics for a Redis client's
// connection pool. The pool stats are collected on each scrape via a
// goroutine-free approach using a custom collector.
//
// Call once after the Redis client is created.
func RegisterRedisStats(client *redis.Client) {
	redisMetrics.totalConns = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_pool_total_connections",
		Help: "Total number of connections in the Redis pool.",
	})
	redisMetrics.idleConns = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_pool_idle_connections",
		Help: "Number of idle connections in the Redis pool.",
	})
	redisMetrics.staleConns = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_pool_stale_connections",
		Help: "Number of stale connections in the Redis pool.",
	})
	redisMetrics.hitsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_pool_hits_total",
		Help: "Total number of Redis connection pool hits.",
	})
	redisMetrics.missesTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_pool_misses_total",
		Help: "Total number of Redis connection pool misses.",
	})
	redisMetrics.timeoutsTotal = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "redis_pool_timeouts_total",
		Help: "Total number of Redis connection pool timeouts.",
	})

	prometheus.MustRegister(&redisCollector{client: client})
}

type redisCollector struct {
	client *redis.Client
}

func (c *redisCollector) Describe(ch chan<- *prometheus.Desc) {
	redisMetrics.totalConns.Describe(ch)
	redisMetrics.idleConns.Describe(ch)
	redisMetrics.staleConns.Describe(ch)
	redisMetrics.hitsTotal.Describe(ch)
	redisMetrics.missesTotal.Describe(ch)
	redisMetrics.timeoutsTotal.Describe(ch)
}

func (c *redisCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.client.PoolStats()

	redisMetrics.totalConns.Set(float64(stats.TotalConns))
	redisMetrics.idleConns.Set(float64(stats.IdleConns))
	redisMetrics.staleConns.Set(float64(stats.StaleConns))

	// Pool stats hits/misses/timeouts are cumulative, use gauges to set raw values
	redisMetrics.hitsTotal.Set(float64(stats.Hits))
	redisMetrics.missesTotal.Set(float64(stats.Misses))
	redisMetrics.timeoutsTotal.Set(float64(stats.Timeouts))

	redisMetrics.totalConns.Collect(ch)
	redisMetrics.idleConns.Collect(ch)
	redisMetrics.staleConns.Collect(ch)
	redisMetrics.hitsTotal.Collect(ch)
	redisMetrics.missesTotal.Collect(ch)
	redisMetrics.timeoutsTotal.Collect(ch)
}
