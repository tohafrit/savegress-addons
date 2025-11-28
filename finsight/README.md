# FinSight

Financial transaction analytics and fraud detection engine for Savegress CDC platform.

## Features

- **Transaction Processing**
  - Real-time transaction ingestion and processing
  - Automatic categorization using MCC codes and ML
  - Multi-currency support
  - Account balance tracking

- **Fraud Detection**
  - Real-time risk scoring
  - Velocity tracking
  - Geolocation analysis (impossible travel)
  - Pattern detection (card testing, round amounts)
  - Configurable rules engine
  - Alert management workflow

- **Reconciliation**
  - Automated transaction matching
  - Multiple matching strategies (exact, fuzzy, reference ID)
  - Exception handling and resolution
  - Batch processing

- **Reporting**
  - Transaction summaries
  - Cash flow analysis
  - Fraud metrics
  - Scheduled report generation
  - Multiple export formats (PDF, CSV, XLSX)

- **Compliance**
  - AML screening
  - SAR/CTR threshold monitoring
  - Watchlist screening
  - Audit logging

## Quick Start

### Using Docker Compose

```bash
# Start all services
make docker-run

# View logs
make docker-logs

# Stop services
make docker-stop
```

### Local Development

```bash
# Install dependencies
make deps
make tidy

# Run in development mode
make dev

# Run tests
make test
```

## Configuration

FinSight uses YAML configuration with environment variable substitution:

```yaml
server:
  port: 3004
  environment: ${ENVIRONMENT:-production}

transactions:
  batch_size: 1000
  process_interval: 5s
  retention_days: 365
  categorization_enabled: true

fraud:
  enabled: true
  realtime_scoring: true
  score_threshold: 0.7
  velocity_window: 1h
  max_daily_amount: 10000
  max_single_amount: 5000
  geofencing_enabled: true

reconciliation:
  auto_reconcile: true
  match_tolerance: 0.01
  date_tolerance: 24h

compliance:
  aml_enabled: true
  sar_threshold: 5000
  ctr_threshold: 10000
```

## API Endpoints

### Transactions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/finsight/transactions` | List transactions |
| POST | `/api/v1/finsight/transactions` | Create transaction |
| GET | `/api/v1/finsight/transactions/{id}` | Get transaction |
| PUT | `/api/v1/finsight/transactions/{id}` | Update transaction |
| GET | `/api/v1/finsight/transactions/stats` | Get statistics |

### Accounts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/finsight/accounts` | List accounts |
| POST | `/api/v1/finsight/accounts` | Create account |
| GET | `/api/v1/finsight/accounts/{id}` | Get account |
| GET | `/api/v1/finsight/accounts/{id}/transactions` | Get account transactions |
| GET | `/api/v1/finsight/accounts/{id}/balance` | Get account balance |

### Fraud Detection

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/finsight/fraud/alerts` | List alerts |
| GET | `/api/v1/finsight/fraud/alerts/{id}` | Get alert |
| POST | `/api/v1/finsight/fraud/alerts/{id}/resolve` | Resolve alert |
| POST | `/api/v1/finsight/fraud/evaluate` | Evaluate transaction |
| GET | `/api/v1/finsight/fraud/stats` | Get fraud stats |

### Reconciliation

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/finsight/reconciliation/batches` | List batches |
| POST | `/api/v1/finsight/reconciliation/batches` | Create batch |
| GET | `/api/v1/finsight/reconciliation/batches/{id}` | Get batch |
| POST | `/api/v1/finsight/reconciliation/batches/{id}/run` | Run reconciliation |
| GET | `/api/v1/finsight/reconciliation/batches/{id}/exceptions` | Get exceptions |
| POST | `/api/v1/finsight/reconciliation/exceptions/{id}/resolve` | Resolve exception |
| GET | `/api/v1/finsight/reconciliation/stats` | Get stats |

### Reports

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/finsight/reports` | List reports |
| POST | `/api/v1/finsight/reports` | Create report |
| GET | `/api/v1/finsight/reports/{id}` | Get report |
| POST | `/api/v1/finsight/reports/{id}/generate` | Generate report |
| DELETE | `/api/v1/finsight/reports/{id}` | Delete report |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/finsight/stats` | Overall statistics |

## Fraud Detection Rules

### Amount Rule
Detects unusual transaction amounts:
- Absolute maximum threshold
- Deviation from account average

### Velocity Rule
Detects unusual transaction velocity:
- High transaction count in short period
- Daily amount limit exceeded
- Transactions from multiple locations

### Geolocation Rule
Detects location-based anomalies:
- High-risk countries
- Unusual locations for account
- Impossible travel (physical impossibility based on time/distance)

### Pattern Rule
Detects suspicious patterns:
- Large round amounts
- Repeated identical amounts
- Card testing (multiple small amounts)

### Time Rule
Detects unusual transaction times:
- Night-time transactions
- Outside typical hours for account

### Merchant Rule
Detects suspicious merchant activity:
- High-risk MCC codes
- New merchants for established accounts

## Reconciliation Matchers

### Exact Matcher
Matches transactions exactly by:
- External ID
- Amount + Date

### Fuzzy Matcher
Matches with configurable tolerance:
- Amount within tolerance percentage
- Date within tolerance window

### Reference ID Matcher
Matches by reference identifiers:
- External IDs
- Embedded reference patterns
- Metadata reference fields

## Integration with Savegress CDC

FinSight consumes CDC events from the Savegress platform:

```go
// CDC event handler
func (e *Engine) HandleCDCEvent(event CDCEvent) {
    switch event.Table {
    case "transactions":
        e.processTransaction(event)
    case "accounts":
        e.updateAccount(event)
    }
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       FinSight                               │
├─────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Transaction  │  │    Fraud     │  │Reconciliation│      │
│  │   Engine     │  │  Detector    │  │   Engine     │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                 │               │
│  ┌──────┴─────────────────┴─────────────────┴──────┐       │
│  │                  Reporting Engine                │       │
│  └──────────────────────────────────────────────────┘       │
├─────────────────────────────────────────────────────────────┤
│                   Compliance & Audit Layer                   │
├─────────────────────────────────────────────────────────────┤
│                    Savegress CDC Platform                    │
└─────────────────────────────────────────────────────────────┘
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FINSIGHT_CONFIG` | Path to config file | - |
| `ENVIRONMENT` | Environment name | production |
| `JWT_SECRET` | JWT signing secret | - |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `REDIS_URL` | Redis connection URL | - |
| `SLACK_WEBHOOK_URL` | Slack webhook for alerts | - |
| `SMTP_HOST` | SMTP server host | - |
| `PAGERDUTY_KEY` | PagerDuty service key | - |

## License

Proprietary - Savegress Platform
