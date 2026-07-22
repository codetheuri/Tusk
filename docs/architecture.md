# Tusk Architecture & Design Philosophy

Tusk is designed to feel like pure, idiomatic Go—not like Laravel, Spring Boot, or ASP.NET. It prioritizes clarity, explicit dependency management, and domain encapsulation over framework magic, hidden reflection, or deep inheritance hierarchies.

---

## Core Architectural Principles

1. **Idiomatic Go Over Framework Magic**: Standard library patterns are preferred. Code should be explicit and easy to trace from entrypoint to database.
2. **Explicit Dependency Injection**: All dependencies (repositories, services, loggers) are passed explicitly via constructors (`NewService(...)`, `NewHandler(...)`).
3. **Strict Separation of Concerns**: Each architectural layer has a single, clear responsibility.
4. **No Business Logic in Handlers**: Handlers only bind and validate input, delegate to services, and return responses.
5. **HTTP Agnostic Services**: Domain services know nothing about HTTP requests, headers, or status codes. They operate strictly on domain data and return domain errors.
6. **Isolated Repository Layer**: Repositories interact directly with storage (GORM/SQL) and enforce transaction boundaries.

---

## Layer Responsibilities

```
┌──────────────────────────────────────────────────────────┐
│                   HTTP Request / Response                │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│                      Handler Layer                       │
│  - Bind & validate HTTP inputs (Huma v2 DTOs)            │
│  - Call service layer methods                            │
│  - Return standardized HTTP responses                    │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│                      Service Layer                       │
│  - Implement core business logic and rules               │
│  - Coordinate database transactions & repositories        │
│  - Return domain entities & domain errors                │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│                     Repository Layer                     │
│  - Execute SQL / GORM storage operations                 │
│  - Handle query parameter mapping                        │
│  - Return database entities / raw errors                 │
└────────────────────────────┬─────────────────────────────┘
```

### 1. Handler Layer (`internal/auth/handler_*.go`)
Handlers expose domain capabilities over HTTP. They consume input DTOs, perform structural validation, delegate processing to services, and map results to response DTOs. Handlers never write raw SQL or apply business rules.

### 2. Service Layer (`internal/auth/service_*.go`)
Services contain core business rules (e.g., password verification, token generation, user lockout rules, permission validation). Services receive domain inputs and return domain structs or errors.

### 3. Repository Layer (`internal/auth/repository_*.go`)
Repositories interface directly with GORM and the underlying database. They accept `context.Context` parameters for proper cancellation propagation and execute queries, updates, and deletes.

### 4. Data Transfer Objects (DTOs) & Models (`internal/auth/dto.go`, `internal/auth/model.go`)
- **Models**: Database entity structs with GORM tags mapping directly to database tables.
- **DTOs**: Explicit API request and response shapes annotated with Huma doc and validation tags.

---

## Project Structure & Feature Extension Guide

```
Tusk/
├── cmd/
│   ├── api/             # Main API application entrypoint
│   ├── migrate/         # Database migration CLI tool
│   └── tusk/            # Framework CLI tool (e.g. auth sync)
├── config/              # Centralized environment configuration loader
├── database/            # Schema migrations and seeders
├── docs/                # Technical documentation guides
├── internal/
│   ├── app/             # Application lifecycle, routing, & dependency wiring
│   ├── auth/            # Authentication & RBAC identity domain module
│   ├── audit/           # Audit logs domain module (business auditing)
│   ├── notification/    # User notification domain module
│   ├── middleware/      # Global HTTP middlewares (CORS, JWT, Rate Limiter)
│   └── platform/        # Infrastructure client setups (DB, Redis, Telemetry)
└── pkg/
    ├── authz/           # Code-first authorization & RBAC registry
    ├── cache/           # Redis / In-memory caching driver package
    ├── crypto/          # Hashing, encryption & signing utilities
    ├── events/          # In-memory event bus / pub-sub package
    ├── files/           # Local & cloud S3 file storage package
    ├── health/          # Health check & readiness probe package
    ├── logger/          # Structured logging package
    ├── mailer/          # Email transmission driver package
    ├── queue/           # Background job queue package (Asynq/Redis)
    ├── query/           # Unified pagination, filtering & sorting engine
    ├── response/        # Standardized API response envelopes
    ├── scheduler/       # Cron & scheduled job package
    ├── sms/             # SMS dispatch driver package (Twilio/Infobip)
    ├── telemetry/       # OpenTelemetry, Tracing & Metrics driver
    └── validate/        # Input validation helpers
```

---

## 🗺️ Where Do Future Infrastructure & Domain Features Belong?

When expanding Tusk with new features, follow this placement guide to maintain architectural consistency:

### 1. Standalone Reusable Infrastructure (`pkg/`)
Infrastructure drivers that are decoupled from specific business logic belong in `pkg/`:

| Feature / Package | Recommended Location | Purpose |
| :--- | :--- | :--- |
| **Cache & Redis** | `pkg/cache/` | Redis client wrapper, in-memory cache interfaces & TTL store |
| **Queue & Background Jobs** | `pkg/queue/` | Background task queues (e.g. Redis/Asynq, worker pool) |
| **Email Queue** | `pkg/mailer/` | Email sending drivers (SMTP, SES, Mailgun) + queued jobs |
| **SMS Queue** | `pkg/sms/` | SMS dispatch drivers (Twilio, Infobip, AfricasTalking) |
| **File Storage** | `pkg/files/` or `pkg/storage/` | Local filesystem, AWS S3, Cloudflare R2 file uploads |
| **Scheduler** | `pkg/scheduler/` | Cron jobs, timed worker scheduling (`robfig/cron`) |
| **Event Bus** | `pkg/events/` | Event pub/sub dispatcher (domain event bus) |
| **Crypto & Hashing** | `pkg/crypto/` | Encryption, decryption, secure token generation, hashing |
| **Health Checks** | `pkg/health/` | Health check probes (Liveness, Readiness, DB/Redis checks) |
| **Telemetry & Tracing** | `pkg/telemetry/` | OpenTelemetry tracer provider, Jaeger exporter, Prometheus metrics |

---

### 2. Cross-Cutting Middleware (`internal/middleware/`)
HTTP request lifecycle handlers and security controls belong in `internal/middleware/`:

- **Rate Limiter**: `internal/middleware/ratelimit.go` (uses `pkg/cache` or Redis sliding window token bucket).
- **Tracing / OpenTelemetry Middleware**: `internal/middleware/tracing.go` (attaches trace context to incoming HTTP requests).
- **Metrics Middleware**: `internal/middleware/metrics.go` (promhttp / request latency metrics).

---

### 3. Core Server Lifecycle & Platform (`internal/platform/`)
Third-party client initializers and server lifecycle management belong in `internal/platform/`:

- **Redis Client Initialization**: `internal/platform/redis/redis.go`
- **Graceful Shutdown**: `internal/app/app.go` (trapping `SIGINT`/`SIGTERM` to gracefully stop HTTP server, queue workers, and close DB/Redis pools).
- **OpenTelemetry Provider Init**: `internal/platform/telemetry/tracer.go`

---

### 4. Domain Business Modules (`internal/<domain>/`)
Features containing business rules, database tables, and API routes belong in their own `internal/` domain package:

- **Audit Logs**: `internal/audit/` (tracks user actions, IP addresses, resource mutations).
- **Activity Logs**: `internal/activity/` (user activity feed, login history).
- **Notifications**: `internal/notification/` (in-app notifications, push notifications, preference management).
