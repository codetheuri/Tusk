package query_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/pkg/query"
	"github.com/codetheuri/tusk/pkg/testdb"
)

// These run against real PostgreSQL because the defect they cover — LIKE's
// case sensitivity — is specific to the database engine. A fake would not
// reproduce it: it would simply do whatever the test expected.

type queryTestItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string
	CreatedAt int64
}

func (queryTestItem) TableName() string { return "query_pkg_test_items" }

func setup(t *testing.T) *gorm.DB {
	t.Helper()
	db := testdb.Connect(t)
	if err := db.AutoMigrate(&queryTestItem{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS query_pkg_test_items") })

	seed := []queryTestItem{
		{ID: uuid.New(), Name: "Wanjiku Njoroge", CreatedAt: 1},
		{ID: uuid.New(), Name: "John Otieno", CreatedAt: 2},
		{ID: uuid.New(), Name: "Wanjiru Kamau", CreatedAt: 3},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

var cfg = query.Config{
	DefaultPerPage:  20,
	MaxPerPage:      100,
	AllowedSearches: []string{"name"},
	AllowedSorts:    map[string]string{"name": "name", "created_at": "created_at"},
}

// TestApply_SearchIsCaseInsensitive is the regression test for the LIKE/ILIKE
// defect: a lowercase search must find a capitalised name, on PostgreSQL, which
// is the only database Tusk targets.
func TestApply_SearchIsCaseInsensitive(t *testing.T) {
	db := setup(t)

	var got []queryTestItem
	err := query.Apply(db, query.Query{Search: "wanj"}, cfg).Find(&got).Error
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2 (Wanjiku, Wanjiru)", len(got))
	}
}

func TestApply_SearchIgnoresUnlistedColumns(t *testing.T) {
	db := setup(t)

	// AllowedSearches lists only "name", so this must not search anything else
	// even if the query struct carried a filter naming another column.
	restrictedCfg := query.Config{AllowedSearches: []string{"name"}}
	var got []queryTestItem
	if err := query.Apply(db, query.Query{Search: "otieno"}, restrictedCfg).Find(&got).Error; err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(got) != 1 || got[0].Name != "John Otieno" {
		t.Fatalf("got %+v, want exactly John Otieno", got)
	}
}

func TestPaginate_ReturnsCorrectMetaAcrossPages(t *testing.T) {
	db := setup(t)
	ctx := context.Background()

	first, meta, err := query.Paginate[queryTestItem](ctx, db, query.Query{Page: 1, PerPage: 2}, cfg)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("page 1 has %d items, want 2", len(first))
	}
	if meta.Total != 3 || meta.TotalPages != 2 || !meta.HasNext || meta.HasPrevious {
		t.Errorf("page 1 meta = %+v", meta)
	}

	second, meta2, err := query.Paginate[queryTestItem](ctx, db, query.Query{Page: 2, PerPage: 2}, cfg)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("page 2 has %d items, want 1", len(second))
	}
	if meta2.HasNext || !meta2.HasPrevious {
		t.Errorf("page 2 meta = %+v", meta2)
	}

	seen := map[uuid.UUID]bool{}
	for _, item := range append(first, second...) {
		if seen[item.ID] {
			t.Errorf("item %s returned on more than one page", item.ID)
		}
		seen[item.ID] = true
	}
	if len(seen) != 3 {
		t.Errorf("saw %d distinct items across both pages, want 3", len(seen))
	}
}

func TestPaginate_PerPageIsBoundedByMaxPerPage(t *testing.T) {
	db := setup(t)
	bounded := cfg
	bounded.MaxPerPage = 1

	items, meta, err := query.Paginate[queryTestItem](context.Background(), db, query.Query{PerPage: 1000}, bounded)
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if len(items) != 1 || meta.PerPage != 1 {
		t.Errorf("got %d items with per_page=%d, want 1 item, per_page=1", len(items), meta.PerPage)
	}
}

func TestApply_SortsByAllowedColumnOnly(t *testing.T) {
	db := setup(t)

	var got []queryTestItem
	q := query.Query{Sorts: []query.Sort{{Field: "created_at", Order: query.SortDesc}}}
	if err := query.Apply(db, q, cfg).Find(&got).Error; err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(got) != 3 || got[0].Name != "Wanjiru Kamau" {
		t.Fatalf("expected newest-first, got %+v", got)
	}
}
