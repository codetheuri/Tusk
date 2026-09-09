package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/v2/pkg/session"
	"github.com/codetheuri/tusk/v2/pkg/testdb"
)

// These run against real PostgreSQL. Sessions are a database row whose whole
// purpose is that deleting it ends access, so the assertions worth making are
// about what a second request finds in the table — not about a fake.

const loginPath = "/console/login"

func fixture(t *testing.T, opts session.Options) (*session.Manager, *session.GormStore, *gorm.DB) {
	t.Helper()

	db := testdb.Connect(t)
	store := session.NewGormStore(db)

	if opts.Kind == "" {
		opts.Kind = "console"
	}
	mgr, err := session.New(store, opts)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return mgr, store, db
}

// login performs a Start and returns the cookie the browser would hold.
func login(t *testing.T, mgr *session.Manager, subject uuid.UUID) *http.Cookie {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/console/login", nil)
	req.RemoteAddr = "203.0.113.7:51234"
	req.Header.Set("User-Agent", "test-agent")

	if _, err := mgr.Start(rec, req, subject); err != nil {
		t.Fatalf("start: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	return cookies[0]
}

// visit sends a request carrying the cookie through Middleware, and reports what
// the handler saw.
func visit(mgr *session.Manager, cookie *http.Cookie) (*httptest.ResponseRecorder, *session.Session, bool) {
	var (
		got     *session.Session
		present bool
	)
	h := mgr.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, present = session.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/console/dashboard", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, got, present
}

func TestStart_IssuesAnUsableSession(t *testing.T) {
	mgr, _, _ := fixture(t, session.Options{})
	subject := uuid.New()

	cookie := login(t, mgr, subject)

	if !cookie.HttpOnly {
		t.Error("cookie is not HttpOnly, so JavaScript can read it")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite is %v, want Lax", cookie.SameSite)
	}
	if len(cookie.Value) != 64 {
		t.Errorf("token is %d characters, want 64 (256 bits, hex)", len(cookie.Value))
	}

	_, sess, ok := visit(mgr, cookie)
	if !ok {
		t.Fatal("the session was not loaded on the next request")
	}
	if sess.SubjectID != subject {
		t.Errorf("session belongs to %s, want %s", sess.SubjectID, subject)
	}
	if sess.IP != "203.0.113.7" {
		t.Errorf("recorded IP %q", sess.IP)
	}
}

// TestStart_StoresOnlyTheDigest is the property that makes a leaked backup
// survivable.
func TestStart_StoresOnlyTheDigest(t *testing.T) {
	mgr, _, db := fixture(t, session.Options{})

	cookie := login(t, mgr, uuid.New())

	var stored session.Session
	if err := db.First(&stored).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored.TokenHash == cookie.Value {
		t.Fatal("the raw token was stored; a database dump would hand over live sessions")
	}

	var matches int64
	db.Model(&session.Session{}).Where("token_hash = ?", cookie.Value).Count(&matches)
	if matches != 0 {
		t.Error("the raw token is present in the table")
	}
}

func TestMiddleware_IgnoresAnUnknownTokenAndClearsTheCookie(t *testing.T) {
	mgr, _, _ := fixture(t, session.Options{})

	rec, _, ok := visit(mgr, &http.Cookie{Name: "tusk_session", Value: "not-a-real-token"})

	if ok {
		t.Error("an unknown token produced a session")
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Errorf("the dead cookie was not cleared: %#v", cookies)
	}
}

// TestMiddleware_RejectsAnExpiredSession checks the absolute lifetime. The row
// is filtered out by the query, not checked afterwards in Go.
func TestMiddleware_RejectsAnExpiredSession(t *testing.T) {
	mgr, store, db := fixture(t, session.Options{})

	cookie := login(t, mgr, uuid.New())

	// Age it past its expiry.
	if err := db.Model(&session.Session{}).
		Where("1 = 1").
		Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("age the session: %v", err)
	}

	if _, _, ok := visit(mgr, cookie); ok {
		t.Error("an expired session was accepted")
	}

	// And Find agrees, so nothing downstream can resurrect it.
	if _, err := store.Find(context.Background(), "console", cookie.Value, time.Now()); err == nil {
		t.Error("Find returned an expired session")
	}
}

func TestMiddleware_RejectsAnIdleSession(t *testing.T) {
	mgr, _, db := fixture(t, session.Options{IdleTimeout: time.Hour})

	cookie := login(t, mgr, uuid.New())

	if err := db.Model(&session.Session{}).
		Where("1 = 1").
		Update("last_seen_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatalf("age the session: %v", err)
	}

	if _, _, ok := visit(mgr, cookie); ok {
		t.Error("a session idle beyond the timeout was accepted")
	}

	var remaining int64
	db.Model(&session.Session{}).Count(&remaining)
	if remaining != 0 {
		t.Error("the idle session was left in the table")
	}
}

// TestKind_SeparatesAudiences: two managers over one table must not see each
// other's sessions, or an operator cookie would authenticate an ordinary user
// route and vice versa.
func TestKind_SeparatesAudiences(t *testing.T) {
	db := testdb.Connect(t)
	store := session.NewGormStore(db)

	console, err := session.New(store, session.Options{Kind: "console", CookieName: "console_session"})
	if err != nil {
		t.Fatalf("console manager: %v", err)
	}
	portal, err := session.New(store, session.Options{Kind: "portal", CookieName: "portal_session"})
	if err != nil {
		t.Fatalf("portal manager: %v", err)
	}

	cookie := login(t, console, uuid.New())

	if _, _, ok := visit(console, cookie); !ok {
		t.Fatal("the console session did not load for its own manager")
	}

	// Same token value, presented under the portal's cookie name.
	if _, _, ok := visit(portal, &http.Cookie{Name: "portal_session", Value: cookie.Value}); ok {
		t.Error("a console token authenticated a portal session")
	}
}

// TestRotate_InvalidatesTheOldToken covers privilege change. The old cookie must
// stop working the moment the new one is issued.
func TestRotate_InvalidatesTheOldToken(t *testing.T) {
	mgr, _, _ := fixture(t, session.Options{})
	subject := uuid.New()

	old := login(t, mgr, subject)

	var rotated *http.Cookie
	h := mgr.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := mgr.Rotate(w, r); err != nil {
			t.Errorf("rotate: %v", err)
		}
	}))
	req := httptest.NewRequest(http.MethodPost, "/console/elevate", nil)
	req.AddCookie(old)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.Name == "tusk_session" && c.Value != "" {
			rotated = c
		}
	}
	if rotated == nil {
		t.Fatal("rotation issued no new cookie")
	}
	if rotated.Value == old.Value {
		t.Fatal("rotation reused the same token")
	}

	if _, _, ok := visit(mgr, old); ok {
		t.Error("the pre-rotation token still works")
	}
	_, sess, ok := visit(mgr, rotated)
	if !ok {
		t.Fatal("the rotated token does not work")
	}
	if sess.SubjectID != subject {
		t.Errorf("rotation changed the subject to %s", sess.SubjectID)
	}
}

// TestRotate_DoesNotExtendTheAbsoluteLifetime: rotation is a security measure,
// not a way to hold a session open indefinitely.
func TestRotate_DoesNotExtendTheAbsoluteLifetime(t *testing.T) {
	mgr, _, db := fixture(t, session.Options{})

	old := login(t, mgr, uuid.New())

	var before session.Session
	if err := db.First(&before).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}

	h := mgr.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := mgr.Rotate(w, r); err != nil {
			t.Errorf("rotate: %v", err)
		}
	}))
	req := httptest.NewRequest(http.MethodPost, "/console/elevate", nil)
	req.AddCookie(old)
	h.ServeHTTP(httptest.NewRecorder(), req)

	var after session.Session
	if err := db.First(&after).Error; err != nil {
		t.Fatalf("read back after rotation: %v", err)
	}
	if !after.ExpiresAt.Equal(before.ExpiresAt) {
		t.Errorf("expiry moved from %v to %v", before.ExpiresAt, after.ExpiresAt)
	}
}

func TestDestroy_EndsTheSession(t *testing.T) {
	mgr, _, db := fixture(t, session.Options{})

	cookie := login(t, mgr, uuid.New())

	h := mgr.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := mgr.Destroy(w, r); err != nil {
			t.Errorf("destroy: %v", err)
		}
	}))
	req := httptest.NewRequest(http.MethodPost, "/console/logout", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(httptest.NewRecorder(), req)

	var remaining int64
	db.Model(&session.Session{}).Count(&remaining)
	if remaining != 0 {
		t.Errorf("%d sessions left in the table after logout", remaining)
	}
	if _, _, ok := visit(mgr, cookie); ok {
		t.Error("the token still works after logout")
	}
}

// TestDestroyAll_SignsOutEveryDevice is what a password reset has to do.
func TestDestroyAll_SignsOutEveryDevice(t *testing.T) {
	mgr, _, db := fixture(t, session.Options{})
	subject, other := uuid.New(), uuid.New()

	phone := login(t, mgr, subject)
	laptop := login(t, mgr, subject)
	elsewhere := login(t, mgr, other)

	if err := mgr.DestroyAll(context.Background(), subject); err != nil {
		t.Fatalf("destroy all: %v", err)
	}

	for name, c := range map[string]*http.Cookie{"phone": phone, "laptop": laptop} {
		if _, _, ok := visit(mgr, c); ok {
			t.Errorf("the %s session survived", name)
		}
	}
	if _, _, ok := visit(mgr, elsewhere); !ok {
		t.Error("another subject's session was destroyed too")
	}

	var remaining int64
	db.Model(&session.Session{}).Count(&remaining)
	if remaining != 1 {
		t.Errorf("%d sessions remain, want 1", remaining)
	}
}

// TestRequire_RedirectsWithoutASession, and lets one through with.
func TestRequire_RedirectsWithoutASession(t *testing.T) {
	mgr, _, _ := fixture(t, session.Options{})

	reached := false
	h := mgr.Middleware()(mgr.Require(loginPath)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	})))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/console/dashboard", nil))

	if reached {
		t.Error("the protected handler ran without a session")
	}
	if rec.Code != http.StatusSeeOther {
		t.Errorf("status %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != loginPath {
		t.Errorf("redirected to %q, want %q", loc, loginPath)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/console/dashboard", nil)
	req.AddCookie(login(t, mgr, uuid.New()))
	h.ServeHTTP(rec, req)

	if !reached {
		t.Error("a valid session was refused")
	}
}

func TestDeleteExpired_SweepsOnlyDeadRows(t *testing.T) {
	mgr, _, db := fixture(t, session.Options{})

	liveSubject, deadSubject := uuid.New(), uuid.New()
	live := login(t, mgr, liveSubject)
	dead := login(t, mgr, deadSubject)

	// Selected by subject rather than by ordering: two sessions created in the
	// same microsecond would make an ORDER BY created_at tie-break arbitrary,
	// and a test that usually picks the right row is worse than no test.
	if err := db.Model(&session.Session{}).
		Where("subject_id = ?", deadSubject).
		Update("expires_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatalf("age one session: %v", err)
	}

	n, err := mgr.DeleteExpired(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Errorf("swept %d rows, want 1", n)
	}
	if _, _, ok := visit(mgr, live); !ok {
		t.Error("the live session was swept")
	}
	if _, _, ok := visit(mgr, dead); ok {
		t.Error("the expired session survived")
	}
}

func TestNew_RequiresAKind(t *testing.T) {
	db := testdb.Connect(t)
	if _, err := session.New(session.NewGormStore(db), session.Options{}); err == nil {
		t.Error("expected New to require a Kind")
	}
}
