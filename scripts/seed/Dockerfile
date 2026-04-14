# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy seed application code
COPY scripts/seed/ ./scripts/seed/

WORKDIR /app/scripts/seed

# Download dependencies
RUN go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /seed-script main.go

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /seed-script .

ENTRYPOINT ["./seed-script"]
