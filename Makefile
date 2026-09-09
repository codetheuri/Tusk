.PHONY: dev run build test test-unit test-integration test-db-setup coverage vet migrate-up migrate-down migrate-reset migrate-status auth-sync auth-sync-prune clean help

# ==============================================================================
# Development commands

## dev: run the application with hot-reloading using Air
dev:
	@echo "Starting hot-reload server..."
	@go run github.com/air-verse/air@latest

## run: run the application without hot-reloading
run:
	go run ./cmd/api/main.go

## build: build the production binary
build:
	@echo "Building production binary..."
	go build -o ./bin/api ./cmd/api/main.go
	@echo "Built ./bin/api successfully!"

## test: run all tests (integration tests skip if no database is reachable)
test:
	go test -race ./...

## test-unit: run only tests that need no database
test-unit:
	go test -race -short ./config/... ./pkg/... ./internal/middleware/...

## test-integration: run all tests and FAIL if the test database is unreachable
## Requires PostgreSQL. Override the target with TEST_DATABASE_URL.
test-integration:
	REQUIRE_DB_TESTS=1 go test -race -count=1 ./...

## test-db-setup: create the local test database (one-off)
test-db-setup:
	@psql "$${TEST_ADMIN_URL:-postgres://root:root@127.0.0.1:5434/postgres}" \
		-c "CREATE DATABASE tusk_test" 2>/dev/null && echo "Created tusk_test" \
		|| echo "tusk_test already exists (or psql is unavailable)"

## coverage: run tests and generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## vet: run static analysis
vet:
	go vet ./...

# ==============================================================================
# Database Migrations

## migrate-up: apply all pending database migrations
migrate-up:
	go run ./cmd/migrate/main.go up

## migrate-down: revert the last database migration 
migrate-down:
	go run ./cmd/migrate/main.go down

## migrate-reset: revert every migration, dropping the schema
migrate-reset:
	go run ./cmd/migrate/main.go reset

## migrate-status: check the status of database migrations
migrate-status:
	go run ./cmd/migrate/main.go status

## auth-sync: sync code permissions to database
auth-sync:
	go run ./cmd/tusk/main.go auth sync

## auth-sync-prune: sync code permissions and prune obsolete database permissions
auth-sync-prune:
	go run ./cmd/tusk/main.go auth sync --prune

# ==============================================================================
# Utilities

## clean: remove binary and temporary files
clean:
	rm -rf ./tmp ./bin build-errors.log coverage.out coverage.html

## help: print this help message
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

