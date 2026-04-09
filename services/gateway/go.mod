module github.com/smart-attendance/gateway

go 1.22

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/joho/godotenv v1.5.1
	github.com/smart-attendance/shared v0.0.0
	golang.org/x/time v0.9.0
)

replace github.com/smart-attendance/shared => ../../shared
