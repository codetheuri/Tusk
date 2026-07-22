# Database & Migrations

Tusk uses **[GORM](https://gorm.io/)** for ORM operations and database connectivity, along with a custom migration CLI tool (`cmd/migrate`) to handle database schema changes and seed data cleanly.

---

## Database Connection Management

Database connections are initialized in `internal/platform/database/gorm.go`. Connection settings (driver, host, port, pool sizes) are loaded directly from `.env`.

---

## Migration CLI (`cmd/migrate`)

The migration CLI allows developers to create, apply, roll back, and reset database schemas safely.

### Migration Commands

#### 1. Apply Pending Migrations

Applies all unapplied database schema migrations:

```bash
make migrate-up
# OR
go run ./cmd/migrate/main.go up
```

#### 2. Roll Back Migrations

Rolls back the last applied migration step:

```bash
make migrate-down
# OR
go run ./cmd/migrate/main.go down
```

To roll back multiple steps:

```bash
go run ./cmd/migrate/main.go down -steps 3
```

#### 3. Create a New Migration File

Generates a timestamped migration file in `database/migrations/`:

```bash
go run ./cmd/migrate/main.go create -name add_posts_table
```

#### 4. Fresh Database Reset (Development Only)

Wipes the database schema clean and re-applies all migrations from scratch:

```bash
go run ./cmd/migrate/main.go fresh
```

#### 5. Database Seeding

Populates the database with initial seed data:

```bash
go run ./cmd/migrate/main.go seed
```

To run a specific seeder by name:

```bash
go run ./cmd/migrate/main.go seed -name 01UsersTableSeeder
```

---

## Creating Migration Files

Migration files are written in standard Go using GORM's `AutoMigrate` or `Migrator()` interface.

### Example Migration Structure (`database/migrations/20260722_create_posts.go`)

```package migrations

import (
    "gorm.io/gorm"
)

type Migration20260722CreatePosts struct{}

func (m *Migration20260722CreatePosts) Up(db *gorm.DB) error {
    type Post struct {
        ID        uint   `gorm:"primaryKey"`
        Title     string `gorm:"not null"`
        Content   string `gorm:"type:text"`
        UserID    uint   `gorm:"not null"`
    }
    return db.AutoMigrate(&Post{})
}

func (m *Migration20260722CreatePosts) Down(db *gorm.DB) error {
    return db.Migrator().DropTable("posts")
}
```
