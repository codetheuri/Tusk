# Tusk Roadmap & Known Gaps

**Last reviewed:** 2026-09-07
**Status:** Phase 1 hardening complete. Phase 2 (UUIDv7 keys) complete.

This document tracks known production gaps and planned work. Items marked ✅ are resolved; treat anything else as unhandled.

---

## 1. Production hardening

Ordered by risk.

### High — all resolved

| # | Gap | Location |
|---|---|---|
| H1 ✅ | **Rate limiter is implemented but never installed.** `middleware/ratelimit.go` is complete; `app.New()` never calls it. Every endpoint currently accepts unlimited requests from a single IP — including login, which makes password brute-forcing free. | `internal/app/app.go` |
| H2 ✅ | **A database query runs on every authenticated request.** `HumaAuthenticate` issues `SELECT is_super_user FROM users WHERE id = ?` purely to read one boolean. Move the flag into the JWT so it arrives with the request. Accept that a revoked super-user keeps the flag until the token expires — which is precisely why fine-grained *permissions* must stay in the database rather than the token. | `internal/middleware/jwt.go` |
| H3 ✅ | **Invalid tokens are treated as absent tokens.** An expired or malformed token currently falls through unauthenticated and fails later at the permission guard, so the client receives `403 Forbidden` when the truth is `401 Unauthorized`. Clients cannot distinguish "you lack permission" from "your session expired", so they never know to refresh. | `internal/middleware/jwt.go` |
| H4 ✅ | **Raw-string context keys.** `context.WithValue(ctx, "user_id", …)` is set alongside the typed `ContextKeyUserID`, and `pkg/authz` reads the raw-string one — so the unsafe key is the load-bearing one. Any dependency using the same string silently overwrites it, at runtime, with no compile error. Contradicts `.agents/rules/gostyle.md`. | `internal/middleware/jwt.go`, `pkg/authz/middleware.go` |
| H5 ✅ | **No request body size limit.** Unbounded request bodies are a trivial memory-exhaustion vector. | `internal/app/app.go` |
| H6 ✅ | **Migrations are MySQL-only.** `00001_identity_auth_rbac.mysql.sql` uses MySQL syntax and will fail against PostgreSQL, which is now the supported database. | `database/migrations/` |

### Medium — all resolved

| # | Gap | Location |
|---|---|---|
| M1 ✅ | **Config performs no validation.** No minimum length enforced on `JWT_SECRET`, connection-pool sizes hardcoded rather than environment-driven, dead commented-out assignments, and no `IsProduction()` helper. Configuration errors should fail loudly at startup, not surface as confusing runtime behaviour. | `config/config.go` |
| M2 ✅ | **Server timeouts are too tight for large payloads.** `ReadTimeout: 5s` / `WriteTimeout: 10s`, with no `ReadHeaderTimeout` or `MaxHeaderBytes`. Any endpoint accepting a substantial upload over a slow connection will be cut off mid-request. | `internal/app/app.go` |
| M3 ✅ | **API documentation is always public.** `/docs` and `/openapi.json` are served unconditionally, including in production. See §2. | `internal/app/app.go` |
| M4 ✅ | **Two parallel authentication middlewares.** `Authenticate` (net/http) and `HumaAuthenticate` (Huma) duplicate JWT parsing and will drift apart. Consolidate on one implementation. | `internal/middleware/jwt.go` |
| M5 ✅ | **Logging is unstructured.** A custom console logger with no JSON output and no request or trace correlation. Migrate to `log/slog` with a JSON handler in production and `request_id` on every line. | `pkg/logger/` |
| M6 ✅ | **No database integration tests.** Config, JWT parsing, and logging now have unit coverage; `pkg/*` has unit coverage; the auth module and the database layer have none. Tests should run against a real throwaway PostgreSQL — not SQLite, which diverges on exactly the behaviour worth testing. | repo-wide |

### Low — resolved

| # | Gap | Location |
|---|---|---|
| L1 ✅ | Health endpoints hand-format JSON with `fmt.Sprintf` rather than encoding it, and there is no `/live` distinct from `/ready`. | `internal/app/app.go` |

**Already correct, no action needed:** `.env` is gitignored and no secrets are tracked in git.

---

## 2. Gating API documentation in production

Endpoints stay live; only the documentation surface disappears.

Huma supports this natively — setting `Config.DocsPath` and `Config.OpenAPIPath` to `""` removes those routes entirely rather than merely hiding them.

Planned design: a `DOCS_ENABLED` environment variable defaulting to **off** when `APP_MODE=production`. An optional later refinement is to serve documentation behind a super-admin session, so it remains reachable in production for those authorized to see it.

---

## 3. Planned capabilities

### 3.1 UUIDv7 primary keys ✅ *(shipped 2026-09-08)*

Move framework tables from `uint` auto-increment to UUIDv7 (`uuid.NewV7()`, already available via `google/uuid` v1.6.0).

**Rationale.** Auto-increment keys cannot be generated outside the database, which rules out offline-capable clients, dataset merges, and multi-writer topologies. The cost of choosing UUID where `BIGINT` would have sufficed is 8 extra bytes per key; the cost of choosing `BIGINT` where UUID was needed is rewriting every table and foreign key in the system. UUIDv7 specifically — rather than v4 — because its leading millisecond timestamp keeps new keys sorting to the end of the B-tree, preserving insert locality that random v4 keys destroy.

**Known tradeoff.** UUIDv7 embeds a creation timestamp, so an identifier reveals when its record was created and any two identifiers reveal their relative ordering. For ordinary business records this is unremarkable. Applications where creation-time correlation is sensitive should expose a separate random public identifier and keep v7 internal.

An application with a measured reason to use `BIGINT` for its own high-volume tables can simply declare them that way — GORM is indifferent, and no generic `Repository[T, ID]` machinery is needed.

### 3.2 Optional multi-tenancy (`pkg/tenant`) ✅ *(shipped 2026-09-08)*

Shared schema, tenant column, automatic query scoping, with PostgreSQL row-level security as a safety net beneath the application's own filtering.

**Tenancy must be opt-in per model.** Most applications are single-tenant, and a framework that imposes a tenant column on every table is unusable for them. A model opts in by implementing a single method:

```go
func (Customer) TenantColumn() string { return "business_id" }
```

The query-scoping callback skips every model that does not implement it. Returning the *column name* rather than assuming `tenant_id` lets each application use its own domain vocabulary. Row-level security policies are written only for opted-in tables, and the tenant middleware stays inert until an application registers a resolver.

Framework tables: `users` becomes tenant-scoped; `roles` and `permissions` stay global, since role names generally mean the same thing across tenants. Per-model opt-in means `roles` can join later without disturbing anything else.

**As shipped.** Three layers, each independently sufficient:

| Layer | Mechanism | Behaviour when it is the only one left |
|---|---|---|
| Context | `tenant.Middleware` + a `Resolver` reading a verified claim | Handlers cannot read a tenant from anywhere a caller controls |
| Query scoping | GORM callbacks on create, query, row, update, delete | A tenanted model queried with no tenant errors rather than returning everything |
| Row-level security | `PolicyStatements` DDL + `tenant.Transaction` setting `app.tenant_id` | The database refuses foreign rows even when the application asks for them |

Callbacks are registered by `database.Connect` for every application, not opted into per project — a model added later cannot go unscoped because someone forgot a line of wiring. `tenant.Unscoped(db)` suspends layer 2 deliberately and greppably.

Two failure modes worth knowing about, both covered by tests:

- **An `OR` in a caller's own conditions can swallow the tenant predicate.** `Where(a).Or(b)` with a naively appended `AND` parses as `a OR (b AND tenant)`, so everything matching `a` leaks. The caller's expressions are grouped before the predicate is added.
- **Row-level security does not apply to superusers or roles holding `BYPASSRLS`**, silently. Policies install, appear in `\d`, and are never consulted. `tenant.VerifyEnforcement(db)` reports this; call it at startup in any deployment relying on the RLS layer. Development databases very often run as a superuser, which is exactly when someone concludes the layer works.

### 3.3 Server-rendered pages (`pkg/view`, `pkg/session`) ✅ *(shipped 2026-09-08)*

Tusk is API-first, but admin panels are a recurring need and previously had no supported answer.

- `pkg/view` — `html/template` with layout inheritance, hot reload in development, caching in production.
- `pkg/session` — database-backed cookie sessions, HTTP-only + Secure + SameSite, rotated on privilege change.

This lets a Huma API and an HTML admin panel coexist in one binary without either becoming an afterthought.

**As shipped.** Both packages exist to make three recurring mistakes impossible rather than merely discouraged.

`pkg/view` renders into a buffer and writes nothing until the whole page succeeds. `html/template` streams output as it executes, so rendering straight to the `ResponseWriter` means an error halfway down a page has already sent `200` and half the document — and the handler can no longer choose a status. It also parses every page at construction, so a malformed template stops the process starting instead of surfacing as a 500 to whoever visits that page first. Each page gets its own template set, so two pages may define `"content"` without colliding.

`pkg/session` stores only the SHA-256 digest of a 256-bit token, never the token, so a leaked backup does not hand over live sessions. SHA-256 rather than bcrypt is deliberate and the opposite reasoning to passwords: the token is unguessable regardless of hash speed, and it is verified on every request, so a slow hash would be a denial-of-service surface. `Middleware` loads a session; `Require` refuses without one. Keeping those separate is what prevents the classic redirect loop, where a login page that redirects on the mere *presence* of a cookie bounces a visitor holding a stale one between `/login` and `/dashboard` forever.

`Options.Kind` separates audiences sharing one table, so an operator-console cookie can never resolve against an ordinary user session. `Rotate` inherits the absolute expiry rather than restarting it — rotation is a security measure, not a way to hold a session open indefinitely.

---

## 4. Changelog

### 2026-09-08 — pkg/query: LIKE was case-sensitive on Postgres

Found while building Salio's customer search. `Apply`'s multi-column search used `LIKE`, which is case-sensitive on PostgreSQL — the only database Tusk targets (D6). A search for "wanjiku" silently missed "Wanjiku". Fixed to `ILIKE`. `internal/auth`'s user search (`AllowedSearches: []string{"users.username", "users.email", "users.phone"}`) had the identical defect and is fixed by the same change.

`pkg/query` had no test coverage before this — added `pkg/query/query_test.go` (5 tests, real PostgreSQL): case-insensitive search, an unlisted-column search is ignored, pagination metadata across pages, `MaxPerPage` enforcement, and sort-column whitelisting.

### 2026-09-08 — Layout: what a downstream service can import

Tusk is meant to be built on, and Go's `internal/` rule makes a package unreachable from any other module. The pieces a service actually needs moved out:

| Was | Now |
|---|---|
| `internal/middleware` | `pkg/middleware` |
| `internal/app` | `pkg/app` |
| `internal/platform/database/gorm.go` | `database/connect.go` (beside the migration runner; `NewGoRMDB` → `Connect`) |
| `internal/auth` | unchanged — still internal |

`pkg/app.New` now takes an `Options` struct and **registers no routes**. It returns an `App` exposing `API()`, `Router()`, `DB()` and `Config()`; callers register their own modules. Tusk's own auth module is registered from `cmd/api/main.go` like any other consumer — it is the framework's first user, not a privileged part of it.

`internal/auth` stays internal deliberately. A service wanting different authentication — phone-only login, no email — writes its own module against `pkg/authz` rather than bending Tusk's.

Paths in the §2 defect table are left as they were when those defects were fixed; git holds the history.


### 2026-09-08 — Phase 4: server-rendered pages

- New `pkg/view` (9 tests) and `pkg/session` (13 tests, real PostgreSQL), plus migration `00002_sessions`.
- New `database.ResetMigrations` / `make migrate-reset` / `migrate reset`, unwinding every migration rather than one step.
- **The migration round-trip test was asserting less than it claimed.** It called `RollbackMigration`, which reverts a single step; with only one migration in the tree that happened to drop everything, so the test appeared to verify the whole rollback path. Adding a second migration exposed it. It now resets fully.
- **`lib/pq` is gone.** `config`, `cmd/migrate` and the tests all registered it while `gorm.io/driver/postgres` used pgx, so the schema was being built by one driver and used by another. Everything is pgx now, and lib/pq is no longer a dependency.

### 2026-09-08 — Phase 3: optional multi-tenancy

- New `pkg/tenant`: `Tenanted` opt-in interface, context propagation, GORM scoping callbacks, `Unscoped` escape hatch, RLS policy DDL, `Transaction`, `VerifyEnforcement`, and HTTP + Huma middleware.
- `Claims` gains an omitempty `tenant_id`; `middleware.TenantResolver` feeds it to the middleware.
- `database.Connect` and `internal/testdb` both register the callbacks, so tests exercise the production configuration.
- **`internal/testdb` now connects through pgx rather than lib/pq.** `gorm.io/driver/postgres` opens production connections with `stdlib.OpenDB`, so the harness had been testing a driver the application never uses — and they differ: GORM's migrator introspection reuses a placeholder, which pgx accepts and lib/pq rejects outright.
- **Database tests now take a PostgreSQL advisory lock.** `go test ./...` runs packages concurrently and every database test TRUNCATEs the whole schema; with a second database-using package this produced failures that looked like application bugs and reproduced only sometimes. The lock lives in the harness rather than in a `-p 1` Makefile flag, because a plain `go test ./...` has to be correct too.

### 2026-09-07 — Phase 1 hardening

Resolved H1–H6, M1–M5 and L1. Notable behaviour changes for anyone upgrading:

- **`JWT_SECRET` must now be at least 32 characters.** Startup fails otherwise. This is a hard break for installations running a short key — rotate it before deploying.
- **`/docs` and `/openapi.json` no longer appear when `APP_MODE=production`.** Set `DOCS_ENABLED=true` to restore them. API endpoints are unchanged.
- **An invalid or expired token now returns 401, not 403.** Clients that treated 403 as "session expired" should be checked.
- **`is_super_user` is read from the JWT** rather than the database. Existing tokens lack the claim and will be treated as non-super until reissued — it fails closed, so no privilege is granted by accident.
- **Migrations moved to `database/migrations/<dialect>/`.** PostgreSQL is primary and receives all new work; the MySQL migration is preserved but frozen.
- **`DB_SSLMODE` is new**, defaulting to `require` in production. Previously the DSN hardcoded `sslmode=disable`, so production database traffic was unencrypted with no way to change it.
- Rate limiting and a request body cap are now actually installed; both are configurable.
- `swaggo/swag` and its dependency tree were dropped — leftovers from before Huma.

**Integration testing (M6).** Tests now run against a real PostgreSQL via `internal/testdb`:

- `make test` — everything; integration tests **skip** if no database is reachable.
- `make test-integration` — the same, but a missing database is a **failure**.
- `make test-db-setup` — creates the local `tusk_test` database once.
- Override the target with `TEST_DATABASE_URL`; default is `127.0.0.1:5434/tusk_test`.

CI sets `REQUIRE_DB_TESTS=1`, so a green build can never mean the integration suite was silently skipped. The workflow now also reads its Go version from `go.mod` — it was pinned to 1.24.x while the module required 1.25.7 — and runs `go vet`.

Covered: registration, login by username or email, account lockout and reset, refresh-token rotation and replay rejection, logout revocation, unique-constraint enforcement, RBAC evaluation across the three-table join, super-user bypass, permission-sync idempotency, and a migration up/down/up round trip in its own throwaway database.

### 2026-09-08 — UUIDv7 primary keys

Framework tables moved from `uint` auto-increment to UUIDv7. Identifiers are generated in Go by the new `pkg/id`, never by the database.

**Breaking changes:**

- **PostgreSQL is now the only supported driver.** MySQL and SQLite were dropped, and `LoadConfig` refuses anything else at startup rather than failing at the first query. Neither has a native UUID type, so their schema could no longer describe the same rows as the models. The MySQL migration was deleted; the SQL remains in git history.
- **Every primary and foreign key is now `UUID`.** Migration `00001` was rewritten rather than layered with a conversion — there was no production data to preserve. An existing database must be recreated.
- `Claims.UserID`, `authz.Subject.UserID`, and every repository, service and DTO signature now take `uuid.UUID`. Tokens issued before this change carry a numeric `user_id` and will fail to parse, so all sessions are invalidated.

**No server-side default.** The migration deliberately omits `DEFAULT`. PostgreSQL gained `uuidv7()` only in version 18, and `gen_random_uuid()` would yield v4 — losing the index locality that motivated v7. More fundamentally, an offline client must be able to create a row and know its ID before the server has seen it, which no server-side default can serve.

**Client-supplied identifiers are preserved.** Each model's `BeforeCreate` assigns an ID only when one is absent. A device that generated its own ID while disconnected keeps it, because other records it created may already reference it. Both directions are covered by tests.
