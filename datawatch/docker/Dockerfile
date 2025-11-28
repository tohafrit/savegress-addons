# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /datawatch ./cmd/datawatch

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates sqlite-libs tzdata

WORKDIR /app

# Create data directory
RUN mkdir -p /var/lib/datawatch/data

# Copy binary
COPY --from=builder /datawatch /app/datawatch

# Copy default config
COPY docker/datawatch.yaml /etc/datawatch/datawatch.yaml

# Environment
ENV DATAWATCH_CONFIG=/etc/datawatch/datawatch.yaml
ENV PORT=3002

EXPOSE 3002

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3002/health || exit 1

ENTRYPOINT ["/app/datawatch"]
