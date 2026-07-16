package metrics

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

var (
	redisMetrics struct {
		totalConns    prometheus.Gauge
		idleConns     prometheus.Gauge
		staleConns    prometheus.Gauge
		hitsTotal     prometheus.Gauge
		missesTotal   prometheus.Gauge
		timeoutsTotal prometheus.Gauge
	}
	redisOnce          sync.Once
	currentRedisClient *redis.Client
)

// RegisterRedisStats registers Prometheus metrics for a Redis client's
// connection pool. The pool stats are collected on each scrape via a
// goroutine-free approach using a custom collector.
//
// It is safe to call multiple times; metrics are registered once and the
// tracked Redis client is updated on subsequent calls.
func RegisterRedisStats(client *redis.Client) {
	if client == nil {
		return
	}

	// Update the client used for collection before ensuring metrics are
	// registered so a late call still points at the right connection.
	currentRedisClient = client

	redisOnce.Do(func() {
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

		if err := prometheus.Register(&redisCollector{}); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				fmt.Printf("failed to register redis stats collector: %v\n", err)
			}
		}
	})
}

type redisCollector struct{}

func (c *redisCollector) Describe(ch chan<- *prometheus.Desc) {
	redisMetrics.totalConns.Describe(ch)
	redisMetrics.idleConns.Describe(ch)
	redisMetrics.staleConns.Describe(ch)
	redisMetrics.hitsTotal.Describe(ch)
	redisMetrics.missesTotal.Describe(ch)
	redisMetrics.timeoutsTotal.Describe(ch)
}

func (c *redisCollector) Collect(ch chan<- prometheus.Metric) {
	if currentRedisClient == nil {
		return
	}

	stats := currentRedisClient.PoolStats()

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
