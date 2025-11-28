# IoTSense

IoT Device Telemetry Platform for the Savegress CDC ecosystem. Manages device registration, telemetry ingestion, and real-time alerting.

## Features

- **Device Management**: Register, monitor, and manage IoT devices with automatic heartbeat detection
- **Telemetry Ingestion**: High-throughput time-series data collection with buffering and batching
- **Device Groups**: Organize devices into logical groups for management and alerting
- **Alert Rules**: Configurable threshold-based alerting with multiple notification channels
- **Commands**: Send commands to devices and track their execution status
- **Time-Series Storage**: In-memory storage with aggregation (extensible to TimescaleDB/InfluxDB)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        IoTSense                              │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Device    │  │  Telemetry  │  │   Alerts    │         │
│  │  Registry   │  │   Engine    │  │   Engine    │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│         │                │                │                  │
│         └────────────────┼────────────────┘                 │
│                          │                                   │
│                   ┌──────┴──────┐                           │
│                   │  REST API   │                           │
│                   └─────────────┘                           │
├─────────────────────────────────────────────────────────────┤
│  Port: 3006                                                  │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Local Development

```bash
# Build and run
make run

# Or with environment config
ENVIRONMENT=development PORT=3006 make run
```

### Docker

```bash
# Build and run with Docker Compose
make docker-run

# View logs
make docker-logs

# Stop
make docker-stop
```

## API Endpoints

### Devices

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/devices` | List all devices |
| POST | `/api/v1/devices` | Register a new device |
| GET | `/api/v1/devices/{id}` | Get device details |
| PUT | `/api/v1/devices/{id}` | Update device |
| DELETE | `/api/v1/devices/{id}` | Unregister device |
| POST | `/api/v1/devices/{id}/heartbeat` | Send heartbeat |

### Device Groups

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/groups` | List all groups |
| POST | `/api/v1/groups` | Create a group |
| GET | `/api/v1/groups/{id}` | Get group details |
| GET | `/api/v1/groups/{id}/devices` | Get devices in group |
| POST | `/api/v1/groups/{id}/devices/{deviceId}` | Add device to group |
| DELETE | `/api/v1/groups/{id}/devices/{deviceId}` | Remove device from group |

### Telemetry

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/telemetry` | Ingest single point |
| POST | `/api/v1/telemetry/batch` | Ingest batch of points |
| GET | `/api/v1/telemetry/query` | Query time-series data |
| GET | `/api/v1/telemetry/latest/{deviceId}/{metric}` | Get latest value |
| GET | `/api/v1/telemetry/aggregated/{deviceId}/{metric}` | Get aggregated data |

### Commands

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/commands` | Send command to device |
| GET | `/api/v1/commands/{id}` | Get command status |
| PUT | `/api/v1/commands/{id}/status` | Update command status |

### Alert Rules

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/rules` | List all rules |
| POST | `/api/v1/rules` | Create a rule |
| GET | `/api/v1/rules/{id}` | Get rule details |
| PUT | `/api/v1/rules/{id}` | Update rule |
| DELETE | `/api/v1/rules/{id}` | Delete rule |

### Alerts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/alerts` | List alerts |
| GET | `/api/v1/alerts/{id}` | Get alert details |
| POST | `/api/v1/alerts/{id}/acknowledge` | Acknowledge alert |
| POST | `/api/v1/alerts/{id}/resolve` | Resolve alert |

## Usage Examples

### Register a Device

```bash
curl -X POST http://localhost:3006/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Temperature Sensor 1",
    "type": "sensor",
    "model": "TMP36",
    "manufacturer": "Analog Devices",
    "location": {
      "building": "Factory A",
      "floor": "2",
      "room": "Production"
    },
    "tags": {
      "zone": "production",
      "priority": "high"
    }
  }'
```

### Send Telemetry Data

```bash
# Single point
curl -X POST http://localhost:3006/api/v1/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "metric": "temperature",
    "value": 23.5,
    "unit": "celsius",
    "quality": "good"
  }'

# Batch
curl -X POST http://localhost:3006/api/v1/telemetry/batch \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-uuid",
    "points": [
      {"metric": "temperature", "value": 23.5, "unit": "celsius"},
      {"metric": "humidity", "value": 45.2, "unit": "percent"},
      {"metric": "pressure", "value": 1013.25, "unit": "hPa"}
    ]
  }'
```

### Create an Alert Rule

```bash
curl -X POST http://localhost:3006/api/v1/rules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Temperature Alert",
    "metric": "temperature",
    "condition": {
      "type": "threshold",
      "operator": "gt",
      "value": 35
    },
    "severity": "warning",
    "cooldown": "5m",
    "enabled": true,
    "actions": [
      {"type": "slack", "target": "#alerts"},
      {"type": "webhook", "target": "https://example.com/webhook"}
    ]
  }'
```

### Query Telemetry Data

```bash
# Get time-series data
curl "http://localhost:3006/api/v1/telemetry/query?device_id=xxx&metric=temperature&start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z"

# Get aggregated data
curl "http://localhost:3006/api/v1/telemetry/aggregated/device-uuid/temperature?start=2024-01-01T00:00:00Z&end=2024-01-02T00:00:00Z&interval=1h"
```

## Configuration

Configuration via environment variables or YAML file:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | 3006 |
| `ENVIRONMENT` | Environment (development/production) | development |
| `DATABASE_URL` | PostgreSQL connection string | postgres://... |
| `REDIS_URL` | Redis connection string | redis://... |
| `TELEMETRY_BATCH_SIZE` | Telemetry batch size | 1000 |
| `TELEMETRY_BUFFER_SIZE` | Telemetry buffer size | 100000 |
| `DEVICE_HEARTBEAT` | Heartbeat check interval | 30s |
| `DEVICE_OFFLINE_THRESHOLD` | Time before device is marked offline | 5m |
| `MAX_DEVICES` | Maximum devices limit | 10000 |
| `ALERTS_ENABLED` | Enable alerting | true |
| `SLACK_WEBHOOK_URL` | Slack webhook for alerts | - |
| `ALERT_WEBHOOK_URL` | Custom webhook for alerts | - |

## Device Types

- `sensor` - Sensors (temperature, humidity, pressure, etc.)
- `actuator` - Actuators (relays, valves, motors)
- `gateway` - IoT gateways
- `controller` - Controllers (PLCs, industrial controllers)
- `camera` - IP cameras
- `meter` - Smart meters

## Alert Severities

- `info` - Informational alerts
- `warning` - Warning alerts
- `error` - Error alerts
- `critical` - Critical alerts requiring immediate attention

## Integration with Savegress CDC

IoTSense integrates with the Savegress CDC platform for:

1. **Real-time Streaming**: Device events and telemetry can be captured via CDC
2. **Data Synchronization**: Device state synchronized across systems
3. **Event Processing**: Complex event processing with DataWatch
4. **Analytics**: Historical analysis with FinSight

## License

Proprietary - Savegress Platform
