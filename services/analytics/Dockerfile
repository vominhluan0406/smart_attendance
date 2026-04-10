# Build stage
FROM golang:1.22-alpine AS builder


WORKDIR /app

# Copy shared module first (for dependency resolution)
COPY shared/ ./shared/

# Copy analytics service
COPY services/analytics/ ./services/analytics/

WORKDIR /app/services/analytics


# Build binary (using vendored dependencies)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /analytics-service ./cmd/main.go

# Runtime stage
FROM alpine:3.20

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app

COPY --from=builder /analytics-service .

EXPOSE 8084

ENTRYPOINT ["./analytics-service"]
