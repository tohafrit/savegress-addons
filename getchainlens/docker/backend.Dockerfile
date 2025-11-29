# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all source code (backend depends on pkg/)
COPY backend ./backend
COPY pkg ./pkg

# Build arguments
ARG VERSION=dev

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X 'main.Version=${VERSION}'" \
    -o /app/chainlens ./backend/cmd/chainlens

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S chainlens && \
    adduser -u 1000 -S chainlens -G chainlens

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/chainlens /app/chainlens

# Copy migrations
COPY backend/migrations /app/migrations

# Set ownership
RUN chown -R chainlens:chainlens /app

# Switch to non-root user
USER chainlens

# Expose port
EXPOSE 3001

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

# Run the application
ENTRYPOINT ["/app/chainlens"]
