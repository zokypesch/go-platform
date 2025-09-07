# Go Platform API with Observability Stack

A complete observability stack using Go + Fiber API with logging, metrics, and tracing.

## Architecture

- **API**: Go + Fiber with demo external API calls
- **Logging**: Loki + Promtail 
- **Metrics**: Prometheus
- **Tracing**: Tempo (OpenTelemetry)
- **Dashboards**: Grafana
- **Alerts**: Alertmanager

## Quick Start

1. **Build and start all services:**
   ```bash
   docker-compose up -d
   ```

2. **Access services:**
   - API: http://localhost:8080
   - Grafana: http://localhost:3000 (admin/admin123)
   - Prometheus: http://localhost:9090
   - Alertmanager: http://localhost:9093

## API Endpoints

- `GET /health` - Health check endpoint
- `GET /api/external` - Demo external API with random responses (200, 401, 500, timeout)
- `GET /metrics` - Prometheus metrics

## Monitoring Features

### Dashboard Metrics
- Request per second
- Response time percentiles (p50, p90, p95, p99)
- Error rate percentage
- External API request status

### Alerts
- Response time > 1s for 5m (warning)
- Error rate > 10% for 5m (warning)
- Response time > 5s for 1m (critical)
- Error rate > 50% for 1m (critical)

## Testing

Generate traffic to see metrics:
```bash
# Generate normal requests
for i in {1..100}; do curl http://localhost:8080/api/external; sleep 0.1; done

# Check health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:8080/metrics
```

## Configuration Files

- `config/prometheus.yml` - Prometheus scraping configuration
- `config/alert-rules.yml` - Alerting rules
- `config/alertmanager.yml` - Alert routing
- `config/grafana/dashboards/api-dashboard.json` - API dashboard
- `config/tempo-config.yml` - Tempo tracing configuration (2 weeks retention)
- `config/loki-config.yml` - Loki logging configuration (2 weeks retention)

## Data Management

### Persistent Volumes
- `data/loki/` - Loki logs storage (mounted volume)
- `data/tempo/` - Tempo traces storage (mounted volume)

### Data Retention
- **Logs**: 2 weeks (336 hours) - automatic cleanup
- **Traces**: 2 weeks (336 hours) - automatic cleanup

### Manual Data Management
```bash
# Check data usage
./scripts/check-data-usage.sh

# Clear all logs and traces (stops services, clears data, restarts)
./scripts/clear-logs.sh

# Manual cleanup (services must be stopped first)
docker-compose stop loki tempo
rm -rf data/loki/* data/tempo/*
docker-compose start loki tempo
```