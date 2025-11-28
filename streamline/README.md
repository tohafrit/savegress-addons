# StreamLine

Multi-channel e-commerce inventory and order synchronization engine for Savegress CDC platform.

## Features

- **Inventory Management**
  - Real-time stock synchronization across channels
  - Intelligent allocation strategies (fixed, weighted, priority, dynamic)
  - Safety stock buffers and low-stock alerts
  - Multi-warehouse support with transfers

- **Order Processing**
  - Automatic order routing optimization
  - Cost, speed, or balanced routing strategies
  - Order fulfillment tracking
  - Oversell prevention and handling

- **Pricing Engine**
  - Rule-based pricing with channel-specific adjustments
  - Minimum margin enforcement
  - Competitive pricing support
  - Bulk price updates

- **Channel Connectors**
  - Shopify integration
  - Amazon (planned)
  - WooCommerce (planned)
  - Custom connector API

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

StreamLine uses YAML configuration with environment variable substitution:

```yaml
server:
  port: 3003
  environment: ${ENVIRONMENT:-production}

inventory:
  sync_interval: 30s
  allocation_strategy: weighted
  safety_buffer: 10
  low_stock_threshold: 15

orders:
  routing_optimization: cost
  auto_route: true
  hold_oversold: true

pricing:
  auto_sync: true
  min_margin: 20
  competitive_pricing: true

channels:
  shopify:
    api_key: ${SHOPIFY_API_KEY}
    api_secret: ${SHOPIFY_API_SECRET}
    shop_domain: ${SHOPIFY_SHOP_DOMAIN}
    access_token: ${SHOPIFY_ACCESS_TOKEN}
```

## API Endpoints

### Inventory

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/streamline/inventory` | List all inventory |
| GET | `/api/v1/streamline/inventory/{sku}` | Get inventory by SKU |
| PUT | `/api/v1/streamline/inventory/{sku}` | Update inventory |
| POST | `/api/v1/streamline/inventory/{sku}/adjust` | Adjust stock quantity |
| POST | `/api/v1/streamline/inventory/{sku}/transfer` | Transfer between warehouses |
| GET | `/api/v1/streamline/inventory/low-stock` | Get low stock items |

### Orders

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/streamline/orders` | List orders |
| POST | `/api/v1/streamline/orders` | Create order |
| GET | `/api/v1/streamline/orders/{id}` | Get order details |
| POST | `/api/v1/streamline/orders/{id}/route` | Route order to warehouse |
| POST | `/api/v1/streamline/orders/{id}/assign` | Assign to specific warehouse |
| POST | `/api/v1/streamline/orders/{id}/fulfill` | Mark as fulfilled |
| POST | `/api/v1/streamline/orders/{id}/cancel` | Cancel order |
| GET | `/api/v1/streamline/orders/stats` | Get order statistics |

### Pricing

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/streamline/pricing/rules` | List pricing rules |
| POST | `/api/v1/streamline/pricing/rules` | Create pricing rule |
| DELETE | `/api/v1/streamline/pricing/rules/{id}` | Delete pricing rule |

### System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/streamline/stats` | System statistics |

## Allocation Strategies

### Fixed
Static percentage allocation per channel. Each channel gets a fixed portion of available stock.

### Weighted
Allocation based on channel performance metrics (sales velocity, conversion rate).

### Priority
Channels are ranked by priority. Higher priority channels get stock first.

### Dynamic
AI-driven allocation that adjusts based on:
- Historical sales data
- Current demand patterns
- Channel-specific margins
- Seasonality factors

## Order Routing

### Cost Optimization
Routes orders to minimize shipping and fulfillment costs.

### Speed Optimization
Routes orders to minimize delivery time to customer.

### Balanced
Considers both cost and speed with configurable weights.

## Integration with Savegress CDC

StreamLine consumes CDC events from the Savegress platform:

```go
// CDC event handler
func (e *Engine) HandleCDCEvent(event CDCEvent) {
    switch event.Table {
    case "products":
        e.syncProduct(event)
    case "inventory":
        e.syncInventory(event)
    case "orders":
        e.syncOrder(event)
    }
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      StreamLine                              │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  Inventory  │  │   Orders    │  │   Pricing   │         │
│  │   Engine    │  │  Processor  │  │   Engine    │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         │                │                │                 │
│  ┌──────┴────────────────┴────────────────┴──────┐         │
│  │              Channel Connectors                │         │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐       │         │
│  │  │ Shopify │  │ Amazon  │  │   ...   │       │         │
│  │  └─────────┘  └─────────┘  └─────────┘       │         │
│  └───────────────────────────────────────────────┘         │
├─────────────────────────────────────────────────────────────┤
│                    Savegress CDC Platform                    │
└─────────────────────────────────────────────────────────────┘
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `STREAMLINE_CONFIG` | Path to config file | - |
| `ENVIRONMENT` | Environment name | production |
| `JWT_SECRET` | JWT signing secret | - |
| `DATABASE_URL` | PostgreSQL connection URL | - |
| `REDIS_URL` | Redis connection URL | - |
| `SHOPIFY_API_KEY` | Shopify API key | - |
| `SHOPIFY_API_SECRET` | Shopify API secret | - |
| `SHOPIFY_SHOP_DOMAIN` | Shopify shop domain | - |
| `SHOPIFY_ACCESS_TOKEN` | Shopify access token | - |

## License

Proprietary - Savegress Platform
