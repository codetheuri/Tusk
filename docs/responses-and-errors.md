# Standardized API Responses & Error Handling

Tusk enforces consistent, predictable API response payloads using the `pkg/response` package.

---

## Unified JSON Envelope Structure

Every API response emitted by Tusk adheres to a uniform JSON envelope format:

```json
{
  "success": true,
  "message": "Resource processed successfully",
  "data": { ... },
  "errors": null
}
```

### Response Fields

- **`success`** (`bool`): Indicates whether the request succeeded or failed.
- **`message`** (`string`): A user-friendly message describing the operation outcome.
- **`data`** (`any`): The primary payload object or array returned on success.
- **`errors`** (`[]string` | `null`): An optional list of specific validation errors or failure details.

---

## 1. Constructing Success Responses

The `pkg/response` package provides helper functions to wrap payloads cleanly:

### Standard Success Response (`response.OK`)

```go
return &ProfileOutput{
    Body: response.OK(ProfileData{
        User:        user,
        Permissions: permissions,
    }, "Profile retrieved successfully"),
}, nil
```

### Created Success Response (`response.Created`)

```go
return &AuthOutput{
    Body: response.Created(AuthResultData{
        User:        user,
        AccessToken: token,
    }, "User account registered successfully"),
}, nil
```

---

## 2. Handling Errors & HTTP Status Codes

Services return domain errors, and Handlers translate them into appropriate HTTP status responses using Huma error utilities (`huma.Error400BadRequest`, `huma.Error401Unauthorized`, `huma.Error403Forbidden`, `huma.Error404NotFound`, `huma.Error500InternalServerError`).

### Error Mapping Best Practices

1. **Never Expose Internal Database Errors**: Internal database or system errors should be logged via `logger` and returned as generic 500 status messages to clients.
2. **Return Meaningful Status Codes**: Use exact status codes matching the error condition:
   - `400 Bad Request`: Input validation failed or invalid payload parameters.
   - `401 Unauthorized`: Missing or expired JWT token.
   - `403 Forbidden`: Authenticated user lacks required permission.
   - `404 Not Found`: Requested resource does not exist.
   - `409 Conflict`: Resource already exists (e.g. duplicate email/username).
   - `422 Unprocessable Entity`: Validation failure on specific payload fields.

### Example Handler Error Response

```go
user, err := h.service.GetRoleByID(ctx, input.ID)
if err != nil {
    return nil, huma.Error404NotFound("Role not found")
}
```

When returned, Huma converts this error into a standardized JSON response:

```json
{
  "$schema": "http://localhost:8080/schemas/ErrorModel.json",
  "title": "Not Found",
  "status": 404,
  "detail": "Role not found"
}
```
