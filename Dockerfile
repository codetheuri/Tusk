# # Build stage
# FROM golang:1.24-alpine AS builder

# WORKDIR /app

# # Install build dependencies
# RUN apk add --no-cache git

# # Copy go mod files
# COPY go.mod go.sum ./
# RUN go mod download

# # Copy source files
# COPY . .

# # Build the API and migration binaries
# RUN CGO_ENABLED=0 GOOS=linux go build -o tusk ./cmd/main.go
# RUN CGO_ENABLED=0 GOOS=linux go build -o tusk_migrate ./cmd/migrate

# # Runtime stage
# FROM alpine:latest

# WORKDIR /app

# # Install runtime dependencies (e.g., for PostgreSQL)
# RUN apk add --no-cache ca-certificates tzdata

# # Copy binaries from builder
# COPY --from=builder /app/tusk .
# COPY --from=builder /app/tusk_migrate .

# # Set permissions
# RUN chmod +x tusk tusk_migrate

# # Expose API port
# EXPOSE 8081

# # Command to run the API
# CMD ["./tusk"]
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

# Debug: List files to see structure
RUN echo "=== Current directory structure ===" && \
    find . -type f -name "*.go" | head -20 && \
    echo "=== Listing cmd directory ===" && \
    ls -la cmd/ || echo "No cmd directory"

# Build the API and migration binaries with explicit paths
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/tusk ./cmd/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/tusk_migrate ./cmd/migrate

# Verify binaries were created
RUN echo "=== Built binaries ===" && \
    ls -la /app/tusk* && \
    file /app/tusk || echo "tusk binary not found"

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binaries from builder
COPY --from=builder /app/tusk .
COPY --from=builder /app/tusk_migrate .

# Debug: Verify binaries in runtime image
RUN echo "=== Runtime image binaries ===" && \
    ls -la /app/ && \
    file /app/tusk || echo "tusk binary missing"

# Set permissions
RUN chmod +x tusk tusk_migrate

# Expose API port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

# Command to run the API
CMD ["./tusk"]