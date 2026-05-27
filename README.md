# Infrastructure Monitor — Kubernetes Observability Stack

A production-style SRE observability stack built from scratch.
Tracks SLIs, error budgets, latency, and resource usage across services
running on Kubernetes, with automated alerting and Grafana dashboards.

## Stack

| Component | Role |
|---|---|
| Go exporter | Produces custom Prometheus metrics over HTTP |
| Prometheus | Scrapes metrics, stores time-series, evaluates alert rules |
| Grafana | Visualises metrics; dashboard provisioned as code |
| Kubernetes | Runs all of the above; manages rolling updates and health checks |

## Project layout

```
infra-monitor/
├── exporter/
│   ├── main.go          # HTTP server + /metrics + /healthz endpoints
│   ├── metrics.go       # Metric definitions (counters, gauges, histograms)
│   ├── go.mod           # Go module dependencies
│   └── Dockerfile       # Multi-stage build → minimal scratch image
├── k8s/
│   ├── configmap.yaml          # Exporter environment config
│   ├── exporter-deployment.yaml # Deployment + Service + probes
│   ├── prometheus.yaml         # Prometheus + RBAC + alert rules
│   ├── grafana.yaml            # Grafana + datasource provisioning
│   └── grafana-dashboard.yaml  # SRE dashboard as code
└── deploy.sh            # Rolling restart / health-check / rollback automation
```

## Quick start (requires Docker Desktop with Kubernetes enabled)

```bash
# 1. Build and push the exporter image
cd exporter
docker build -t ipranaysatija/infra-monitor-exporter:latest .
docker push ipranaysatija/infra-monitor-exporter:latest

# 2. Deploy everything
cd ..
chmod +x deploy.sh
./deploy.sh apply

# 3. Open Grafana
kubectl port-forward svc/grafana 3000:3000
# Visit http://localhost:3000  (admin / admin)

# 4. Open Prometheus
kubectl port-forward svc/prometheus 9090:9090
# Visit http://localhost:9090
```

## Metrics exposed

| Metric | Type | What it tracks |
|---|---|---|
| `app_http_requests_total` | Counter | Total requests by method, path, status |
| `app_http_request_duration_seconds` | Histogram | Request latency (enables p50/p95/p99) |
| `sli_availability_ratio` | Gauge | Rolling availability — SLO target 99.9% |
| `sli_error_budget_remaining_ratio` | Gauge | Fraction of monthly error budget left |
| `system_cpu_usage_percent` | Gauge | CPU % per node |
| `system_memory_usage_percent` | Gauge | Memory % per node |
| `app_active_connections` | Gauge | Live connections per service |
| `k8s_pod_restart_total` | Counter | Pod restart count (crash-loop detection) |

## Alert rules

| Alert | Condition | Severity |
|---|---|---|
| SLOBreachAvailability | availability < 99.9% for 2m | critical |
| ErrorBudgetBurnHigh | >50% budget consumed | warning |
| ErrorBudgetNearExhaustion | <10% budget remaining | critical |
| HighCPUUsage | CPU > 80% for 5m | warning |
| PodCrashLooping | >5 restarts in 10m | critical |

## SLO / Error budget explained

SLO (Service Level Objective): 99.9% availability = max 43.8 min downtime/month.
Error budget = 100% − SLO = 0.1% of all requests may fail.

The `sli_error_budget_remaining_ratio` metric lets you answer:
"How much of our failure allowance have we used this month?"
When it hits 0, all risky deployments must stop until the window resets.
