package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// 1) Request volume
	RequestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "requests_total",
		Help: "Total number of API requests received.",
	})

	// 2) Concurrency (in flight)
	ActiveRequests = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "active_requests",
		Help: "Current number of in-flight requests.",
	})

	// 3) Request latency (handler duration)
	RequestDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "request_duration_seconds",
		Help:    "End-to-end handler duration for API requests.",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 20, 40, 60, 75},
	})

	// 4) Output volume
	WordsGeneratedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "words_generated_total",
		Help: "Total number of words generated across all streams.",
	})

	// 5) DB write latency
	DBWriteDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "db_write_duration_seconds",
		Help:    "Duration of INSERT into the requests table.",
		Buckets: []float64{0.005, 0.01, 0.02, 0.05, 0.1, 0.25, 0.5},
	})

	// 6) Rate limiting drops
	RateLimitDroppedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rate_limit_dropped_total",
		Help: "Requests rejected by the per-user rate limiter.",
	})
)

func MustRegister(reg prometheus.Registerer) {
	reg.MustRegister(
		RequestsTotal,
		ActiveRequests,
		RequestDurationSeconds,
		WordsGeneratedTotal,
		DBWriteDurationSeconds,
		RateLimitDroppedTotal,
	)
}
