# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o monserv ./cmd/server/main.go

# Final Stage
FROM golang:1.23-alpine AS final
WORKDIR /app

# Copy binary dari builder
COPY --from=builder /build/monserv .

# Copy .env dan web templates
COPY .env .env
COPY web/ ./web/

# Install godotenv untuk load .env
RUN go install github.com/joho/godotenv/cmd/godotenv@latest

# Expose port untuk monitoring API
EXPOSE 18904

CMD ["sh", "-c", "godotenv ./monserv"]
