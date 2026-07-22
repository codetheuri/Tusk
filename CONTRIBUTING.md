# Contributing to Tusk

Thank you for contributing to **Tusk**! Tusk is an opinionated, highly maintainable Go framework built on idiomatic Go standards, clean architecture, and explicit dependency injection.

---

## 🏛️ Architectural Principles

When extending Tusk or adding new features, strictly adhere to these core rules:

1. **Keep it Go-like**: Avoid Laravel/Spring Boot style reflection magic, hidden magic, or global state.
2. **Explicit Constructor DI**: Pass all dependencies (loggers, database connections, services) explicitly via constructors.
3. **No Business Logic in Handlers**: Handlers only bind input, call services, and return responses.
4. **HTTP-Agnostic Services**: Services operate strictly on domain structs and return domain errors. They do not know about HTTP status codes or headers.
5. **Repositories Only Handle Storage**: Repositories interact directly with database engines using `context.Context`.
6. **Package Placement Rules**:
   - Reusable infrastructure packages live in `pkg/` (e.g. `pkg/cache`, `pkg/queue`, `pkg/storage`, `pkg/events`).
   - Domain business modules live in `internal/<domain>` (e.g. `internal/auth`, `internal/audit`).
   - Server platform & lifecycle setups live in `internal/platform/` (e.g. `internal/platform/database`, `internal/platform/telemetry`).

---

## 🛠️ Development Workflow

### 1. Prerequisites
- Go 1.20+
- PostgreSQL or MySQL
- Air (`go install github.com/air-verse/air@latest`)

### 2. Local Environment Setup
```bash
git clone https://github.com/codetheuri/Tusk.git
cd Tusk
cp .env.example .env
make migrate-up
make auth-sync
make dev
```

### 3. Testing & Verification
Before submitting a pull request, run all verification targets:
```bash
make test
make vet
make coverage
```

---

## 📝 Documentation Rules

- Every exported function, type, struct, and package **must** have clear Go-doc comments.
- Code should teach developers *why* something exists.
- Update relevant guides in [`docs/`](docs/) and [`CHANGELOG.md`](CHANGELOG.md) when introducing new capabilities.
