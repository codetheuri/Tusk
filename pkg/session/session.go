// Package session provides database-backed cookie sessions for server-rendered
// pages, alongside the JWT authentication Tusk uses for its API.
//
// # Why not just use the JWT
//
// A browser session and an API token want opposite things. A JWT is stateless
// and therefore cannot be revoked before it expires — fine for a mobile client
// holding a short-lived access token, wrong for an operator console where
// "sign out everywhere" and "revoke that person's access now" are requirements.
// Sessions are a database row precisely so that deleting the row ends access.
//
// # What the browser holds
//
// A 256-bit random token, in an HttpOnly cookie. The database stores only its
// SHA-256 digest, so a leaked backup does not hand over live sessions.
//
// SHA-256 rather than bcrypt, deliberately, and for the opposite reason to
// passwords: a password is low-entropy and guessable, so hashing it must be
// slow. This token is 256 bits of randomness — unguessable regardless of hash
// speed — and it is verified on every single request, so a deliberately slow
// hash would be a denial-of-service surface rather than a protection.
//
// # Wiring
//
//	mgr, _ := session.New(store, session.Options{
//	    Kind: "console", CookieName: "console_session",
//	    Path: "/console", Secure: cfg.IsProduction(),
//	})
//	r.Route("/console", func(r chi.Router) {
//	    r.Use(mgr.Middleware())        // loads a session if there is one
//	    r.Get("/login", showLogin)     // reads session.FromContext to skip the form
//	    r.Group(func(r chi.Router) {
//	        r.Use(mgr.Require("/console/login"))
//	        r.Get("/dashboard", dashboard)
//	    })
//	})
//
// Middleware loads without requiring, and Require refuses. Keeping them separate
// is what stops the classic redirect loop: a login page that redirects on the
// mere presence of a cookie will bounce a visitor holding a stale one between
// /login and /dashboard forever. Here the login page asks whether a *valid*
// session exists, and the middleware has already cleared the cookie if not.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Options configures a Manager.
type Options struct {
	// Kind names the audience these sessions belong to, for example "console".
	// Required: it is what keeps two session audiences in one database apart.
	Kind string

	// CookieName defaults to "tusk_session". Give each Kind its own name.
	CookieName string

	// Path scopes the cookie. Defaults to "/". Setting it to the console's
	// mount point keeps the cookie off every other request.
	Path string

	// Domain is normally left empty, which scopes the cookie to the exact host
	// that set it — the safer default.
	Domain string

	// Secure restricts the cookie to HTTPS. It must be true in production; drive
	// it from configuration rather than hard-coding either value.
	Secure bool

	// SameSite defaults to http.SameSiteLaxMode, which keeps the cookie off
	// cross-site POSTs — the main CSRF vector — while still allowing ordinary
	// inbound links to work.
	SameSite http.SameSite

	// Lifetime is the absolute maximum age of a session. Defaults to 8 hours.
	// It is not extended by activity: a stolen session cannot be kept alive
	// forever by using it.
	Lifetime time.Duration

	// IdleTimeout ends a session that has gone unused. Zero disables it.
	IdleTimeout time.Duration
}

// Manager issues, validates and ends sessions.
type Manager struct {
	store Store
	opts  Options
}

// New validates the options and returns a Manager.
func New(store Store, opts Options) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("session: a Store is required")
	}
	if opts.Kind == "" {
		return nil, fmt.Errorf("session: Options.Kind is required, so that sessions from different audiences cannot be confused")
	}
	if opts.CookieName == "" {
		opts.CookieName = "tusk_session"
	}
	if opts.Path == "" {
		opts.Path = "/"
	}
	if opts.SameSite == 0 {
		opts.SameSite = http.SameSiteLaxMode
	}
	if opts.Lifetime == 0 {
		opts.Lifetime = 8 * time.Hour
	}
	if opts.IdleTimeout < 0 {
		return nil, fmt.Errorf("session: IdleTimeout cannot be negative")
	}
	return &Manager{store: store, opts: opts}, nil
}

// contextKey is unexported so no other package can write or overwrite the value.
type contextKey struct{}

// FromContext returns the session loaded by Middleware, if any.
func FromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(contextKey{}).(*Session)
	return s, ok
}

// Start creates a session for a subject and sets the cookie.
//
// Call it only after credentials have been verified. It always mints a fresh
// token rather than adopting any the request arrived with, which is what makes
// session fixation impossible: an attacker who plants a cookie value in a
// victim's browser does not learn the one issued at login.
func (m *Manager) Start(w http.ResponseWriter, r *http.Request, subjectID uuid.UUID) (*Session, error) {
	token, hash, err := newToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sess := &Session{
		TokenHash:   hash,
		SubjectKind: m.opts.Kind,
		SubjectID:   subjectID,
		IP:          clientIP(r),
		UserAgent:   truncate(r.UserAgent(), 255),
		CreatedAt:   now,
		LastSeenAt:  now,
		ExpiresAt:   now.Add(m.opts.Lifetime),
	}
	if err := m.store.Create(r.Context(), sess); err != nil {
		return nil, fmt.Errorf("session: creating: %w", err)
	}

	m.setCookie(w, token, sess.ExpiresAt)
	return sess, nil
}

// Rotate issues a new token for the current session and invalidates the old one.
//
// Call it whenever what the session is allowed to do changes — an elevation to
// administrator, a password change, a switch of acting account. If a token had
// leaked before that point, rotation is what stops it inheriting the new
// privileges.
func (m *Manager) Rotate(w http.ResponseWriter, r *http.Request) (*Session, error) {
	current, ok := FromContext(r.Context())
	if !ok {
		return nil, ErrNotFound
	}

	token, hash, err := newToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	next := &Session{
		TokenHash:   hash,
		SubjectKind: current.SubjectKind,
		SubjectID:   current.SubjectID,
		IP:          clientIP(r),
		UserAgent:   truncate(r.UserAgent(), 255),
		CreatedAt:   now,
		LastSeenAt:  now,

		// The absolute expiry is inherited, not restarted. Rotation is a
		// security measure; letting it extend the lifetime would turn it into a
		// way to hold a session open indefinitely.
		ExpiresAt: current.ExpiresAt,
	}
	if err := m.store.Create(r.Context(), next); err != nil {
		return nil, fmt.Errorf("session: rotating: %w", err)
	}
	if err := m.store.Delete(r.Context(), current.ID); err != nil {
		return nil, fmt.Errorf("session: removing the rotated session: %w", err)
	}

	m.setCookie(w, token, next.ExpiresAt)
	return next, nil
}

// Destroy ends the current session and clears the cookie.
func (m *Manager) Destroy(w http.ResponseWriter, r *http.Request) error {
	defer m.clearCookie(w)

	if current, ok := FromContext(r.Context()); ok {
		if err := m.store.Delete(r.Context(), current.ID); err != nil {
			return fmt.Errorf("session: destroying: %w", err)
		}
		return nil
	}

	// No session in context: the cookie may still name one, if Middleware was
	// not mounted on this route. Delete by token so logout is not silently a
	// no-op.
	cookie, err := r.Cookie(m.opts.CookieName)
	if err != nil {
		return nil
	}
	sess, err := m.store.Find(r.Context(), m.opts.Kind, hashToken(cookie.Value), time.Now())
	if err != nil {
		return nil // already gone or expired; the cookie is cleared regardless
	}
	return m.store.Delete(r.Context(), sess.ID)
}

// DestroyAll ends every session belonging to a subject.
//
// The right response to a password reset or a compromise report: whoever else
// was holding a token is signed out, not merely prevented from signing in again.
func (m *Manager) DestroyAll(ctx context.Context, subjectID uuid.UUID) error {
	return m.store.DeleteBySubject(ctx, m.opts.Kind, subjectID)
}

// DeleteExpired removes rows whose expiry has passed. Run it periodically;
// nothing else deletes them, and expired rows are still returned by nothing but
// still occupy the table.
func (m *Manager) DeleteExpired(ctx context.Context) (int64, error) {
	return m.store.DeleteExpired(ctx, time.Now())
}

// Middleware loads a valid session into the request context.
//
// It does not reject anything. A request without a session proceeds, which is
// what public pages under the same mount — the login form itself — need.
func (m *Manager) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(m.opts.CookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			now := time.Now()
			sess, err := m.store.Find(r.Context(), m.opts.Kind, hashToken(cookie.Value), now)
			if err != nil {
				// Clear it. Leaving a dead cookie in place means every later
				// request repeats this lookup, and any page that tests for the
				// cookie rather than the session will loop.
				m.clearCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			if m.opts.IdleTimeout > 0 && now.Sub(sess.LastSeenAt) > m.opts.IdleTimeout {
				_ = m.store.Delete(r.Context(), sess.ID)
				m.clearCookie(w)
				next.ServeHTTP(w, r)
				return
			}

			m.touch(r.Context(), sess, now)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, sess)))
		})
	}
}

// Require refuses requests that have no session, redirecting to loginPath.
//
// Mount it inside Middleware, on the routes that need protection.
func (m *Manager) Require(loginPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := FromContext(r.Context()); !ok {
				http.Redirect(w, r, loginPath, http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// touch records activity, but not on every request.
//
// An idle timeout needs a last-seen time, and updating it per request turns
// every page view into a database write. Writing only once the recorded time is
// meaningfully stale keeps the timeout accurate to within a small fraction of
// itself, which is all it needs to be.
func (m *Manager) touch(ctx context.Context, sess *Session, now time.Time) {
	if m.opts.IdleTimeout == 0 {
		return
	}
	if now.Sub(sess.LastSeenAt) < m.opts.IdleTimeout/10 {
		return
	}
	if err := m.store.Touch(ctx, sess.ID, now); err == nil {
		sess.LastSeenAt = now
	}
}

func (m *Manager) setCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.opts.CookieName,
		Value:    token,
		Path:     m.opts.Path,
		Domain:   m.opts.Domain,
		Expires:  expires,
		HttpOnly: true, // unreadable from JavaScript, so XSS cannot exfiltrate it
		Secure:   m.opts.Secure,
		SameSite: m.opts.SameSite,
	})
}

func (m *Manager) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   m.opts.CookieName,
		Value:  "",
		Path:   m.opts.Path, // must match, or the browser keeps the original
		Domain: m.opts.Domain,
		MaxAge: -1,

		// The security attributes have to match too. A clearing cookie that
		// drops Secure is rejected outright by some browsers on HTTPS.
		HttpOnly: true,
		Secure:   m.opts.Secure,
		SameSite: m.opts.SameSite,
	})
}

// newToken returns a fresh session token and the digest to store for it.
func newToken() (token, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("session: generating a token: %w", err)
	}
	token = hex.EncodeToString(buf)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// clientIP records where a session was used from, for the benefit of an
// operator reviewing active sessions.
//
// It is recorded, never trusted: behind a proxy RemoteAddr is the proxy, and
// X-Forwarded-For is caller-supplied and trivially forged. Nothing here makes an
// access decision from it.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return truncate(r.RemoteAddr, 45)
	}
	return truncate(host, 45)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
