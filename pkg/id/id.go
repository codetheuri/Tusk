// Package id generates Tusk's primary keys.
//
// It exists to state one decision in one place: Tusk identifiers are UUIDv7.
// Without it, that choice would be re-made implicitly in every model file, and
// changing it would mean finding every call site.
//
// # Why UUID rather than an auto-incrementing integer
//
// An auto-increment key can only be issued by the database, so it cannot be
// created by an offline client, and two databases cannot be merged without
// renumbering. The cost of choosing UUID where an integer would have sufficed is
// eight extra bytes per key; the cost of choosing an integer where UUID was
// needed is rewriting every table and foreign key in the system. The mistakes
// are not symmetric, so Tusk takes the cheap one.
//
// # Why v7 rather than v4
//
// A primary key lives in a sorted B-tree. Random v4 keys land at arbitrary
// positions, so each insert dirties a different page and the index fragments.
// v7 places a millisecond timestamp in its leading 48 bits, so new keys sort to
// the end of the tree — the write pattern of an auto-increment key, with the
// independence of a UUID.
//
// # What v7 gives away
//
// The timestamp is readable by anyone holding the identifier: an ID reveals when
// its row was created, and any two IDs reveal their relative order. For ordinary
// business records this is unremarkable, since creation time is usually visible
// anyway. Where that correlation is sensitive, expose a separate random
// identifier and keep the v7 key internal.
//
// Note that this is not an enumeration risk. v7 retains 74 random bits, which is
// far beyond guessing; only the timestamp is inferable, never the whole value.
package id

import "github.com/google/uuid"

// New returns a new time-ordered identifier.
//
// It panics only if the system's cryptographic random source fails, which is not
// a condition any caller can meaningfully handle — a process that cannot generate
// random bytes cannot safely issue tokens or hash passwords either, so failing
// loudly at the point of breakage is better than propagating an error that every
// call site would ignore.
func New() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// IsZero reports whether an identifier is unset.
//
// Useful in GORM BeforeCreate hooks, which must assign an ID only when the caller
// has not supplied one — an offline client that generated its own ID must keep it.
func IsZero(u uuid.UUID) bool {
	return u == uuid.Nil
}
