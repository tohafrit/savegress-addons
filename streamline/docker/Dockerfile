# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /streamline ./cmd/streamline

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /streamline /app/streamline
COPY docker/streamline.yaml /etc/streamline/streamline.yaml

ENV STREAMLINE_CONFIG=/etc/streamline/streamline.yaml
ENV PORT=3003

EXPOSE 3003

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3003/health || exit 1

ENTRYPOINT ["/app/streamline"]
