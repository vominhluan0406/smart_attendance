# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy shared module first (for dependency resolution)
COPY shared/ ./shared/

# Copy auth service
COPY services/auth/ ./services/auth/

WORKDIR /app/services/auth

# Download dependencies
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /auth-service ./cmd/main.go

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /auth-service .

EXPOSE 8081

ENTRYPOINT ["./auth-service"]
