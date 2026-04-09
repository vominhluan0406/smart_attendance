# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy shared module first (for dependency resolution)
COPY shared/ ./shared/

# Copy organization service
COPY services/organization/ ./services/organization/

WORKDIR /app/services/organization

# Download dependencies
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /organization-service ./cmd/main.go

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /organization-service .

EXPOSE 8085

ENTRYPOINT ["./organization-service"]
