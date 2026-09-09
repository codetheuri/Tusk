package view_test

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/codetheuri/tusk/v2/pkg/view"
)

// templates returns a small but realistic set: a base wrapping the document, a
// layout contributing a shared block, and two pages defining the same block
// names differently.
func templates() fstest.MapFS {
	return fstest.MapFS{
		"tpl/base.html": {Data: []byte(
			`{{define "base"}}<html><title>{{template "title" .}}</title><body>{{template "content" .}}</body></html>{{end}}`)},
		"tpl/layout.html": {Data: []byte(
			`{{define "sidebar"}}<nav>{{.User}}</nav>{{end}}`)},
		"tpl/dashboard.html": {Data: []byte(
			`{{define "title"}}Dashboard{{end}}{{define "content"}}{{template "sidebar" .}}<h1>{{.Heading}}</h1>{{end}}`)},
		"tpl/login.html": {Data: []byte(
			`{{define "title"}}Sign in{{end}}{{define "content"}}<form></form>{{end}}`)},
	}
}

func newEngine(t *testing.T, fsys fstest.MapFS) *view.Engine {
	t.Helper()
	e, err := view.New(view.Config{
		FS:      fsys,
		Dir:     "tpl",
		Layouts: []string{"base.html", "layout.html"},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	return e
}

func TestRender_ComposesLayoutsWithThePage(t *testing.T) {
	e := newEngine(t, templates())

	rec := httptest.NewRecorder()
	err := e.Render(rec, http.StatusOK, "dashboard.html", map[string]any{
		"User":    "ada",
		"Heading": "Overview",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	body := rec.Body.String()
	for _, want := range []string{"<title>Dashboard</title>", "<nav>ada</nav>", "<h1>Overview</h1>"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\ngot: %s", want, body)
		}
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type %q", ct)
	}
}

// TestRender_PagesDoNotCollide is why each page gets its own template set. Both
// pages define "title" and "content"; a single shared set would let whichever
// parsed last win for every page.
func TestRender_PagesDoNotCollide(t *testing.T) {
	e := newEngine(t, templates())

	for page, want := range map[string]string{
		"dashboard.html": "Dashboard",
		"login.html":     "Sign in",
	} {
		rec := httptest.NewRecorder()
		if err := e.Render(rec, http.StatusOK, page, map[string]any{}); err != nil {
			t.Fatalf("render %s: %v", page, err)
		}
		if !strings.Contains(rec.Body.String(), "<title>"+want+"</title>") {
			t.Errorf("%s rendered the wrong title: %s", page, rec.Body.String())
		}
	}
}

// TestRender_WritesNothingWhenTheTemplateFails is the defect this package exists
// to prevent. html/template streams as it executes, so a naive implementation
// has already sent 200 and a partial document before the error appears.
func TestRender_WritesNothingWhenTheTemplateFails(t *testing.T) {
	fsys := templates()
	// A field the data type does not have. Execution fails only once it reaches
	// that action — by which point "before" has already been produced, which is
	// exactly the situation a streaming renderer cannot recover from.
	fsys["tpl/broken.html"] = &fstest.MapFile{Data: []byte(
		`{{define "title"}}Broken{{end}}{{define "content"}}<div>before</div>{{.NoSuchField}}{{end}}`)}

	e := newEngine(t, fsys)

	rec := httptest.NewRecorder()
	err := e.Render(rec, http.StatusOK, "broken.html", struct{ Heading string }{"ok"})

	if err == nil {
		t.Fatal("expected a render error")
	}
	if rec.Body.Len() != 0 {
		t.Errorf("a partial page reached the client: %q", rec.Body.String())
	}
	if rec.Flushed {
		t.Error("the response was flushed, so the handler can no longer set a status")
	}
}

// TestNew_RejectsABrokenTemplateAtStartup: a page that does not compile must
// stop the process, not wait for a visitor.
func TestNew_RejectsABrokenTemplateAtStartup(t *testing.T) {
	fsys := templates()
	fsys["tpl/bad.html"] = &fstest.MapFile{Data: []byte(`{{define "content"}}{{ .Unclosed `)}

	_, err := view.New(view.Config{
		FS:      fsys,
		Dir:     "tpl",
		Layouts: []string{"base.html", "layout.html"},
	})
	if err == nil {
		t.Fatal("expected New to reject a malformed template")
	}
	if !strings.Contains(err.Error(), "bad.html") {
		t.Errorf("error should name the offending file, got: %v", err)
	}
}

// TestNew_RejectsAPageMissingTheEntryBlock catches the mistake of adding a page
// that forgets {{define "content"}} — it would otherwise render as a blank body.
func TestNew_RejectsAPageMissingTheEntryBlock(t *testing.T) {
	fsys := fstest.MapFS{
		"tpl/layout.html": {Data: []byte(`{{define "sidebar"}}x{{end}}`)},
		"tpl/page.html":   {Data: []byte(`<p>no blocks at all</p>`)},
	}

	_, err := view.New(view.Config{FS: fsys, Dir: "tpl", Layouts: []string{"layout.html"}})
	if err == nil {
		t.Fatal("expected New to reject a page that defines no entry template")
	}
}

func TestRender_UnknownPageIsAnError(t *testing.T) {
	e := newEngine(t, templates())

	err := e.Render(httptest.NewRecorder(), http.StatusOK, "nope.html", nil)
	if err == nil {
		t.Fatal("expected an error for an unknown page")
	}
}

// TestReload_PicksUpEdits covers the development path. Without Reload the first
// parse is cached forever, which is correct in production and maddening locally.
func TestReload_PicksUpEdits(t *testing.T) {
	fsys := templates()

	e, err := view.New(view.Config{
		FS:      fsys,
		Dir:     "tpl",
		Layouts: []string{"base.html", "layout.html"},
		Reload:  true,
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	rec := httptest.NewRecorder()
	if err := e.Render(rec, http.StatusOK, "login.html", nil); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "Sign in") {
		t.Fatalf("unexpected first render: %s", rec.Body.String())
	}

	fsys["tpl/login.html"] = &fstest.MapFile{Data: []byte(
		`{{define "title"}}Edited{{end}}{{define "content"}}<form></form>{{end}}`)}

	rec = httptest.NewRecorder()
	if err := e.Render(rec, http.StatusOK, "login.html", nil); err != nil {
		t.Fatalf("re-render: %v", err)
	}
	if !strings.Contains(rec.Body.String(), "Edited") {
		t.Errorf("Reload did not pick up the edit: %s", rec.Body.String())
	}
}

// TestRender_EscapesInterpolatedData confirms html/template's contextual
// escaping survives the layout composition. It is the reason for using
// html/template over text/template, and worth a test that would fail loudly if
// anyone ever "fixed" a rendering problem by switching packages.
func TestRender_EscapesInterpolatedData(t *testing.T) {
	e := newEngine(t, templates())

	rec := httptest.NewRecorder()
	err := e.Render(rec, http.StatusOK, "dashboard.html", map[string]any{
		"User":    "<script>alert(1)</script>",
		"Heading": "ok",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(rec.Body.String(), "<script>alert(1)</script>") {
		t.Error("user data was interpolated without escaping")
	}
	if !strings.Contains(rec.Body.String(), "&lt;script&gt;") {
		t.Errorf("expected escaped output, got: %s", rec.Body.String())
	}
}

func TestFuncs_AreAvailableToEveryPage(t *testing.T) {
	fsys := fstest.MapFS{
		"tpl/base.html": {Data: []byte(`{{define "base"}}{{template "content" .}}{{end}}`)},
		"tpl/a.html":    {Data: []byte(`{{define "content"}}{{shout "hi"}}{{end}}`)},
		"tpl/b.html":    {Data: []byte(`{{define "content"}}{{shout "yo"}}{{end}}`)},
	}

	e, err := view.New(view.Config{
		FS:      fsys,
		Dir:     "tpl",
		Layouts: []string{"base.html"},
		Funcs:   template.FuncMap{"shout": strings.ToUpper},
	})
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}

	rec := httptest.NewRecorder()
	if err := e.Render(rec, http.StatusOK, "b.html", nil); err != nil {
		t.Fatalf("render: %v", err)
	}
	if rec.Body.String() != "YO" {
		t.Errorf("got %q, want YO", rec.Body.String())
	}
}
