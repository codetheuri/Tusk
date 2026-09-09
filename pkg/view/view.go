// Package view renders server-rendered HTML pages alongside a Huma API.
//
// Tusk is API-first, but admin panels are a recurring need, and the usual
// answer — ad-hoc html/template calls scattered through handlers — repeats the
// same three mistakes in every project:
//
//  1. Rendering straight to the ResponseWriter. Go templates write output as
//     they execute, so an error halfway through a page has already sent a 200
//     and half the HTML. The client receives a truncated page it believes is
//     complete, and the error handler can no longer change the status code.
//     Engine renders into a buffer and writes nothing until it succeeds.
//
//  2. Parsing on every request, forever. Convenient in development, wasteful in
//     production, and the "cache this later" comment never gets revisited.
//     Reload makes it a configuration choice with a correct default.
//
//  3. Discovering a broken template when a user visits the page. Templates are
//     parsed once at construction, so a malformed page stops the process from
//     starting rather than surfacing as a 500 in production.
//
// # Layouts
//
// Go's html/template has no inheritance; it has named blocks that can reference
// each other within a set. Layout inheritance is therefore expressed by parsing
// the shared files together with one page file, and executing whichever block is
// the outermost one:
//
//	base.html    {{define "base"}}<html>… {{template "content" .}} …</html>{{end}}
//	layout.html  {{define "sidebar"}}…{{end}}
//	dashboard.html
//	             {{define "title"}}Dashboard{{end}}
//	             {{define "content"}}…{{template "sidebar" .}}…{{end}}
//
// Each page gets its own template set, so two pages may define "content"
// differently without colliding.
//
// # Files
//
// The engine reads from an fs.FS, so the same code serves an embed.FS in
// production and os.DirFS in development, where Reload picks up edits without a
// restart.
package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"sync"
)

// Config describes an Engine.
type Config struct {
	// FS holds the templates. Required.
	FS fs.FS

	// Dir is the directory within FS to read, for example "templates/console".
	// Empty means the root of FS.
	Dir string

	// Layouts are the shared files parsed into every page set, relative to Dir.
	// Order matters only in that later definitions of the same block win.
	Layouts []string

	// Entry is the defined template executed for a render, typically the one
	// wrapping <html>. Defaults to "base".
	Entry string

	// Funcs are made available to every template. Register them here rather than
	// per-page; a function missing from one set is a parse error in that set
	// alone, which is a confusing way to find out.
	Funcs template.FuncMap

	// Reload re-reads and re-parses on every render. Development only: it makes
	// template edits visible without a restart, at the cost of doing the work
	// every time. Drive it from the environment, never hard-code it true.
	Reload bool
}

// Engine renders pages from a set of templates.
type Engine struct {
	cfg   Config
	entry string

	mu    sync.RWMutex
	cache map[string]*template.Template
}

// New parses every page in the configured directory and returns an Engine.
//
// Parsing everything up front is the point: a template that does not compile
// should stop deployment, not wait for a user to open that page.
func New(cfg Config) (*Engine, error) {
	if cfg.FS == nil {
		return nil, fmt.Errorf("view: Config.FS is required")
	}
	entry := cfg.Entry
	if entry == "" {
		entry = "base"
	}

	e := &Engine{cfg: cfg, entry: entry, cache: map[string]*template.Template{}}

	pages, err := e.pages()
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("view: no page templates found in %q", e.dir())
	}

	for _, page := range pages {
		tmpl, err := e.parse(page)
		if err != nil {
			return nil, err
		}
		if !cfg.Reload {
			e.cache[page] = tmpl
		}
	}
	return e, nil
}

// Render writes a page to the response.
//
// The status is written only once the page has rendered successfully, so a
// template error can still become a 500 instead of a half-sent 200.
func (e *Engine) Render(w http.ResponseWriter, status int, page string, data any) error {
	var buf bytes.Buffer
	if err := e.RenderTo(&buf, page, data); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}

// RenderTo renders to an arbitrary writer, for tests and for HTML mail.
//
// Callers writing to a network connection should render to a buffer first; this
// method makes no such guarantee of its own.
func (e *Engine) RenderTo(w io.Writer, page string, data any) error {
	tmpl, err := e.lookup(page)
	if err != nil {
		return err
	}
	if err := tmpl.ExecuteTemplate(w, e.entry, data); err != nil {
		return fmt.Errorf("view: rendering %q: %w", page, err)
	}
	return nil
}

// lookup returns the parsed set for a page, honouring Reload.
func (e *Engine) lookup(page string) (*template.Template, error) {
	if e.cfg.Reload {
		return e.parse(page)
	}

	e.mu.RLock()
	tmpl, ok := e.cache[page]
	e.mu.RUnlock()
	if !ok {
		// Not a missing file — New parsed every page that exists, so reaching
		// here means the caller named one that does not.
		return nil, fmt.Errorf("view: no such page %q", page)
	}
	return tmpl, nil
}

// parse builds one page's template set: the layouts plus the page itself.
func (e *Engine) parse(page string) (*template.Template, error) {
	files := make([]string, 0, len(e.cfg.Layouts)+1)
	for _, l := range e.cfg.Layouts {
		files = append(files, e.join(l))
	}
	files = append(files, e.join(page))

	// Named for the page so template errors identify which one failed.
	tmpl, err := template.New(page).Funcs(e.cfg.Funcs).ParseFS(e.cfg.FS, files...)
	if err != nil {
		return nil, fmt.Errorf("view: parsing %q: %w", page, err)
	}
	if tmpl.Lookup(e.entry) == nil {
		return nil, fmt.Errorf("view: %q does not define template %q; "+
			"either the page or one of its layouts must define it", page, e.entry)
	}
	return tmpl, nil
}

// pages lists every .html file in Dir that is not a layout.
func (e *Engine) pages() ([]string, error) {
	entries, err := fs.ReadDir(e.cfg.FS, e.dir())
	if err != nil {
		return nil, fmt.Errorf("view: reading %q: %w", e.dir(), err)
	}

	layouts := make(map[string]bool, len(e.cfg.Layouts))
	for _, l := range e.cfg.Layouts {
		layouts[l] = true
	}

	var pages []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || path.Ext(name) != ".html" || layouts[name] {
			continue
		}
		pages = append(pages, name)
	}
	sort.Strings(pages) // deterministic startup errors
	return pages, nil
}

func (e *Engine) dir() string {
	if e.cfg.Dir == "" {
		return "."
	}
	return e.cfg.Dir
}

func (e *Engine) join(name string) string {
	if e.cfg.Dir == "" {
		return name
	}
	return path.Join(e.cfg.Dir, name)
}
