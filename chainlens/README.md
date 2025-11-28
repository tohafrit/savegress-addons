# ChainLens

> Chrome DevTools for Blockchain Development

ChainLens is a Web3 code intelligence platform that provides real-time security analysis, gas estimation, transaction debugging, and contract monitoring for smart contract developers.

## Architecture

```
chainlens/
├── pkg/
│   └── workerpool/      # Worker Pool Engine (core)
│       ├── pool.go      # Main pool implementation
│       ├── enterprise/  # Enterprise features
│       │   ├── resilience/    # Circuit breaker, rate limiter, retry
│       │   ├── persistence/   # DLQ, persistent queue
│       │   ├── observability/ # Metrics, tracing, logging
│       │   └── multitenancy/  # Tenant isolation
│       └── examples/    # Usage examples
│
├── backend/             # API server (Go)
│   ├── cmd/chainlens/   # Entry point
│   ├── internal/
│   │   ├── api/         # HTTP handlers
│   │   ├── analyzer/    # Contract analysis (uses workerpool)
│   │   ├── blockchain/  # RPC integration
│   │   └── config/      # Configuration
│   └── migrations/      # Database schema
│
├── vscode-extension/    # VS Code extension (TypeScript)
│   ├── src/
│   │   ├── analyzers/   # Security & gas analysis
│   │   ├── providers/   # VS Code integration
│   │   └── services/    # API client
│   └── package.json
│
├── frontend/            # Web dashboard (Next.js)
│   └── app/             # App router pages
│
└── docker/              # Docker configuration
```

## Core: Worker Pool

ChainLens is powered by a production-grade worker pool engine with enterprise features:

```go
import "github.com/chainlens/chainlens/pkg/workerpool"

// Create pool with enterprise features
pool, _ := workerpool.NewWorkerPool(workerpool.Config{
    Workers:         8,
    QueueSize:       1000,
    ShutdownTimeout: 30 * time.Second,

    // Enterprise features
    RateLimiter:    workerpool.NewTokenBucket(100), // 100 req/sec
    CircuitBreaker: workerpool.NewCircuitBreaker(5, time.Minute),
    RetryPolicy:    workerpool.ExponentialBackoff(3),
})

// Submit analysis task
pool.Submit(func() error {
    return analyzeContract(source)
})
```

### Worker Pool Features

| Feature | Description |
|---------|-------------|
| **Rate Limiting** | Token bucket algorithm for RPC rate limits |
| **Circuit Breaker** | Automatic failure detection and recovery |
| **Retry with Backoff** | Exponential backoff for transient failures |
| **Dead Letter Queue** | Never lose failed tasks |
| **Multi-tenancy** | Tenant isolation for SaaS |
| **Metrics** | Prometheus-compatible metrics |
| **Graceful Shutdown** | Wait for in-flight tasks |

## Quick Start

### Prerequisites

- Go 1.23+
- Node.js 20+
- Docker & Docker Compose
- PostgreSQL 16 (or use Docker)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/chainlens/chainlens.git
cd chainlens

# Start infrastructure
make dev-infra

# Start backend
make dev-backend

# Start VS Code extension (in another terminal)
make dev-vscode
```

### Using Docker

```bash
# Start all services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

## Components

### VS Code Extension

Real-time analysis in your editor:
- Security vulnerability detection
- Gas estimation per function
- Inline warnings and suggestions
- Hover information for Solidity

```bash
cd vscode-extension
npm install
npm run compile

# Press F5 in VS Code to launch
```

### Backend API

Go-based API server with:
- Contract analysis engine
- Transaction tracing
- Fork simulation
- Monitoring & alerts

```bash
cd backend
go mod download
go run cmd/chainlens/main.go
```

### Frontend Dashboard

Next.js web application:
- Project management
- Contract monitoring
- Transaction debugger
- Analytics & reports

```bash
cd frontend
npm install
npm run dev
```

## Features

### Security Analysis

Detects common vulnerabilities:
- Reentrancy
- Integer overflow (pre-0.8.0)
- Access control issues
- Unchecked return values
- TX origin authentication
- Timestamp dependency
- Dangerous delegatecall
- Unprotected selfdestruct

### Gas Optimization

- Function-level gas estimates
- Optimization suggestions
- Deployment cost calculation
- Storage operation analysis

### Transaction Tracing

- Visual call graph
- State changes tracking
- Error location mapping
- Source code linking

### Contract Monitoring

- Real-time event streaming
- Alert configuration
- Webhook integration
- Multi-chain support

## API

### Authentication

```bash
# Register
curl -X POST http://localhost:3001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123","name":"Dev"}'

# Login
curl -X POST http://localhost:3001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'
```

### Analysis

```bash
# Analyze contract
curl -X POST http://localhost:3001/api/v1/analyze \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "pragma solidity ^0.8.0; contract Test { ... }",
    "compiler": "0.8.19",
    "options": {"security": true, "gas": true}
  }'
```

### Transaction Tracing

```bash
# Trace transaction
curl http://localhost:3001/api/v1/trace/0xabc123...?chain=ethereum \
  -H "Authorization: Bearer <token>"
```

## Configuration

### Environment Variables

```bash
# Server
PORT=3001
ENVIRONMENT=development

# Database
DATABASE_URL=postgres://chainlens:chainlens@localhost:5432/chainlens

# Redis
REDIS_URL=redis://localhost:6379

# Auth
JWT_SECRET=your-secret-key

# Blockchain
ETHEREUM_RPC_URL=https://eth.llamarpc.com
POLYGON_RPC_URL=https://polygon-rpc.com

# Stripe (optional)
STRIPE_SECRET_KEY=sk_...
```

## Pricing

| Tier | Price | Features |
|------|-------|----------|
| **Free** | $0 | VS Code extension, 1 project, local analysis |
| **Pro** | $49/mo | Cloud features, 10 projects, monitoring |
| **Team** | $149/mo/seat | Team collaboration, unlimited projects |
| **Enterprise** | Custom | SSO, on-premise, SLA |

## Tech Stack

- **VS Code Extension**: TypeScript, @solidity-parser/parser
- **Backend**: Go, Chi, pgx, go-ethereum
- **Frontend**: Next.js 14, Tailwind CSS, shadcn/ui
- **Database**: PostgreSQL 16
- **Cache**: Redis 7
- **Infrastructure**: Docker, worker-pool

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE)

---

Built with ❤️ by the ChainLens team

**Links:**
- Website: https://chainlens.dev
- Docs: https://docs.chainlens.dev
- Discord: https://discord.gg/chainlens
- Twitter: https://twitter.com/chainlens
