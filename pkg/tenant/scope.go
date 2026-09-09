package tenant

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// ErrCrossTenantWrite is returned when a record carrying one tenant's identifier
// is written while another tenant is in context.
//
// This is not the same failure as ErrNoTenant. No tenant at all is usually a
// wiring mistake; a *mismatched* tenant is an attempt, deliberate or not, to
// write into someone else's data, and is worth distinguishing in logs.
var ErrCrossTenantWrite = errors.New("tenant: record belongs to a different tenant")

// unscopedSetting marks a statement as a deliberate cross-tenant operation.
const unscopedSetting = "tenant:unscoped"

// Unscoped disables tenant scoping for one chain of calls.
//
//	var all []Customer
//	tenant.Unscoped(db).Find(&all)   // every tenant's customers
//
// Legitimate uses are narrow: migrations, background jobs that fan out across
// tenants, and support tooling. It is a named function rather than a context
// flag precisely so that `grep -rn 'tenant.Unscoped'` lists every place the
// guarantee is suspended, and a reviewer can check each one.
//
// Call it fresh before every statement — never store its return value and issue
// more than one statement against it:
//
//	// WRONG: both Creates run against the same *gorm.DB.
//	utx := tenant.Unscoped(tx)
//	utx.Create(&a)
//	utx.Create(&b)  // silently reuses a's table and columns
//
//	// RIGHT.
//	tenant.Unscoped(tx).Create(&a)
//	tenant.Unscoped(tx).Create(&b)
//
// This is not specific to Unscoped — it is true of any chained GORM call
// (db.Where(...) has the identical trap) — but it is easy to hit here
// specifically, because Unscoped looks like a modifier you set once per
// transaction rather than per statement. The value it returns has already been
// "obtained" in GORM's terms, so the next chained call mutates its Statement in
// place instead of cloning one.
//
// This is unrelated to GORM's own db.Unscoped(), which disables soft-delete
// filtering. The two are independent and can be combined.
func Unscoped(db *gorm.DB) *gorm.DB {
	return db.Set(unscopedSetting, true)
}

func isUnscoped(db *gorm.DB) bool {
	v, ok := db.Statement.Settings.Load(unscopedSetting)
	if !ok {
		return false
	}
	on, _ := v.(bool)
	return on
}

// Register installs the tenant callbacks on a GORM connection.
//
// Call it once, immediately after opening the database and before any query
// runs. It is safe in single-tenant applications: every callback returns
// immediately for models that do not implement Tenanted, which is all of them if
// nothing opts in.
func Register(db *gorm.DB) error {
	cb := db.Callback()

	return errors.Join(
		cb.Create().Before("gorm:create").Register("tenant:create", stampOnCreate),
		cb.Query().Before("gorm:query").Register("tenant:query", scopeRead),
		cb.Row().Before("gorm:row").Register("tenant:row", scopeRead),
		cb.Update().Before("gorm:update").Register("tenant:update", scopeWrite),
		cb.Delete().Before("gorm:delete").Register("tenant:delete", scopeWrite),
	)
}

// tenantColumnOf returns the tenant column declared by the statement's model, or
// "" when the model has not opted in.
//
// It reads Schema.ModelType rather than inspecting Statement.Model or Dest
// directly, because GORM has already resolved those into a schema: a *Customer,
// a *[]Customer and a *[]*Customer all arrive here as the same ModelType. Raw
// SQL leaves Schema nil and is therefore never scoped — see the package doc.
func tenantColumnOf(stmt *gorm.Statement) string {
	if stmt == nil || stmt.Schema == nil {
		return ""
	}
	t, ok := reflect.New(stmt.Schema.ModelType).Interface().(Tenanted)
	if !ok {
		return ""
	}
	return t.TenantColumn()
}

// tenantField resolves the declared column to a model field, so a typo in
// TenantColumn() surfaces as a clear error rather than invalid SQL.
func tenantField(stmt *gorm.Statement, col string) (*schema.Field, error) {
	f := stmt.Schema.LookUpField(col)
	if f == nil {
		return nil, fmt.Errorf("tenant: %s declares tenant column %q, which is not a field on the model", stmt.Schema.Name, col)
	}
	return f, nil
}

// scopeRead adds the tenant predicate to SELECTs.
func scopeRead(db *gorm.DB) {
	col := tenantColumnOf(db.Statement)
	if col == "" || isUnscoped(db) {
		return
	}

	id, ok := FromContext(db.Statement.Context)
	if !ok {
		_ = db.AddError(ErrNoTenant)
		return
	}
	if _, err := tenantField(db.Statement, col); err != nil {
		_ = db.AddError(err)
		return
	}

	addTenantPredicate(db.Statement, col, id)
}

// scopeWrite adds the tenant predicate to UPDATEs and DELETEs.
//
// It carries one extra responsibility over scopeRead. GORM refuses an UPDATE or
// DELETE that carries no conditions, so a mistyped chain cannot rewrite a whole
// table. That check runs *after* this callback and is satisfied by any WHERE
// clause — including the one added here. Adding the tenant predicate
// unconditionally would therefore convert "GORM refuses this" into "GORM
// silently rewrites every row this tenant owns", which is worse for being
// plausible. So the same condition is required first, on the same terms GORM
// uses: an explicit WHERE, a primary key on the model, or AllowGlobalUpdate.
func scopeWrite(db *gorm.DB) {
	col := tenantColumnOf(db.Statement)
	if col == "" || isUnscoped(db) {
		return
	}

	id, ok := FromContext(db.Statement.Context)
	if !ok {
		_ = db.AddError(ErrNoTenant)
		return
	}
	if _, err := tenantField(db.Statement, col); err != nil {
		_ = db.AddError(err)
		return
	}
	if !db.AllowGlobalUpdate && !hasCondition(db.Statement) {
		_ = db.AddError(gorm.ErrMissingWhereClause)
		return
	}

	addTenantPredicate(db.Statement, col, id)
}

// hasCondition reports whether the statement already restricts which rows it
// touches, by the same two routes GORM accepts: an explicit WHERE, or a
// populated primary key on the model it was given.
func hasCondition(stmt *gorm.Statement) bool {
	if _, ok := stmt.Clauses["WHERE"]; ok {
		return true
	}
	if stmt.Schema == nil || stmt.ReflectValue.Kind() != reflect.Struct {
		return false
	}
	for _, pk := range stmt.Schema.PrimaryFields {
		if _, isZero := pk.ValueOf(stmt.Context, stmt.ReflectValue); !isZero {
			return true
		}
	}
	return false
}

// addTenantPredicate appends `AND <table>.<col> = <id>` to the statement.
func addTenantPredicate(stmt *gorm.Statement, col string, id uuid.UUID) {
	// Group the caller's existing conditions before appending, or an OR in them
	// swallows the tenant check. `Where("a = ?", 1).Or("b = ?", 2)` with a
	// naively appended AND builds:
	//
	//	a = 1 OR b = 2 AND tenant = x
	//
	// which SQL precedence reads as `a = 1 OR (b = 2 AND tenant = x)` — every
	// row matching `a = 1` leaks, regardless of who owns it. Wrapping first
	// gives `(a = 1 OR b = 2) AND tenant = x`. GORM's own soft-delete clause
	// does exactly this, for exactly this reason.
	if c, ok := stmt.Clauses["WHERE"]; ok {
		if where, ok := c.Expression.(clause.Where); ok && len(where.Exprs) >= 1 {
			for _, expr := range where.Exprs {
				if or, ok := expr.(clause.OrConditions); ok && len(or.Exprs) == 1 {
					where.Exprs = []clause.Expression{clause.And(where.Exprs...)}
					c.Expression = where
					stmt.Clauses["WHERE"] = c
					break
				}
			}
		}
	}

	// clause.CurrentTable qualifies the column, so the predicate stays correct
	// once a join puts a second business_id in scope.
	stmt.AddClause(clause.Where{Exprs: []clause.Expression{
		clause.Eq{
			Column: clause.Column{Table: clause.CurrentTable, Name: col},
			Value:  id,
		},
	}})
}

// stampOnCreate fills in the tenant column on INSERT, and rejects records that
// arrive already belonging to someone else.
//
// Stamping matters as much as scoping. If callers had to set business_id by hand
// on every create, they would eventually forget one — the same failure the query
// callback exists to prevent, in the other direction.
func stampOnCreate(db *gorm.DB) {
	stmt := db.Statement

	col := tenantColumnOf(stmt)
	if col == "" || isUnscoped(db) {
		return
	}

	id, ok := FromContext(stmt.Context)
	if !ok {
		_ = db.AddError(ErrNoTenant)
		return
	}
	field, err := tenantField(stmt, col)
	if err != nil {
		_ = db.AddError(err)
		return
	}

	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := range stmt.ReflectValue.Len() {
			if err := stampOne(stmt, field, stmt.ReflectValue.Index(i), id); err != nil {
				_ = db.AddError(err)
				return
			}
		}
	case reflect.Struct:
		if err := stampOne(stmt, field, stmt.ReflectValue, id); err != nil {
			_ = db.AddError(err)
		}
	}
}

func stampOne(stmt *gorm.Statement, field *schema.Field, rv reflect.Value, id uuid.UUID) error {
	current, isZero := field.ValueOf(stmt.Context, rv)
	if isZero {
		return field.Set(stmt.Context, rv, id)
	}

	// Already set. Honour it only if it agrees with the caller's own tenant;
	// anything else is a write into another tenant's data.
	switch cur := current.(type) {
	case uuid.UUID:
		if cur != id {
			return fmt.Errorf("%w: record carries %s, context holds %s", ErrCrossTenantWrite, cur, id)
		}
	case *uuid.UUID:
		if cur != nil && *cur != id {
			return fmt.Errorf("%w: record carries %s, context holds %s", ErrCrossTenantWrite, *cur, id)
		}
	default:
		// Fail closed. A tenant column of some other type cannot be compared
		// here, and guessing would mean allowing an unverified write.
		return fmt.Errorf("tenant: column %q on %s must be uuid.UUID or *uuid.UUID, got %T",
			field.DBName, stmt.Schema.Name, current)
	}
	return nil
}
