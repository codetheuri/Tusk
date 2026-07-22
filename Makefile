.PHONY: dev run build test coverage vet migrate-up migrate-down migrate-status clean help

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

## test: run unit tests across all packages
test:
	go test -v -race ./...

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

