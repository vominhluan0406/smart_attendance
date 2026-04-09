# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy shared module first (for dependency resolution)
COPY shared/ ./shared/

# Copy attendance service
COPY services/attendance/ ./services/attendance/

WORKDIR /app/services/attendance

# Download dependencies
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /attendance-service ./cmd/main.go

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /attendance-service .

EXPOSE 8082

ENTRYPOINT ["./attendance-service"]
