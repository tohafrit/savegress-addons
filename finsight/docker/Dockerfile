# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /finsight ./cmd/finsight

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /finsight /app/finsight
COPY docker/finsight.yaml /etc/finsight/finsight.yaml

ENV FINSIGHT_CONFIG=/etc/finsight/finsight.yaml
ENV PORT=3004

EXPOSE 3004

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3004/health || exit 1

ENTRYPOINT ["/app/finsight"]
