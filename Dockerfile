# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o tusk cmd/main.go

# RUN CGO_ENABLED=0 GOOS=linux go build -o /usr/local/bin/tusk_migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o tusk_migrate ./cmd/migrate
# Runtime stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/tusk .
COPY --from=builder /app/tusk_migrate .
COPY --from=builder /app/.env .

EXPOSE 8081

CMD ["./tusk"]