# Routing & OpenAPI Specification (Huma v2)

Tusk uses **[Huma v2](https://huma.rocks/)** built on top of Chi for strongly-typed, declarative API routing and automatic OpenAPI 3.0 specification generation.

---

## Why Huma v2?

Huma v2 eliminates hand-written Swagger/OpenAPI annotations by inferring schemas directly from Go structs and function signatures:

- **Type Safety**: Request bodies, path parameters, and query parameters are bound directly into typed Go structs.
- **Automatic OpenAPI Generation**: OpenAPI 3.0 specs (`/openapi.json`) are updated automatically whenever routes or DTOs change.
- **Built-in Documentation UI**: Interactive documentation is served automatically at `/docs`.

---

## Defining API Endpoints

Routes are registered using `huma.Register` inside module routers (e.g., `internal/auth/router.go`).

### Example Route Definition

```go
huma.Register(api, guard.Protected(huma.Operation{
    OperationID: "auth-list-roles",
    Method:      http.MethodGet,
    Path:        "/api/v1/auth/roles",
    Summary:     "List all security roles",
    Description: "Retrieves all security roles configured in the system.",
    Tags:        []string{"Authorization"},
}, PermRolesRead), handler.ListRoles)
```

---

## Structuring Request & Response DTOs

Huma utilizes struct tags for routing metadata, validation, and documentation.

### Request DTO Example

```go
type CreateRoleInput struct {
    Body struct {
        Name        string `json:"name" minLength:"2" doc:"Name of the role (e.g., admin, editor)"`
        Description string `json:"description" doc:"Optional description of role capabilities"`
    }
}
```

### Parameter Tags Supported by Huma:

- `path:"id"`: Maps route path parameters (e.g. `/roles/{id}`).
- `query:"search"`: Maps URL query parameters (e.g. `?search=admin`).
- `header:"Authorization"`: Maps HTTP request headers.
- `doc:"..."`: Provides human-readable descriptions for OpenAPI documentation.
- `minLength:"2"`, `format:"email"`: Enforces automatic validation constraints.

### Response DTO Example

Tusk uses a generic envelope `response.Data[T]` for consistent API responses:

```go
type RoleData struct {
    Role *Role `json:"role"`
}

type RoleOutput struct {
    Body response.Data[RoleData]
}
```

---

## Handler Implementation Pattern

Handlers receive `context.Context` and the strongly-typed Input struct, then return the Output struct and an `error`:

```go
func (h *RoleHandler) ListRoles(ctx context.Context, input *ListRolesInput) (*RolesOutput, error) {
    roles, err := h.service.ListRoles(ctx)
    if err != nil {
        return nil, huma.Error500InternalServerError("Failed to fetch roles")
    }

    return &RolesOutput{
        Body: response.OK(RolesData{Roles: roles}, "Roles retrieved successfully"),
    }, nil
}
```

---

## Interactive Documentation & Specs

When the application runs:
- **Interactive UI**: Navigate to `http://localhost:8080/docs`
- **OpenAPI Schema**: `http://localhost:8080/openapi.json`
