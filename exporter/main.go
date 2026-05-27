package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Register all custom metrics
	registerMetrics()

	// Start background goroutine that simulates real workload metrics
	go simulateMetrics()

	// Expose /metrics endpoint for Prometheus to scrape
	http.Handle("/metrics", promhttp.Handler())

	// Health check endpoint - used by Kubernetes liveness probe
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Println("Exporter running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
