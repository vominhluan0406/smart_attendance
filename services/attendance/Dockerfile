# Build stage
FROM golang:1.22-alpine AS builder


WORKDIR /app

# Copy shared module first (for dependency resolution)
COPY shared/ ./shared/

# Copy attendance service
COPY services/attendance/ ./services/attendance/

WORKDIR /app/services/attendance


# Build binary (using vendored dependencies)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-s -w" -o /attendance-service ./cmd/main.go

# Runtime stage
FROM alpine:3.20

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENV TZ=Asia/Ho_Chi_Minh

WORKDIR /app

COPY --from=builder /attendance-service .

EXPOSE 8082

ENTRYPOINT ["./attendance-service"]
