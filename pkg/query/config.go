package query

// Config defines column whitelists and bounds for safe query execution.
type Config struct {
	// DefaultSort specifies the default sorting directive (e.g., "-created_at").
	DefaultSort string

	// DefaultPerPage specifies the default per_page limit when omitted by the client.
	DefaultPerPage int

	// MaxPerPage enforces a hard upper bound on per_page to prevent memory exhaustion attacks.
	MaxPerPage int

	// AllowedSorts maps URL parameter names to verified database column names.
	// Example: map[string]string{"name": "users.username", "created_at": "users.created_at"}
	AllowedSorts map[string]string

	// AllowedSearches lists database column names evaluated during multi-column search.
	// Example: []string{"users.username", "users.email", "users.phone"}
	AllowedSearches []string

	// AllowedFilters maps URL filter parameter names to verified database column names.
	// Example: map[string]string{"status": "users.is_active", "role": "roles.name"}
	AllowedFilters map[string]string
}

// SanitizePage ensures page is at least 1.
func (c Config) SanitizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// SanitizePerPage enforces DefaultPerPage and MaxPerPage bounds.
func (c Config) SanitizePerPage(perPage int) int {
	defaultLimit := c.DefaultPerPage
	if defaultLimit <= 0 {
		defaultLimit = 20
	}

	maxLimit := c.MaxPerPage
	if maxLimit <= 0 {
		maxLimit = 100
	}

	if perPage <= 0 {
		return defaultLimit
	}
	if perPage > maxLimit {
		return maxLimit
	}
	return perPage
}
