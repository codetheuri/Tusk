# 🐘 Tusk - The Sharpest Go Backend Framework for APIs

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.20-blue.svg)](https://golang.org/doc/go1.20)
[![GitHub release](https://img.shields.io/github/v/release/codetheuri/Tusk?include_prereleases)](https://github.com/codetheuri/Tusk/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/codetheuri/Tusk/go.yml?branch=main)](https://github.com/codetheuri/Tusk/actions)
[![GitHub stars](https://img.shields.io/github/stars/codetheuri/Tusk?style=social)](https://github.com/codetheuri/Tusk/stargazers)

**Tusk** is an opinionated, production-ready Go backend framework designed for building scalable, maintainable, and type-safe REST APIs. It enforces explicit dependency injection, clean layered architecture, code-first authorization (RBAC), and automatic OpenAPI 3.0 documentation generation.

---

## 📚 Documentation Index

Explore the full documentation guides in the [`docs/`](docs/) directory:

- 🚀 **[Getting Started](docs/getting-started.md)** - Prerequisites, environment configuration, and running the server.
- 🏗️ **[Architecture & Design Philosophy](docs/architecture.md)** - Layered architecture, dependency injection, and coding guidelines.
- ⚡ **[Routing & OpenAPI (Huma v2)](docs/routing-and-api.md)** - Strongly-typed DTOs, route definition, and auto-generated API docs.
- 🔐 **[Code-First RBAC Authorization](docs/authorization.md)** - Code permissions, database sync CLI (`tusk auth sync`), and route guards.
- 🗄️ **[Database & Migrations](docs/database-and-migrations.md)** - GORM connectivity, seeder tools, and schema migration CLI (`cmd/migrate`).
- 🔍 **[Querying, Filtering & Pagination](docs/querying-and-pagination.md)** - Dynamic searching, sorting, field filtering, and metadata envelopes (`pkg/query`).
- 📬 **[Standardized Responses & Error Handling](docs/responses-and-errors.md)** - Uniform JSON response structure (`pkg/response`) and status code conventions.

---

## ✨ Features

- **Huma v2 + Chi Router**: Strongly-typed HTTP handlers with automatic OpenAPI 3.0 spec generation (`/openapi.json`) and interactive documentation UI (`/docs`).
- **Clean Layered Architecture**: Strict separation of concerns (Handlers -> Services -> Repositories -> Models/DTOs).
- **Code-First RBAC Authorization (`pkg/authz`)**: Declare permissions in code, synchronize them to runtime database tables via `tusk auth sync`, and enforce permissions with route guards.
- **Authentication & Identity**: Complete authentication suite (Registration, Login, Refresh Tokens, Profile Management, Account Lockout after failed attempts) with bcrypt password hashing and JWT sessions.
- **GORM ORM & Database Migration CLI (`cmd/migrate`)**: Type-safe database operations with migration tools (`up`, `down`, `fresh`, `seed`).
- **Unified Querying & Pagination Engine (`pkg/query`)**: Standardized searching, sorting, and pagination metadata envelopes (`query.Meta`).
- **Standardized API Response Envelopes (`pkg/response`)**: Consistent JSON responses across all endpoints.
- **Structured Logging (`pkg/logger`)**: Console and production logger setup.
- **Transactional Seeding & Mailer Integration (`pkg/mailer`)**: Email sending helper utilities and database seeder support.

---

## 🚀 Quick Start

### 1. Prerequisites

- **Go**: 1.20 or newer
- **Database**: PostgreSQL or MySQL

### 2. Setup & Configuration

```bash
# Clone the repository
git clone https://github.com/codetheuri/Tusk.git
cd Tusk

# Copy environment configuration
cp .env.example .env

# Run database migrations
make migrate-up

# Sync code authorization permissions to database
make auth-sync

# Start hot-reload server (with Air)
make dev
```

The server will start at `http://localhost:8080`.

- **Interactive API Documentation UI**: `http://localhost:8080/docs`
- **OpenAPI 3.0 JSON Spec**: `http://localhost:8080/openapi.json`

---

## 🛠️ CLI Commands & Makefile Utilities

Tusk includes a complete `Makefile` for common tasks:

| Command | Description |
| :--- | :--- |
| `make dev` | Start development server with hot-reloading (via Air) |
| `make run` | Run application without hot-reloading (`cmd/api/main.go`) |
| `make build` | Compile optimized production binary to `./bin/api` |
| `make test` | Run unit tests across all packages |
| `make coverage` | Generate HTML test coverage report (`coverage.html`) |
| `make vet` | Run Go static code analysis |
| `make migrate-up` | Apply pending database schema migrations |
| `make migrate-down` | Roll back the last database migration |
| `make auth-sync` | Synchronize code-declared permissions into the database |
| `make auth-sync-prune` | Synchronize code permissions and prune obsolete database records |
| `make clean` | Clean build artifacts and temporary log files |

---

## 🏛️ Project Directory Structure

```
Tusk/
├── cmd/
│   ├── api/             # HTTP API entrypoint (main.go)
│   ├── migrate/         # Database migration & seeder CLI
│   └── tusk/            # Framework CLI tool (permission sync)
├── config/              # Configuration loader (.env parsing)
├── database/            # Schema migrations and database seeders
├── docs/                # Comprehensive technical documentation
├── internal/
│   ├── app/             # Application lifecycle, routing, and container wiring
│   ├── auth/            # Authentication & RBAC identity domain module
│   ├── middleware/      # Global HTTP middleware (CORS, JWT authentication)
│   └── platform/        # Infrastructure setup (GORM DB connection)
└── pkg/
    ├── authz/           # Code-first authorization & RBAC engine
    ├── logger/          # Structured logging package
    ├── mailer/          # Email transmission utility package
    ├── query/           # Pagination, filtering, & sorting query engine
    ├── response/        # Standardized API response builder
    └── validate/        # Input validation utilities
```

---

## 📖 Learn More

Read the full documentation guides in the [`docs/`](docs/) directory to master building applications with Tusk.