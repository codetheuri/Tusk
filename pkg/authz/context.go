package authz

import "context"

// subjectKey is the context key under which the authenticated Subject is stored.
//
// It is an unexported struct type rather than a string on purpose. Context keys
// are compared by type as well as value, so an unexported type cannot be forged
// or accidentally collided with from outside this package — even by code that
// uses the identical name. A plain string key such as "user_id" offers no such
// protection: any dependency writing the same string silently overwrites the
// value, at runtime, with no compile error.
type subjectKey struct{}

// WithSubject returns a copy of ctx carrying the authenticated subject.
//
// Authentication middleware calls this exactly once per request. Everything
// downstream — guards, handlers, services — reads it back through
// SubjectFromContext rather than reaching for individual claim values.
func WithSubject(ctx context.Context, sub Subject) context.Context {
	return context.WithValue(ctx, subjectKey{}, sub)
}

// SubjectFromContext returns the authenticated subject and whether one is present.
//
// The boolean must be checked. An anonymous request yields the zero Subject,
// which has UserID 0 and IsSuperUser false — indistinguishable from a real
// subject if the caller ignores the flag. Treat false as "unauthenticated",
// never as "a user with no permissions".
func SubjectFromContext(ctx context.Context) (Subject, bool) {
	sub, ok := ctx.Value(subjectKey{}).(Subject)
	return sub, ok
}
