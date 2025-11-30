#!/bin/bash
# =============================================================================
# Quick port-forward script for local development
# =============================================================================

echo "Starting port-forwards for EventFlow services..."
echo ""

# Kill any existing port-forwards
pkill -f "kubectl.*port-forward.*eventflow" 2>/dev/null || true

# Start port-forwards in background
kubectl -n eventflow port-forward svc/dashboard 3001:3000 &
kubectl -n eventflow port-forward svc/ui-backend 8007:8080 &
kubectl -n eventflow port-forward svc/grafana 3000:3000 &
kubectl -n eventflow port-forward svc/prometheus 9090:9090 &
kubectl -n eventflow port-forward svc/jaeger 16686:16686 &

echo ""
echo "Port-forwards started:"
echo "  Dashboard:     http://localhost:3001"
echo "  UI Backend:    http://localhost:8007"
echo "  Grafana:       http://localhost:3000 (admin/admin)"
echo "  Prometheus:    http://localhost:9090"
echo "  Jaeger:        http://localhost:16686"
echo ""
echo "Press Ctrl+C to stop all port-forwards"

# Wait for Ctrl+C
wait
