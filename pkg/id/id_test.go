package id

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNew_ProducesVersion7(t *testing.T) {
	u := New()
	if got := u.Version(); got != 7 {
		t.Errorf("UUID version = %d, want 7", got)
	}
	if got := u.Variant(); got != uuid.RFC4122 {
		t.Errorf("UUID variant = %v, want RFC4122", got)
	}
}

// The whole reason for choosing v7 over v4 is that keys sort by creation time,
// which is what preserves B-tree insert locality. If that ordering ever broke,
// the choice would have no remaining benefit over v4 — so it is worth asserting.
func TestNew_IsTimeOrdered(t *testing.T) {
	const n = 1000
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = New()
	}

	for i := 1; i < n; i++ {
		prev, curr := ids[i-1].String(), ids[i].String()
		if curr < prev {
			t.Fatalf("identifier %d sorts before its predecessor:\n  prev: %s\n  curr: %s", i, prev, curr)
		}
	}
}

func TestNew_IsUnique(t *testing.T) {
	const n = 10000
	seen := make(map[uuid.UUID]struct{}, n)
	for i := 0; i < n; i++ {
		u := New()
		if _, dup := seen[u]; dup {
			t.Fatalf("duplicate identifier generated after %d draws: %s", i, u)
		}
		seen[u] = struct{}{}
	}
}

// The embedded timestamp is a documented property, not an accident — the tradeoff
// it implies (an ID reveals its creation time) is called out in the package docs.
func TestNew_EmbedsCurrentTimestamp(t *testing.T) {
	before := time.Now().Add(-time.Second)
	u := New()
	after := time.Now().Add(time.Second)

	sec, nsec := u.Time().UnixTime()
	created := time.Unix(sec, nsec)

	if created.Before(before) || created.After(after) {
		t.Errorf("embedded timestamp %s is outside the expected window [%s, %s]", created, before, after)
	}
}

func TestIsZero(t *testing.T) {
	if !IsZero(uuid.Nil) {
		t.Error("IsZero(uuid.Nil) = false, want true")
	}
	if IsZero(New()) {
		t.Error("IsZero(New()) = true, want false")
	}
}
