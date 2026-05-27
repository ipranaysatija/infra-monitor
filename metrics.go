package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ---------------------------------------------------------------------------
// Metric definitions
// Each metric has a name, help string, and optional label dimensions.
// These map directly to what you'd track in a real SRE role.
// ---------------------------------------------------------------------------

var (
	// REQUEST METRICS
	// Counter: monotonically increasing. Good for "how many total requests".
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_http_requests_total",
			Help: "Total number of HTTP requests, partitioned by method, path and status code.",
		},
		[]string{"method", "path", "status"},
	)

	// Histogram: tracks distribution of values (e.g. latency buckets).
	// Enables percentile calculations like p50, p95, p99.
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "app_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
		},
		[]string{"method", "path"},
	)

	// ERROR BUDGET / SLI METRICS
	// Gauge: can go up or down. Good for "current state" values.
	errorBudgetRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sli_error_budget_remaining_ratio",
			Help: "Remaining error budget as a ratio (1.0 = full, 0.0 = exhausted). SLO target: 99.9% availability.",
		},
		[]string{"service"},
	)

	sloComplianceRatio = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sli_availability_ratio",
			Help: "Current availability ratio over the rolling window. Below 0.999 = SLO breach.",
		},
		[]string{"service"},
	)

	// SYSTEM / RESOURCE METRICS
	cpuUsagePercent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "Simulated CPU usage percentage per node.",
		},
		[]string{"node"},
	)

	memoryUsagePercent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_percent",
			Help: "Simulated memory usage percentage per node.",
		},
		[]string{"node"},
	)

	// POD / KUBERNETES METRICS
	podRestartCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "k8s_pod_restart_total",
			Help: "Total pod restart count. High values indicate instability.",
		},
		[]string{"namespace", "pod"},
	)

	activeConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_active_connections",
			Help: "Number of currently active connections per service.",
		},
		[]string{"service"},
	)
)

// registerMetrics pre-initialises label combinations so Grafana sees them
// immediately on startup rather than waiting for the first event.
func registerMetrics() {
	services := []string{"api-gateway", "auth-service", "data-service"}
	nodes := []string{"node-1", "node-2", "node-3"}
	paths := []string{"/api/v1/users", "/api/v1/data", "/healthz"}

	for _, svc := range services {
		errorBudgetRemaining.WithLabelValues(svc).Set(1.0)
		sloComplianceRatio.WithLabelValues(svc).Set(1.0)
		activeConnections.WithLabelValues(svc).Set(0)
	}
	for _, node := range nodes {
		cpuUsagePercent.WithLabelValues(node).Set(0)
		memoryUsagePercent.WithLabelValues(node).Set(0)
	}
	for _, path := range paths {
		httpRequestsTotal.WithLabelValues("GET", path, "200").Add(0)
		httpRequestDuration.WithLabelValues("GET", path).Observe(0)
	}
}

// simulateMetrics runs in a background goroutine and generates realistic-looking
// metric values. In a real exporter you'd replace this with actual system calls,
// database queries, or SDK calls to your cloud provider.
func simulateMetrics() {
	services := []string{"api-gateway", "auth-service", "data-service"}
	nodes := []string{"node-1", "node-2", "node-3"}
	paths := []string{"/api/v1/users", "/api/v1/data", "/healthz"}

	// Simulate a gradual error-budget burn over time
	budgets := map[string]float64{
		"api-gateway":  1.0,
		"auth-service": 1.0,
		"data-service": 1.0,
	}

	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for range tick.C {
		t := float64(time.Now().Unix())

		// --- HTTP traffic ---
		for _, path := range paths {
			// Simulate varying request rates
			reqCount := rand.Intn(20) + 5
			for i := 0; i < reqCount; i++ {
				// Inject ~1% errors to simulate a realistic SLI
				status := "200"
				if rand.Float64() < 0.01 {
					status = "500"
				}
				httpRequestsTotal.WithLabelValues("GET", path, status).Inc()

				// Latency follows a sine wave to simulate traffic patterns
				baseLatency := 0.05 + 0.03*math.Sin(t/60)
				jitter := rand.Float64() * 0.02
				httpRequestDuration.WithLabelValues("GET", path).Observe(baseLatency + jitter)
			}
		}

		// --- SLI / Error budget ---
		for _, svc := range services {
			// Burn the budget slowly; reset when exhausted (simulates a new window)
			burn := rand.Float64() * 0.002
			budgets[svc] -= burn
			if budgets[svc] < 0 {
				budgets[svc] = 1.0 // new 30-day window
			}
			errorBudgetRemaining.WithLabelValues(svc).Set(budgets[svc])

			// Availability: mostly above SLO, with occasional dips
			avail := 0.9995 + rand.Float64()*0.0005
			if rand.Float64() < 0.05 { // 5% chance of a bad tick
				avail = 0.997 + rand.Float64()*0.002
			}
			sloComplianceRatio.WithLabelValues(svc).Set(avail)

			// Active connections follow a sine wave (day/night traffic)
			conns := 50 + 40*math.Sin(t/300) + rand.Float64()*10
			activeConnections.WithLabelValues(svc).Set(math.Max(0, conns))
		}

		// --- Node resource usage ---
		for i, node := range nodes {
			// Each node has a different baseline load
			baseCPU := 30.0 + float64(i)*15
			cpu := baseCPU + 10*math.Sin(t/120) + rand.Float64()*5
			cpuUsagePercent.WithLabelValues(node).Set(math.Min(100, cpu))

			baseMem := 45.0 + float64(i)*10
			mem := baseMem + 5*math.Sin(t/200) + rand.Float64()*3
			memoryUsagePercent.WithLabelValues(node).Set(math.Min(100, mem))
		}

		// --- Occasional pod restarts (simulate instability) ---
		if rand.Float64() < 0.1 { // 10% chance each tick
			pod := fmt.Sprintf("app-pod-%d", rand.Intn(5))
			podRestartCount.WithLabelValues("default", pod).Inc()
		}
	}
}
