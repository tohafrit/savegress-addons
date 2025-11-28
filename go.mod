module github.com/chainlens/chainlens

go 1.23

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-chi/cors v1.2.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.7.1
	github.com/redis/go-redis/v9 v9.7.0
	github.com/stripe/stripe-go/v76 v76.25.0
	github.com/ethereum/go-ethereum v1.14.0
	golang.org/x/crypto v0.28.0
)

// Worker pool is now internal
// Use: github.com/chainlens/chainlens/pkg/workerpool
