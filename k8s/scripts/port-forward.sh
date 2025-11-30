#!/bin/bash
# =============================================================================
# Quick port-forward script for local development
# =============================================================================

echo "Starting port-forwards for EventFlow services..."
echo ""

# Kill any existing port-forwards
pkill -f "kubectl.*port-forward.*eventflow" 2>/dev/null || true

# Start port-forwards in background (accessible from all IPs)
kubectl -n eventflow port-forward --address 0.0.0.0 svc/dashboard 3001:3000 &
kubectl -n eventflow port-forward --address 0.0.0.0 svc/ui-backend 8007:8080 &
kubectl -n eventflow port-forward --address 0.0.0.0 svc/grafana 3000:3000 &
kubectl -n eventflow port-forward --address 0.0.0.0 svc/prometheus 9090:9090 &
kubectl -n eventflow port-forward --address 0.0.0.0 svc/jaeger 16686:16686 &

echo ""
echo "Port-forwards started (accessible from all IPs):"
echo "  Dashboard:     http://<YOUR_IP>:3001"
echo "  UI Backend:    http://<YOUR_IP>:8007"
echo "  Grafana:       http://<YOUR_IP>:3000 (admin/admin)"
echo "  Prometheus:    http://<YOUR_IP>:9090"
echo "  Jaeger:        http://<YOUR_IP>:16686"
echo ""
echo "Press Ctrl+C to stop all port-forwards"

# Wait for Ctrl+C
wait
