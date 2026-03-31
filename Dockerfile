# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o smart-attendance cmd/server/main.go

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/smart-attendance .
COPY --from=builder /app/web ./web
COPY --from=builder /app/.env.example ./.env.example

RUN mkdir -p data

EXPOSE 8080

CMD ["./smart-attendance"]
