package authz

import "sync"

// Registry holds all registered permissions defined across modules.
type Registry struct {
	mu          sync.RWMutex
	permissions map[string]Permission
}

// NewRegistry initializes a new thread-safe permission registry.
func NewRegistry() *Registry {
	return &Registry{
		permissions: make(map[string]Permission),
	}
}

// Register explicitly adds permissions to the registry.
// Duplicate registrations overwrite previous definitions to allow explicit updates.
func (r *Registry) Register(perms ...Permission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range perms {
		if p.Name != "" {
			r.permissions[p.Name] = p
		}
	}
}

// All returns a slice of all registered permissions in the registry.
func (r *Registry) All() []Permission {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]Permission, 0, len(r.permissions))
	for _, p := range r.permissions {
		all = append(all, p)
	}
	return all
}

// Find retrieves a permission by its name identifier.
func (r *Registry) Find(name string) (Permission, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.permissions[name]
	return p, ok
}

// Exists checks whether a permission with the given name is registered.
func (r *Registry) Exists(name string) bool {
	_, ok := r.Find(name)
	return ok
}
