# Advanced Querying, Filtering & Pagination

Tusk provides a unified, reusable querying package (`pkg/query`) to standardize pagination, sorting, search, and dynamic filtering across all API endpoints.

---

## The Query Contract (`pkg/query`)

The `query.Query` struct captures common URL parameters directly from HTTP requests:

```go
type Query struct {
    Page     int               // Page number (default: 1)
    PerPage  int               // Items per page (default: 20, max: 100)
    Search   string            // General keyword search parameter
    Sort     string            // Field sorting e.g. "-created_at" or "username"
    Filters  map[string]string // Key-value field equality or operator filters
}
```

---

## 1. Using Query Parameters in DTOs

Huma structs capture query parameters cleanly using struct tags:

```go
type ListUsersInput struct {
    Page     int    `query:"page" doc:"Page number (default 1)"`
    PerPage  int    `query:"per_page" doc:"Items per page (default 20)"`
    Search   string `query:"search" doc:"Search across username, email, and phone"`
    Sort     string `query:"sort" doc:"Sort field e.g. -created_at or username"`
    IsActive string `query:"is_active" doc:"Filter by active status (true/false)"`
}
```

---

## 2. Applying Queries in Repositories

Repositories apply `query.Query` to GORM database queries seamlessly using helper methods:

```go
func (r *Repository) ListUsers(ctx context.Context, q query.Query) ([]User, query.Meta, error) {
    var users []User
    var totalRecords int64

    db := r.db.WithContext(ctx).Model(&User{})

    // Apply search across multiple columns
    if q.Search != "" {
        searchTerm := "%" + q.Search + "%"
        db = db.Where("username LIKE ? OR email LIKE ? OR phone LIKE ?", searchTerm, searchTerm, searchTerm)
    }

    // Apply specific field filters
    if active, ok := q.Filters["is_active"]; ok && active != "" {
        db = db.Where("is_active = ?", active == "true")
    }

    // Count total matching records
    if err := db.Count(&totalRecords).Error; err != nil {
        return nil, query.Meta{}, err
    }

    // Apply sorting and pagination limits
    offset := (q.Page - 1) * q.PerPage
    orderClause := q.SortClause("created_at DESC") // Default sort

    if err := db.Order(orderClause).Offset(offset).Limit(q.PerPage).Find(&users).Error; err != nil {
        return nil, query.Meta{}, err
    }

    // Construct pagination metadata
    meta := query.NewMeta(q.Page, q.PerPage, totalRecords)

    return users, meta, nil
}
```

---

## 3. Pagination Metadata Envelope (`query.Meta`)

All paginated responses return metadata informing clients of pagination status:

```json
{
  "success": true,
  "message": "Users retrieved successfully",
  "data": {
    "users": [...],
    "meta": {
      "page": 1,
      "per_page": 20,
      "total_records": 42,
      "total_pages": 3,
      "has_next": true,
      "has_prev": false
    }
  }
}
```
