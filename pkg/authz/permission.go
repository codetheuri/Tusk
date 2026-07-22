package authz

// Permission represents a developer-defined, immutable permission declared in code.
type Permission struct {
	// Name is the unique hierarchical identifier (e.g., "users.create", "finance.invoice.approve").
	Name string
	// Description provides context for administrators configuring roles.
	Description string
}

// Global default registry instance for convenient application-wide access.
var defaultRegistry = NewRegistry()

// Register adds permissions to the global default registry.
func Register(perms ...Permission) {
	defaultRegistry.Register(perms...)
}

// DefaultRegistry returns the global default permission registry.
func DefaultRegistry() *Registry {
	return defaultRegistry
}
