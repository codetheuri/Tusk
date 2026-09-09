package tenant

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// SettingName is the PostgreSQL run-time parameter carrying the active tenant.
//
// Policies read it with current_setting(SettingName, true) — the second argument
// asks for NULL instead of an error when it is unset. That choice is what makes
// the policy fail closed: an unset parameter compares as NULL, `column = NULL`
// is NULL rather than true, and no rows match. A connection that forgot to
// declare its tenant sees an empty database, not everyone's.
const SettingName = "app.tenant_id"

// policyName is the name given to the generated policy on every table. Fixed
// rather than derived, so DisableRLS and re-running EnableRLS both know it.
const policyName = "tusk_tenant_isolation"

// PolicyStatements returns the DDL that puts one table under tenant isolation.
//
// Exported as strings because row-level security belongs in a migration, next to
// the table it protects and under review — not applied at boot by whichever
// process starts first. Paste the output into a migration; EnableRLS is the
// convenience wrapper for tests and local development.
func PolicyStatements(table, column string) []string {
	t, c := quoteIdent(table), quoteIdent(column)
	predicate := fmt.Sprintf("%s = current_setting('%s', true)::uuid", c, SettingName)

	return []string{
		fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", t),

		// FORCE is not optional. Without it PostgreSQL exempts the table's owner
		// from its own policies, and application code very often connects as the
		// owner — so the protection would be installed, visible in \d, and doing
		// nothing at all.
		fmt.Sprintf("ALTER TABLE %s FORCE ROW LEVEL SECURITY", t),

		fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s", quoteIdent(policyName), t),

		// USING filters what a statement may see; WITH CHECK constrains what it
		// may write. Both are needed: USING alone would let a tenant INSERT a row
		// stamped with someone else's identifier, or UPDATE one of its own rows
		// to move it across the boundary.
		fmt.Sprintf("CREATE POLICY %s ON %s USING (%s) WITH CHECK (%s)",
			quoteIdent(policyName), t, predicate, predicate),
	}
}

// DropPolicyStatements reverses PolicyStatements, for a migration's down step.
func DropPolicyStatements(table string) []string {
	t := quoteIdent(table)
	return []string{
		fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s", quoteIdent(policyName), t),
		fmt.Sprintf("ALTER TABLE %s NO FORCE ROW LEVEL SECURITY", t),
		fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY", t),
	}
}

// EnableRLS applies the policies for the given models.
//
// Each model must implement Tenanted; passing one that does not is a mistake
// worth reporting rather than skipping, because the caller clearly believed it
// was protected.
func EnableRLS(db *gorm.DB, models ...any) error {
	return applyPerModel(db, models, func(table, column string) []string {
		return PolicyStatements(table, column)
	})
}

// DisableRLS removes them again.
func DisableRLS(db *gorm.DB, models ...any) error {
	return applyPerModel(db, models, func(table, _ string) []string {
		return DropPolicyStatements(table)
	})
}

func applyPerModel(db *gorm.DB, models []any, build func(table, column string) []string) error {
	for _, model := range models {
		table, column, err := tableAndTenantColumn(db, model)
		if err != nil {
			return err
		}
		for _, stmt := range build(table, column) {
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("tenant: %q: %w", stmt, err)
			}
		}
	}
	return nil
}

// tableAndTenantColumn resolves a model to the table and column the policy needs.
func tableAndTenantColumn(db *gorm.DB, model any) (string, string, error) {
	t, ok := model.(Tenanted)
	if !ok {
		return "", "", fmt.Errorf("tenant: %T does not implement Tenanted, so it has no tenant column to protect", model)
	}

	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return "", "", fmt.Errorf("tenant: cannot parse %T: %w", model, err)
	}

	column := t.TenantColumn()
	if stmt.Schema.LookUpField(column) == nil {
		return "", "", fmt.Errorf("tenant: %s declares tenant column %q, which is not a field on the model", stmt.Schema.Name, column)
	}
	return stmt.Schema.Table, column, nil
}

// Transaction runs fn with the tenant declared to PostgreSQL, so row-level
// security applies for the duration.
//
// The declaration has to live inside a transaction. SET LOCAL is scoped to one,
// and reverts on commit or rollback — which is the property that makes this safe
// against connection pooling. A plain SET would persist on the pooled connection
// and hand the next request, belonging to a different tenant, whatever this one
// left behind.
func Transaction(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	id, err := MustFromContext(ctx)
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// set_config(name, value, is_local=true) is SET LOCAL in function form.
		// SET cannot take a bind parameter, so the literal spelling would mean
		// interpolating a value into SQL; this keeps it a parameter.
		if err := tx.Exec("SELECT set_config(?, ?, true)", SettingName, id.String()).Error; err != nil {
			return fmt.Errorf("tenant: failed to declare %s: %w", SettingName, err)
		}
		return fn(tx)
	})
}

// VerifyEnforcement reports whether row-level security actually binds the role
// this connection uses.
//
// Worth calling at startup and logging loudly. PostgreSQL exempts superusers and
// any role holding BYPASSRLS from every policy, without warning — so a system can
// have correct policies, a passing review, and no protection whatsoever. The
// failure is invisible precisely because nothing errors.
func VerifyEnforcement(db *gorm.DB) error {
	var role struct {
		Name     string
		Super    bool
		BypassRL bool
	}

	err := db.Raw(`
		SELECT rolname AS name, rolsuper AS super, rolbypassrls AS bypass_rl
		FROM pg_roles WHERE rolname = current_user
	`).Scan(&role).Error
	if err != nil {
		return fmt.Errorf("tenant: cannot determine whether RLS is enforced: %w", err)
	}

	switch {
	case role.Super:
		return fmt.Errorf("tenant: connected as superuser %q, which bypasses every row-level security policy; "+
			"run the application as a dedicated non-superuser role", role.Name)
	case role.BypassRL:
		return fmt.Errorf("tenant: role %q holds BYPASSRLS, so row-level security does not apply to it; "+
			"revoke it with ALTER ROLE %s NOBYPASSRLS", role.Name, role.Name)
	default:
		return nil
	}
}

// quoteIdent quotes a SQL identifier. These come from struct tags rather than
// from users, but a table called "order" still needs quoting to parse.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
