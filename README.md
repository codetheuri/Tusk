# 🐘 Tusk - The Sharpest Go Starter for Building APIs

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.20-blue.svg)](https://golang.org/doc/go1.20)
[![GitHub release](https://img.shields.io/github/v/release/codetheuri/Tusk?include_prereleases)](https://github.com/codetheuri/Tusk/releases)
<!-- [![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE) -->
[![Build Status](https://img.shields.io/github/actions/workflow/status/codetheuri/Tusk/go.yml?branch=main)](https://github.com/codetheuri/Tusk/actions)
<!-- [![Codecov](https://img.shields.io/codecov/c/github/codetheuri/Tusk)](https://codecov.io/gh/codetheuri/Tusk) -->
[![GitHub stars](https://img.shields.io/github/stars/codetheuri/Tusk?style=social)](https://github.com/codetheuri/Tusk/stargazers)
<!-- [![Discord](https://img.shields.io/discord/your-discord-channel-id?label=Discord)](https://discord.gg/your-invite-link) -->

Tusk is a robust and opinionated starter project designed to jumpstart your API development in Go. It provides a solid foundation with essential features and a clear architectural structure, allowing you to focus on your core business logic rather than boilerplate setup.

---
##  Features


* **Go-Chi Router:** Utilizes `go-chi/chi` for a lightweight, idiomatic, and composable HTTP router, simplifying API routing and middleware integration.
* **GORM ORM:** Leverages GORM for powerful and developer-friendly Object-Relational Mapping, providing seamless database interaction and auto-migrations.
* **Clean Architecture:** Organizes code into distinct layers (handlers, services, repositories, models) for maintainability, testability, and scalability.
* **Authentication:** Includes a complete authentication flow (registration, login, password management) with **JWT-based** token handling and **Bcrypt** for secure password hashing.
* **Database Migrations:** Built-in CLI for managing database schema changes (`up`, `down`, `create`, `fresh`).
* **Database Seeding:** CLI support for populating your database with initial data.
* **Module Generator:** A handy CLI tool to scaffold new API modules quickly.
* **Structured Logging:** Integrates a powerful logging solution (`pkg/logger`) for effective debugging and monitoring.
* **Request Validation:** Leverages a `pkg/validators` package for robust input validation.
* **Error Handling:** Implements a custom application error (`pkg/errors`) system for consistent and informative error responses.
* **Web Utilities:** Provides common web utilities (`pkg/web`) for consistent JSON responses and error handling.
* **Pagination:** Built-in support for paginated responses, making it easy to handle large datasets.


---
##  Getting Started

These instructions will get your Tusk project up and running on your local machine.

### Prerequisites

* Go (version 1.20 or newer recommended)
* A database (e.g., PostgreSQL, MySQL) compatible with GORM.

### Installation & Setup

1.  **Clone the Repository:**
    ```bash
    git clone [https://github.com/codetheuri/Tusk.git](https://github.com/codetheuri/Tusk.git)
    cd Tusk
    ```

2.  **Configure Environment Variables:**
    Create your `.env` file from the example template:
    ```bash
    cp .env.example .env
    ```
    Now, open `.env` and populate it with your database connection string, JWT secret, and any other necessary configuration values.

3.  **Run the Server:**
    ```bash
    go run cmd/main.go
    ```
    Your API server should now be running, typically on `http://localhost:8080` (check your configuration in `.env`).



##  CLI Tools

Tusk comes with helpful command-line tools to streamline development tasks.

### Database Migrations (`cmd/migrate`)

This CLI tool helps you manage your database schema. Ensure your `.env` file is correctly configured before running migration commands.

**Usage:**

```bash
go run ./cmd/migrate <command> [arguments]
```
#### Create new migration
Generates a new migration file with a timestamp and the given name. The file will be created in `database/migrations/`.
Inside the generated file, you'll find Up and Down methods where you define your schema changes (using GORM's AutoMigrate, Migrator(), or raw SQL).
```bash
go run ./cmd/migrate create -name add_users_table
# Example Output: Migration file created: database/migrations/YYYYMMDDHHMMSS_create_users_table.go
```

`up:`  Applies all pending migrations to the database.
```bash 
go run ./cmd/migrate up

```
`down:` Rolls back the last applied migration. You can specify how many migrations to roll back using the -steps flag.
```bash
go run ./cmd/migrate down             # Rolls back 1 migration
go run ./cmd/migrate down -steps 3   # Rolls back the last 3 migrations

```

`fresh:` Reset database (DEV ONLY)
```bash
go run ./cmd/migrate fresh
```

`seed`: Runs all registered database seeders, populating your database with initial data.
```bash
go run ./cmd/migrate seed
```
`seed -name <NAME>:` Runs a specific database seeder by its name (e.g., 01UsersTableSeeder).
```bash
go run ./cmd/migrate seed -name 01UsersTableSeeder
```
`help: `Displays the usage information for the migration tool.
```bash
go run ./cmd/migrate help
```

### Module Generator (`cmd/genmodule`)
This CLI tool helps you quickly scaffold new API modules (e.g., products, orders) by creating the necessary directory structure and boilerplate Go files for handlers, services, and repositories.
**Usage**
```bash
go run ./cmd/genmodule <module_name>
```
Example
```bash
go run ./cmd/genmodule tasks
```
This will create the following structure:
```bash
internal/app/tasks/
├── handlers/
│   └── handler.go
├── models/
├── repositories/
│   └── repository.go
├── services/
│   └── service.go
└── module.go
```
**Important Notes After Generation:**

Add to `app.go`: You must register your new module in `internal/bootstrap/app.go` by adding it to the appModules slice.

Create Models: Crucially, create your GORM model file(s) (e.g., task.go) inside `internal/app/modules/<module_name>/models/`.

Adjust Boilerplate: The generated files contain basic boilerplate. You'll need to uncomment and adjust imports, define methods, and implement your specific business logic.

Run to clean up your Go module dependencies.
 ```bash
  go mod tidy 
  ```
  
---
##  Why Tusk Exists

I found myself repeatedly building similar foundational components for various side projects. Tusk emerged from the need to consolidate these common, yet often time-consuming, pieces into a coherent and structured starting point. While it reflects my personal preferences and might not fit every Go developer's workflow, it serves as a robust template for quickly launching new API services. It's built for sanity, efficiency, and my future self.

---


##  TODO & Future Improvements

 Here are some planned enhancements:

* **Middleware Enhancements:** 
* **rbac**
* **Robust Testing:** Expand test coverage, focusing on unit and integration tests for handlers, services, and repositories.
* **Routing Framework Evaluation:** Explore options to switch to `gin`,`Fiber` or ensure optimal usage of `chi` for routing performance and ergonomics.
* **API Documentation:** Integrate Swagger/OpenAPI for automated API documentation generation.

---
**Thank you for exploring Tusk!** We hope it provides a solid and sharp foundation for your next Go API project.