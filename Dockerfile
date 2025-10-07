# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build the API and migration binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o tusk ./cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o tusk_migrate ./cmd/migrate

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (e.g., for PostgreSQL)
RUN apk add --no-cache ca-certificates tzdata

# Copy binaries from builder
COPY --from=builder /app/tusk .
COPY --from=builder /app/tusk_migrate .

# Set permissions
RUN chmod +x tusk tusk_migrate

# Expose API port
EXPOSE 8081

# Command to run the API
CMD ["./tusk"]