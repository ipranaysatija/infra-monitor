#!/usr/bin/env bash
# deploy.sh — SRE automation script
# Applies all manifests, performs rolling restarts, and verifies health
# Usage: ./deploy.sh [apply|restart|status|rollback]

set -euo pipefail

NAMESPACE="default"
DEPLOYMENT="infra-monitor-exporter"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
fail() { echo -e "${RED}[FAIL]${NC}  $*"; exit 1; }

# ------------------------------------------------------------------
apply() {
  log "Applying all Kubernetes manifests..."
  kubectl apply -f k8s/configmap.yaml
  kubectl apply -f k8s/prometheus.yaml
  kubectl apply -f k8s/grafana.yaml
  kubectl apply -f k8s/grafana-dashboard.yaml
  kubectl apply -f k8s/exporter-deployment.yaml
  log "All manifests applied."
  wait_for_rollout
}

# ------------------------------------------------------------------
# Rolling restart: terminates pods one at a time so there is always
# at least one healthy pod serving traffic (zero-downtime update)
rolling_restart() {
  log "Triggering rolling restart of ${DEPLOYMENT}..."
  kubectl rollout restart deployment/"${DEPLOYMENT}" -n "${NAMESPACE}"
  wait_for_rollout
}

# ------------------------------------------------------------------
# Wait for rollout to complete — exits non-zero on timeout
wait_for_rollout() {
  log "Waiting for rollout to complete (timeout: 120s)..."
  if kubectl rollout status deployment/"${DEPLOYMENT}" \
       -n "${NAMESPACE}" --timeout=120s; then
    log "Rollout complete."
  else
    fail "Rollout timed out! Run: kubectl describe deployment/${DEPLOYMENT}"
  fi
}

# ------------------------------------------------------------------
# Health check: queries each pod's /healthz endpoint directly
health_check() {
  log "Running health checks..."
  PODS=$(kubectl get pods -n "${NAMESPACE}" \
    -l app="${DEPLOYMENT}" \
    -o jsonpath='{.items[*].metadata.name}')

  ALL_OK=true
  for pod in $PODS; do
    STATUS=$(kubectl exec "$pod" -n "${NAMESPACE}" -- \
      wget -qO- http://localhost:8080/healthz 2>/dev/null || echo "FAIL")
    if [ "$STATUS" = "ok" ]; then
      log "Pod ${pod}: healthy"
    else
      warn "Pod ${pod}: UNHEALTHY (got: ${STATUS})"
      ALL_OK=false
    fi
  done

  if [ "$ALL_OK" = false ]; then
    fail "One or more pods are unhealthy."
  fi
  log "All pods healthy."
}

# ------------------------------------------------------------------
# Rollback: reverts to the previous ReplicaSet (last known good)
rollback() {
  warn "Rolling back ${DEPLOYMENT} to previous revision..."
  kubectl rollout undo deployment/"${DEPLOYMENT}" -n "${NAMESPACE}"
  wait_for_rollout
  log "Rollback complete."
}

# ------------------------------------------------------------------
# Status summary
status() {
  log "=== Deployment status ==="
  kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}"
  echo ""
  log "=== Pod status ==="
  kubectl get pods -n "${NAMESPACE}" -l app="${DEPLOYMENT}" -o wide
  echo ""
  log "=== Recent events ==="
  kubectl get events -n "${NAMESPACE}" \
    --field-selector involvedObject.name="${DEPLOYMENT}" \
    --sort-by='.lastTimestamp' | tail -10
}

# ------------------------------------------------------------------
CMD="${1:-apply}"
case "$CMD" in
  apply)   apply ;;
  restart) rolling_restart ;;
  health)  health_check ;;
  rollback) rollback ;;
  status)  status ;;
  *)       echo "Usage: $0 [apply|restart|health|rollback|status]" ; exit 1 ;;
esac
