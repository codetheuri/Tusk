# Changelog

All notable changes to the **Tusk** framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Code-First RBAC Engine (`pkg/authz`)**: Multi-tenant-ready Role-Based Access Control system with permission constants, global registry, and runtime GORM DB synchronization.
- **Huma v2 Integration**: Strongly-typed HTTP API framework built on Chi with automated OpenAPI 3.0 specification (`/openapi.json`) and interactive documentation UI (`/docs`).
- **Permission Sync CLI (`tusk auth sync`)**: CLI tool with `--prune` option to sync code-declared permissions directly to runtime database tables.
- **Unified Querying Engine (`pkg/query`)**: Standardized query struct for pagination, field sorting, keyword searching, and response metadata (`query.Meta`).
- **Standardized API Envelope (`pkg/response`)**: Uniform JSON envelope structure (`success`, `message`, `data`, `errors`) across all endpoints.
- **Technical Documentation Suite (`docs/`)**: Comprehensive technical guides covering Architecture, Routing, Authorization, Database Migrations, Querying, and API Responses.
- **Contribution Guidelines (`CONTRIBUTING.md`)**: Architectural conventions and contribution rules for extending Tusk.

### Changed
- **Router Tagging & Schema Refinement**: Standardized Huma API operation tags and route descriptions across authentication and authorization endpoints.
- **Updated CI Pipelines**: Refreshed `.github/workflows/go.yml` to support Go 1.24.x and updated action versions.

### Removed
- **Scaffolding Tool (`cmd/genmodule`)**: Removed obsolete CLI code generator in favor of explicit, clean module composition.
- **Deployment Workflow (`deploy.yml`)**: Removed unused VPS SSH deployment GitHub action.
