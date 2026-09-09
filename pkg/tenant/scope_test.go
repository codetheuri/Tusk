package tenant_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/v2/pkg/id"
	"github.com/codetheuri/tusk/v2/pkg/tenant"
	"github.com/codetheuri/tusk/v2/pkg/testdb"
)

// These tests run against real PostgreSQL. The thing being verified is the SQL
// GORM ends up sending, and asserting on SQL *text* would pass while the query
// still returned the wrong rows. So every case here asks the only question that
// matters: given two tenants' data in one table, which rows come back?

// scopedNote opts in to tenancy.
type scopedNote struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	BusinessID uuid.UUID `gorm:"type:uuid;not null;index"`
	Title      string
}

func (scopedNote) TableName() string    { return "tenant_test_notes" }
func (scopedNote) TenantColumn() string { return "business_id" }

func (n *scopedNote) BeforeCreate(*gorm.DB) error {
	if id.IsZero(n.ID) {
		n.ID = id.New()
	}
	return nil
}

// plainNote does not. It exists to prove D5: a model that never mentions
// tenancy must behave as though this package were not installed.
type plainNote struct {
	ID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Title string
}

func (plainNote) TableName() string { return "tenant_test_plain_notes" }

func (n *plainNote) BeforeCreate(*gorm.DB) error {
	if id.IsZero(n.ID) {
		n.ID = id.New()
	}
	return nil
}

// fixture returns a database with the callbacks installed and one note owned by
// each of two tenants.
func fixture(t *testing.T) (db *gorm.DB, alice, bob uuid.UUID) {
	t.Helper()

	db = testdb.Connect(t)

	// The callbacks are already installed by the harness, exactly as they are by
	// database.Connect in production.
	if err := db.AutoMigrate(&scopedNote{}, &plainNote{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	alice, bob = id.New(), id.New()

	// Seeded through Unscoped rather than two separate contexts: the seed is
	// setup, and using the code under test to build its own fixture would let a
	// bug hide itself.
	seed := []scopedNote{
		{ID: id.New(), BusinessID: alice, Title: "alice-one"},
		{ID: id.New(), BusinessID: alice, Title: "alice-two"},
		{ID: id.New(), BusinessID: bob, Title: "bob-one"},
	}
	if err := tenant.Unscoped(db).Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	return db, alice, bob
}

func TestQuery_ReturnsOnlyTheTenantsOwnRows(t *testing.T) {
	db, alice, _ := fixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	var got []scopedNote
	if err := db.WithContext(ctx).Find(&got).Error; err != nil {
		t.Fatalf("find: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d notes, want alice's 2", len(got))
	}
	for _, n := range got {
		if n.BusinessID != alice {
			t.Errorf("leaked a note belonging to %s", n.BusinessID)
		}
	}
}

// TestQuery_OrConditionsCannotEscapeTheTenantPredicate covers the precedence
// trap that makes a naive implementation of this package unsafe.
//
// `WHERE title = 'alice-one' OR title = 'bob-one' AND business_id = alice`
// parses as `title = 'alice-one' OR (title = 'bob-one' AND business_id = alice)`.
// Bob's note matches the left branch and comes back to Alice.
func TestQuery_OrConditionsCannotEscapeTheTenantPredicate(t *testing.T) {
	db, alice, _ := fixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	var got []scopedNote
	err := db.WithContext(ctx).
		Where("title = ?", "alice-one").
		Or("title = ?", "bob-one").
		Find(&got).Error
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d notes, want only alice-one", len(got))
	}
	if got[0].Title != "alice-one" {
		t.Errorf("got %q, want alice-one", got[0].Title)
	}
}

// TestQuery_ByPrimaryKeyAcrossTenantsFindsNothing is the "404, not 403" case.
// Bob knows Alice's note ID and asks for it directly.
func TestQuery_ByPrimaryKeyAcrossTenantsFindsNothing(t *testing.T) {
	db, alice, bob := fixture(t)

	var target scopedNote
	if err := tenant.Unscoped(db).Where("business_id = ?", alice).First(&target).Error; err != nil {
		t.Fatalf("locate alice's note: %v", err)
	}

	var got scopedNote
	err := db.WithContext(tenant.WithTenant(context.Background(), bob)).
		First(&got, "id = ?", target.ID).Error

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("got %v, want ErrRecordNotFound — a known ID must not be a way in", err)
	}
}

// TestQuery_WithNoTenantFailsClosed is the design decision that separates this
// from a filter someone forgot to apply: absent scoping is an error, never a
// silent full-table read.
func TestQuery_WithNoTenantFailsClosed(t *testing.T) {
	db, _, _ := fixture(t)

	var got []scopedNote
	err := db.WithContext(context.Background()).Find(&got).Error

	if !errors.Is(err, tenant.ErrNoTenant) {
		t.Fatalf("got %v, want ErrNoTenant", err)
	}
	if len(got) != 0 {
		t.Fatalf("query returned %d rows despite erroring", len(got))
	}
}

func TestCreate_StampsTheTenantFromContext(t *testing.T) {
	db, alice, _ := fixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	note := scopedNote{Title: "written-without-a-business-id"}
	if err := db.WithContext(ctx).Create(&note).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	if note.BusinessID != alice {
		t.Errorf("got business_id %s, want %s", note.BusinessID, alice)
	}

	// Confirm it reached the row, not just the struct.
	var stored scopedNote
	if err := tenant.Unscoped(db).First(&stored, "id = ?", note.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored.BusinessID != alice {
		t.Errorf("stored business_id %s, want %s", stored.BusinessID, alice)
	}
}

func TestCreate_RejectsARecordBelongingToAnotherTenant(t *testing.T) {
	db, alice, bob := fixture(t)
	ctx := tenant.WithTenant(context.Background(), bob)

	note := scopedNote{Title: "planted", BusinessID: alice}
	err := db.WithContext(ctx).Create(&note).Error

	if !errors.Is(err, tenant.ErrCrossTenantWrite) {
		t.Fatalf("got %v, want ErrCrossTenantWrite", err)
	}
}

func TestCreate_StampsEveryRecordInABatch(t *testing.T) {
	db, alice, _ := fixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	batch := []scopedNote{{Title: "a"}, {Title: "b"}, {Title: "c"}}
	if err := db.WithContext(ctx).Create(&batch).Error; err != nil {
		t.Fatalf("create batch: %v", err)
	}

	for i, n := range batch {
		if n.BusinessID != alice {
			t.Errorf("batch[%d] got business_id %s, want %s", i, n.BusinessID, alice)
		}
	}
}

func TestUpdate_CannotReachAnotherTenantsRow(t *testing.T) {
	db, alice, bob := fixture(t)

	var target scopedNote
	if err := tenant.Unscoped(db).Where("business_id = ?", alice).First(&target).Error; err != nil {
		t.Fatalf("locate alice's note: %v", err)
	}

	res := db.WithContext(tenant.WithTenant(context.Background(), bob)).
		Model(&scopedNote{}).
		Where("id = ?", target.ID).
		Update("title", "defaced")

	if res.Error != nil {
		t.Fatalf("update: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Errorf("updated %d rows across a tenant boundary", res.RowsAffected)
	}

	var after scopedNote
	if err := tenant.Unscoped(db).First(&after, "id = ?", target.ID).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if after.Title != target.Title {
		t.Errorf("title changed to %q; the row was modified", after.Title)
	}
}

func TestDelete_CannotReachAnotherTenantsRow(t *testing.T) {
	db, alice, bob := fixture(t)

	var target scopedNote
	if err := tenant.Unscoped(db).Where("business_id = ?", alice).First(&target).Error; err != nil {
		t.Fatalf("locate alice's note: %v", err)
	}

	res := db.WithContext(tenant.WithTenant(context.Background(), bob)).
		Where("id = ?", target.ID).
		Delete(&scopedNote{})

	if res.Error != nil {
		t.Fatalf("delete: %v", res.Error)
	}
	if res.RowsAffected != 0 {
		t.Errorf("deleted %d rows across a tenant boundary", res.RowsAffected)
	}

	var count int64
	tenant.Unscoped(db).Model(&scopedNote{}).Where("id = ?", target.ID).Count(&count)
	if count != 1 {
		t.Error("alice's note was deleted by bob")
	}
}

// TestUpdate_StillRefusesAnUnconditionalStatement guards a regression this
// package could easily introduce.
//
// GORM rejects an UPDATE with no conditions, so a mistyped chain cannot rewrite
// a table. Its check runs after the tenant callback and is satisfied by any
// WHERE clause — so adding the tenant predicate unconditionally would turn a
// refusal into "quietly rewrite every row this tenant owns".
func TestUpdate_StillRefusesAnUnconditionalStatement(t *testing.T) {
	db, alice, _ := fixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	err := db.WithContext(ctx).Model(&scopedNote{}).Update("title", "everything").Error

	if !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Fatalf("got %v, want ErrMissingWhereClause", err)
	}

	var untouched int64
	tenant.Unscoped(db).Model(&scopedNote{}).Where("title = ?", "everything").Count(&untouched)
	if untouched != 0 {
		t.Errorf("%d rows were rewritten anyway", untouched)
	}
}

func TestUnscoped_ReachesEveryTenant(t *testing.T) {
	db, _, _ := fixture(t)

	var got []scopedNote
	if err := tenant.Unscoped(db).Find(&got).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d notes, want all 3", len(got))
	}
}

// TestUntenantedModelIsUnaffected is the D5 proof at package level: a model that
// does not implement Tenanted must read, write and delete with no tenant in
// context at all, exactly as it would without this package.
func TestUntenantedModelIsUnaffected(t *testing.T) {
	db, _, _ := fixture(t)
	ctx := context.Background()

	note := plainNote{Title: "single-tenant"}
	if err := db.WithContext(ctx).Create(&note).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	var got []plainNote
	if err := db.WithContext(ctx).Find(&got).Error; err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d notes, want 1", len(got))
	}

	if err := db.WithContext(ctx).Delete(&plainNote{}, "id = ?", note.ID).Error; err != nil {
		t.Fatalf("delete: %v", err)
	}
}

// TestUnscoped_MustBeCalledFreshPerStatement pins the trap documented on
// Unscoped: its return value behaves like any chained GORM call (db.Where(...)
// has the identical property) — reusing it across two statements mutates the
// first statement's Table and Schema in place rather than starting a new one.
// This is not a bug in Unscoped; it is a property of GORM this package's doc
// exists to warn callers about, demonstrated here so the warning is never
// allowed to go stale.
//
// Reusing it across two differently-shaped models is severe enough to panic
// inside GORM's reflection rather than merely writing wrong data — which is
// arguably the better outcome, since it fails loudly instead of corrupting a
// row. The panic is caught here specifically to demonstrate that this is what
// misuse produces, not to treat it as acceptable.
func TestUnscoped_MustBeCalledFreshPerStatement(t *testing.T) {
	db, _, _ := fixture(t)

	misuseCrashed := func() (crashed bool) {
		defer func() {
			if recover() != nil {
				crashed = true
			}
		}()
		// WRONG usage: one instance reused for two creates of different shapes.
		reused := tenant.Unscoped(db)
		if err := reused.Create(&scopedNote{Title: "first"}).Error; err != nil {
			t.Fatalf("first create: %v", err)
		}
		_ = reused.Create(&plainNote{Title: "second"}).Error
		return false
	}()

	if !misuseCrashed {
		// It did not panic this time, which can happen depending on the two
		// models' field counts. Either way the row must not have landed cleanly
		// in plainNote's own table.
		var wrongTableCount int64
		tenant.Unscoped(db).Model(&plainNote{}).Where("title = ?", "second").Count(&wrongTableCount)
		if wrongTableCount == 1 {
			t.Fatal("reusing tenant.Unscoped(db) across two Creates worked correctly — " +
				"this test's premise is stale, but treat this failure as real until it is " +
				"actually verified fixed upstream")
		}
	}

	// The fix: a fresh call before each statement. Unaffected by the misuse above.
	if err := tenant.Unscoped(db).Create(&plainNote{Title: "correct"}).Error; err != nil {
		t.Fatalf("fresh Unscoped create: %v", err)
	}
	var correct plainNote
	if err := tenant.Unscoped(db).First(&correct, "title = ?", "correct").Error; err != nil {
		t.Fatalf("the correctly-created row is missing: %v", err)
	}
}
