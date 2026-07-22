# Getting Started with Tusk

Welcome to **Tusk**, an opinionated, production-ready Go backend framework designed for building clear, modular, and maintainable API applications.

---

## Prerequisites

Before running Tusk, ensure you have the following installed on your system:

- **Go**: Version 1.20 or newer ([download Go](https://golang.org/dl/))
- **Database**: PostgreSQL or MySQL (or any database supported by GORM)
- **Make**: Standard build automation tool (optional, but recommended)
- **Air**: Hot-reloading daemon for Go development ([github.com/air-verse/air](https://github.com/air-verse/air))

---

## Installation & Environment Setup

### 1. Clone the Repository

```bash
git clone https://github.com/codetheuri/Tusk.git
cd Tusk
```

### 2. Configure Environment Variables

Copy the example environment configuration file:

```bash
cp .env.example .env
```

Edit `.env` to match your local environment settings:

```env
APP_NAME=Tusk
APP_ENV=development
APP_PORT=8080

# Database Configuration
DB_DRIVER=postgres
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=tusk_user
DB_PASSWORD=tusk_password
DB_NAME=tusk_db
DB_SSLMODE=disable

# Authentication & JWT
JWT_SECRET=super-secret-jwt-key-change-this-in-production
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
```

---

## Running the Application

Tusk includes a comprehensive `Makefile` to simplify common development tasks.

### Hot-Reloading Development Mode

Run the server with live hot-reloading using [Air](https://github.com/air-verse/air):

```bash
make dev
```

### Standard Development Mode

Run the application directly without hot-reloading:

```bash
make run
```

### Production Build

Build a standalone, optimized binary:

```bash
make build
```

This compiles the production executable to `./bin/api`. You can run it with:

```bash
./bin/api
```

---

## Testing & Quality Assurance

Run the test suite across all packages:

```bash
make test
```

Generate an HTML code coverage report:

```bash
make coverage
```

Run static analysis using `go vet`:

```bash
make vet
```

---

## Interactive API Documentation

Tusk automatically generates OpenAPI 3.0 specifications and serves an interactive API UI powered by Huma v2.

Once the application is running:

- **Interactive Documentation UI**: Visit `http://localhost:8080/docs` in your browser.
- **OpenAPI 3.0 JSON Spec**: Accessible at `http://localhost:8080/openapi.json`.

---

## Next Steps

- Explore the [Architecture Overview](architecture.md) to understand Tusk's design principles.
- Learn about [Routing & OpenAPI Integration](routing-and-api.md).
- Understand [Code-First RBAC Authorization](authorization.md).
