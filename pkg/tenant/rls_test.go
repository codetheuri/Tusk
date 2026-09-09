package tenant_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/codetheuri/tusk/pkg/id"
	"github.com/codetheuri/tusk/pkg/tenant"
	"github.com/codetheuri/tusk/pkg/testdb"
)

// Row-level security is the layer that still holds when the application layer is
// wrong. Testing it therefore means bypassing the application layer on purpose:
// every query below goes through tenant.Unscoped, so nothing but PostgreSQL is
// deciding which rows come back.
//
// The tests connect as a dedicated non-superuser role. This is not fastidiousness
// — PostgreSQL exempts superusers and BYPASSRLS roles from every policy, so the
// same suite run as the usual development superuser would pass without a single
// policy being consulted.

const (
	rlsRole     = "tusk_rls_tester"
	rlsPassword = "tusk_rls_tester_pw"
)

type rlsNote struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	BusinessID uuid.UUID `gorm:"type:uuid;not null;index"`
	Title      string
}

func (rlsNote) TableName() string    { return "tenant_test_rls_notes" }
func (rlsNote) TenantColumn() string { return "business_id" }

func (n *rlsNote) BeforeCreate(*gorm.DB) error {
	if id.IsZero(n.ID) {
		n.ID = id.New()
	}
	return nil
}

// rlsFixture builds a table under policy, seeds it as the owner, and returns a
// second connection held by a role the policies actually bind.
func rlsFixture(t *testing.T) (asRole *gorm.DB, alice, bob uuid.UUID) {
	t.Helper()

	owner := testdb.Connect(t)

	if err := owner.AutoMigrate(&rlsNote{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	t.Cleanup(func() { owner.Exec("DROP TABLE IF EXISTS " + (rlsNote{}).TableName()) })

	if err := tenant.EnableRLS(owner, &rlsNote{}); err != nil {
		t.Fatalf("enable rls: %v", err)
	}

	alice, bob = id.New(), id.New()
	seed := []rlsNote{
		{ID: id.New(), BusinessID: alice, Title: "alice-secret"},
		{ID: id.New(), BusinessID: bob, Title: "bob-secret"},
	}
	// Explicitly unscoped: the fixture is deliberately building rows for two
	// tenants at once, which is precisely what the application layer forbids.
	// Going through it here would make the fixture depend on the layer these
	// tests exist to be independent of.
	if err := tenant.Unscoped(owner).Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	createRole(t, owner)
	return connectAsRole(t), alice, bob
}

func createRole(t *testing.T, owner *gorm.DB) {
	t.Helper()

	// Idempotent: a previous run that died before cleanup must not block this one.
	stmts := []string{
		fmt.Sprintf(`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
				CREATE ROLE %s LOGIN PASSWORD '%s';
			END IF;
		END $$`, rlsRole, rlsRole, rlsPassword),
		fmt.Sprintf(`GRANT USAGE ON SCHEMA public TO %s`, rlsRole),
		fmt.Sprintf(`GRANT SELECT, INSERT, UPDATE, DELETE ON %s TO %s`, (rlsNote{}).TableName(), rlsRole),
	}
	for _, s := range stmts {
		if err := owner.Exec(s).Error; err != nil {
			if testdb.Required() {
				t.Fatalf("could not prepare the unprivileged test role: %v", err)
			}
			t.Skipf("cannot create a test role on this database, skipping RLS tests: %v", err)
		}
	}

	t.Cleanup(func() {
		owner.Exec(fmt.Sprintf("DROP OWNED BY %s", rlsRole))
		owner.Exec(fmt.Sprintf("DROP ROLE IF EXISTS %s", rlsRole))
	})
}

// connectAsRole opens a second connection as the unprivileged role, reusing the
// host and database the harness was pointed at.
func connectAsRole(t *testing.T) *gorm.DB {
	t.Helper()

	// Parsed rather than string-substituted, so TEST_DATABASE_URL works in either
	// the keyword/value or the URL form.
	cfg, err := pgx.ParseConfig(testdb.DSN())
	if err != nil {
		t.Fatalf("parse test DSN: %v", err)
	}
	sslmode := "disable"
	if cfg.TLSConfig != nil {
		sslmode = "require"
	}
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Database, rlsRole, rlsPassword, sslmode)

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open as %s: %v", rlsRole, err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm open as %s: %v", rlsRole, err)
	}
	if err := tenant.Register(db); err != nil {
		t.Fatalf("register callbacks: %v", err)
	}
	return db
}

// TestRLS_HidesOtherTenantsEvenWithApplicationScopingBypassed is the whole point
// of the layer: tenant.Unscoped disables everything the Go code does, and the
// rows still do not appear.
func TestRLS_HidesOtherTenantsEvenWithApplicationScopingBypassed(t *testing.T) {
	db, alice, _ := rlsFixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	err := tenant.Transaction(ctx, db, func(tx *gorm.DB) error {
		var got []rlsNote
		if err := tenant.Unscoped(tx).Find(&got).Error; err != nil {
			return err
		}
		if len(got) != 1 {
			t.Errorf("got %d rows, want only alice's 1", len(got))
		}
		for _, n := range got {
			if n.BusinessID != alice {
				t.Errorf("database returned a row owned by %s", n.BusinessID)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("transaction: %v", err)
	}
}

// TestRLS_WithNoTenantDeclaredReturnsNothing checks the direction that matters.
// An unset parameter must mean "no rows", not "all rows".
func TestRLS_WithNoTenantDeclaredReturnsNothing(t *testing.T) {
	db, _, _ := rlsFixture(t)

	var count int64
	if err := db.Raw("SELECT count(*) FROM " + (rlsNote{}).TableName()).Scan(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("a connection that declared no tenant saw %d rows", count)
	}
}

// TestRLS_RefusesAWriteStampedWithAnotherTenant covers WITH CHECK. USING alone
// would let a tenant insert rows it could then never see — and that another
// tenant would.
func TestRLS_RefusesAWriteStampedWithAnotherTenant(t *testing.T) {
	db, alice, bob := rlsFixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	err := tenant.Transaction(ctx, db, func(tx *gorm.DB) error {
		return tenant.Unscoped(tx).Create(&rlsNote{
			ID:         id.New(),
			BusinessID: bob,
			Title:      "planted",
		}).Error
	})

	if err == nil {
		t.Fatal("the database accepted a row belonging to another tenant")
	}
}

// TestRLS_RefusesToMoveARowAcrossTenants is the same guarantee for UPDATE.
func TestRLS_RefusesToMoveARowAcrossTenants(t *testing.T) {
	db, alice, bob := rlsFixture(t)
	ctx := tenant.WithTenant(context.Background(), alice)

	err := tenant.Transaction(ctx, db, func(tx *gorm.DB) error {
		return tenant.Unscoped(tx).
			Model(&rlsNote{}).
			Where("business_id = ?", alice).
			Update("business_id", bob).Error
	})

	if err == nil {
		t.Fatal("the database allowed a row to be moved to another tenant")
	}
}

// TestVerifyEnforcement_DistinguishesTheTwoRoles pins the check that stops this
// entire layer from being quietly inert in production.
func TestVerifyEnforcement_DistinguishesTheTwoRoles(t *testing.T) {
	asRole, _, _ := rlsFixture(t)

	if err := tenant.VerifyEnforcement(asRole); err != nil {
		t.Errorf("expected RLS to bind %s, got: %v", rlsRole, err)
	}

	owner := testdb.Connect(t)
	if err := tenant.VerifyEnforcement(owner); err == nil {
		t.Error("expected the superuser connection to be reported as exempt from RLS")
	} else {
		t.Logf("correctly reported: %v", err)
	}
}
