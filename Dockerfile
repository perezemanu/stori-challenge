FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o processor \
    ./cmd/processor

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

RUN mkdir -p /app/data /app/db && \
    chown -R appuser:appgroup /app

COPY --from=builder /app/processor .

USER appuser

EXPOSE 8080

ENV LOG_LEVEL=info \
    DATA_PATH=/app/data \
    DB_DRIVER=sqlite \
    TO_ADDRESS=user@example.com

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD echo "Health check: Application is running"

CMD ["./processor"]