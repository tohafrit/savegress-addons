# DataWatch

> Real-time insights, zero ETL

DataWatch transforms Savegress CDC events into actionable analytics without traditional ETL pipelines. Automatic metrics, anomaly detection, and data quality monitoring out of the box.

## Architecture

```
datawatch/
├── cmd/
│   └── datawatch/
│       └── main.go              # Entry point
├── internal/
│   ├── api/                     # HTTP API handlers
│   │   ├── router.go
│   │   └── handlers.go
│   ├── metrics/                 # Metrics engine
│   │   ├── engine.go            # Core metrics processing
│   │   ├── auto_discover.go     # Auto-metric generation
│   │   ├── aggregator.go        # Time-window aggregations
│   │   └── types.go             # Data types
│   ├── anomaly/                 # Anomaly detection
│   │   ├── detector.go          # Detection coordinator
│   │   ├── statistical.go       # Z-score, MAD algorithms
│   │   ├── seasonal.go          # STL decomposition
│   │   └── types.go             # Anomaly types
│   ├── storage/                 # Metric storage
│   │   ├── storage.go           # Storage interface
│   │   └── embedded.go          # SQLite-based storage
│   └── config/                  # Configuration
│       └── config.go
├── docker/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── datawatch.yaml           # Default config
└── pkg/
    └── sdk/                     # Client SDK
```

## Features

### Auto-Metrics Engine

Automatically generates metrics from CDC events:

```yaml
# Auto-generated metrics for "orders" table
auto_metrics:
  orders:
    - orders_events_total        # Total CDC events
    - orders_inserts_total       # INSERT events
    - orders_updates_total       # UPDATE events
    - orders_deletes_total       # DELETE events
    - orders_amount_sum          # SUM of amount field
    - orders_amount_avg          # AVG of amount field
    - orders_by_status           # Count by status
```

### Anomaly Detection

ML-based detection without configuration:

- **Statistical** - Z-score and MAD-based detection
- **Seasonal** - STL decomposition for time patterns
- Automatic baseline learning
- Configurable sensitivity (low/medium/high)

### Dashboard Builder

Create custom dashboards with widgets:

- Counter - Single metric with trend
- Line Chart - Time-series visualization
- Bar Chart - Categorical comparison
- Pie/Donut - Distribution view
- Table - Raw data listing
- Gauge - Progress tracking

## Quick Start

### Using Docker

```bash
# Start all services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

### Local Development

```bash
# Install dependencies
make deps

# Run in development mode
make dev
```

## API Reference

### Metrics

```bash
# List all metrics
curl http://localhost:3002/api/v1/datawatch/metrics

# Get metric details
curl http://localhost:3002/api/v1/datawatch/metrics/orders_events_total

# Query metric data
curl "http://localhost:3002/api/v1/datawatch/metrics/orders_amount_sum/query?from=2024-01-01T00:00:00Z&to=2024-01-02T00:00:00Z&aggregation=sum"

# Query with time steps
curl "http://localhost:3002/api/v1/datawatch/metrics/orders_events_total/range?from=2024-01-01T00:00:00Z&to=2024-01-02T00:00:00Z&step=1h&aggregation=sum"
```

### Anomalies

```bash
# List anomalies
curl http://localhost:3002/api/v1/datawatch/anomalies

# Get anomaly details
curl http://localhost:3002/api/v1/datawatch/anomalies/{id}

# Acknowledge anomaly
curl -X POST http://localhost:3002/api/v1/datawatch/anomalies/{id}/acknowledge \
  -H "Content-Type: application/json" \
  -d '{"user": "admin@example.com"}'
```

### Dashboards

```bash
# Create dashboard
curl -X POST http://localhost:3002/api/v1/datawatch/dashboards \
  -H "Content-Type: application/json" \
  -d '{"name": "E-commerce Overview", "description": "Main KPIs"}'

# Add widget
curl -X POST http://localhost:3002/api/v1/datawatch/dashboards/{id}/widgets \
  -H "Content-Type: application/json" \
  -d '{
    "type": "counter",
    "title": "Orders Today",
    "metric": "orders_events_total",
    "position": {"x": 0, "y": 0, "width": 4, "height": 2}
  }'
```

### CDC Event Ingestion

```bash
# Ingest single event
curl -X POST http://localhost:3002/api/v1/datawatch/events \
  -H "Content-Type: application/json" \
  -d '{
    "id": "evt_123",
    "type": "INSERT",
    "table": "orders",
    "schema": "public",
    "timestamp": "2024-01-15T10:30:00Z",
    "after": {
      "id": 1001,
      "amount": 99.99,
      "status": "pending",
      "customer_id": 42
    }
  }'

# Batch ingestion
curl -X POST http://localhost:3002/api/v1/datawatch/events/batch \
  -H "Content-Type: application/json" \
  -d '[
    {"id": "evt_124", "type": "INSERT", "table": "orders", ...},
    {"id": "evt_125", "type": "UPDATE", "table": "orders", ...}
  ]'
```

## Configuration

```yaml
# datawatch.yaml
server:
  port: 3002
  environment: production

metrics:
  auto_discover: true
  retention: 720h  # 30 days
  storage: embedded  # embedded, prometheus, influxdb, timescale

anomaly:
  enabled: true
  algorithms:
    - statistical
    - seasonal
  sensitivity: medium  # low, medium, high
  baseline_window: 168h  # 7 days

alerts:
  evaluation_interval: 1m
  channels:
    slack:
      webhook_url: ${SLACK_WEBHOOK_URL}
    email:
      smtp_host: smtp.example.com
      from: alerts@example.com
```

## Integration with Savegress

DataWatch integrates seamlessly with Savegress CDC:

```go
// In Savegress pipeline configuration
pipeline:
  source: postgres
  sink: datawatch
  datawatch:
    url: http://datawatch:3002
    batch_size: 100
    flush_interval: 1s
```

## Tech Stack

- **Language**: Go 1.23
- **Storage**: SQLite (embedded), PostgreSQL (optional)
- **Cache**: Redis (optional)
- **API**: Chi router, REST

## License

Proprietary - Part of Savegress Add-ons

---

Built with ❤️ by the Savegress team
