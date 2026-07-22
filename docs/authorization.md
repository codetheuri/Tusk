# Code-First RBAC Authorization

Tusk features a **code-first, registry-driven Role-Based Access Control (RBAC)** package (`pkg/authz`).

---

## Architectural Concept

1. **Code-Declared Permissions**: Permissions are defined as immutable code constants inside domain modules.
2. **Global Permission Registry**: Modules register permissions during package initialization (`init()`).
3. **Database Synchronization**: A CLI tool (`tusk auth sync`) syncs code-declared permissions into the database `permissions` table cleanly and idempotently.
4. **Role & Permission Management**: Administrators assign permissions to roles, and roles to users, via API endpoints.
5. **Route Guard Middleware**: HTTP routes specify required permission strings to enforce authorization automatically.

---

## 1. Declaring & Registering Permissions

Permissions are defined in domain packages (e.g. `internal/auth/permissions.go`):

```go
package auth

import "github.com/codetheuri/tusk/pkg/authz"

const (
    PermUsersRead   = "users.read"
    PermUsersCreate = "users.create"
    PermRolesRead   = "roles.read"
    PermRolesCreate = "roles.create"
)

var Permissions = []authz.Permission{
    {
        Name:        PermUsersRead,
        Description: "Allows viewing user accounts and profiles",
    },
    {
        Name:        PermUsersCreate,
        Description: "Allows registering and creating new user accounts",
    },
}

func init() {
    authz.Register(Permissions...)
}
```

---

## 2. Synchronizing Permissions to Database

Permissions registered in code are synchronized to runtime database tables (`permissions`, `roles`, `role_permissions`, `user_roles`) using the Tusk CLI tool.

### Syncing Registered Permissions

```bash
make auth-sync
# OR
go run ./cmd/tusk/main.go auth sync
```

### Syncing and Pruning Obsolete Database Permissions

To automatically delete permissions from the database that no longer exist in code:

```bash
make auth-sync-prune
# OR
go run ./cmd/tusk/main.go auth sync --prune
```

---

## 3. Protecting Routes with Permission Guards

Routes specify required permissions using the `guard.Protected(...)` helper:

```go
huma.Register(api, guard.Protected(huma.Operation{
    OperationID: "auth-list-users",
    Method:      http.MethodGet,
    Path:        "/api/v1/auth/users",
    Summary:     "List users",
    Tags:        []string{"Authentication"},
}, PermUsersRead), handler.ListUsers)
```

If the authenticated user lacks the `users.read` permission (and is not a superuser), the middleware returns `403 Forbidden`.

---

## 4. Superuser Bypass

Users with `IsSuperUser = true` automatically bypass all RBAC permission checks, ensuring system administrators always maintain full access.

---

## 5. Authorization API Endpoints

Tusk provides endpoints to manage security roles and permission assignments:

- `GET /api/v1/auth/permissions`: List all registered system permissions.
- `GET /api/v1/auth/roles`: List all security roles with their attached permissions.
- `POST /api/v1/auth/roles`: Create a new security role.
- `POST /api/v1/auth/roles/{id}/permissions`: Attach a permission to a role.
- `DELETE /api/v1/auth/roles/{id}/permissions/{permission_name}`: Detach a permission from a role.
- `POST /api/v1/auth/users/{user_id}/roles`: Assign a role to a user.
- `DELETE /api/v1/auth/users/{user_id}/roles/{role_id}`: Revoke a role from a user.
