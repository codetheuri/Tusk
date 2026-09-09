package auth

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/codetheuri/tusk/config"
	"github.com/codetheuri/tusk/pkg/authz"
	"github.com/codetheuri/tusk/pkg/id"
	"github.com/codetheuri/tusk/pkg/logger"
	"github.com/codetheuri/tusk/pkg/testdb"
)

// newTestService wires the real repository and service against a real database.
// Nothing is mocked: the point of these tests is the SQL, which a mock would
// replace with the very assumption under test.
func newTestService(t *testing.T) (*Service, *Repository, *gorm.DB) {
	t.Helper()

	db := testdb.Connect(t)
	if db == nil {
		return nil, nil, nil // Connect already skipped the test
	}

	cfg := &config.Config{
		JWTSecret:      "integration-test-secret-key-long-enough",
		AccessTokenTTL: time.Hour,
	}
	repo := NewRepository(db, logger.NewTextLogger("error"))
	return NewService(repo, cfg), repo, db
}

func mustRegister(t *testing.T, svc *Service, username, email, password string) *User {
	t.Helper()
	user, err := svc.Register(context.Background(), &RegisterRequest{
		Username:        username,
		Email:           email,
		Password:        password,
		PasswordConfirm: password,
		FirstName:       "Test",
		LastName:        "User",
	})
	if err != nil {
		t.Fatalf("register %q failed: %v", username, err)
	}
	return user
}

func TestRegisterAndLogin(t *testing.T) {
	svc, _, _ := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	user := mustRegister(t, svc, "alice", "alice@example.com", "correct-horse-battery")
	if id.IsZero(user.ID) {
		t.Fatal("expected an identifier to be assigned")
	}

	// The stored password must be a hash, never the plaintext.
	if user.Password == "correct-horse-battery" {
		t.Fatal("password was stored in plaintext")
	}

	tokens, err := svc.Login(ctx, &LoginRequest{Login: "alice", Password: "correct-horse-battery"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Error("expected both an access and a refresh token")
	}

	if _, err := svc.Login(ctx, &LoginRequest{Login: "alice", Password: "wrong"}); err == nil {
		t.Error("expected login with a wrong password to fail")
	}
}

// Login accepts a username, an email, or a phone in one field. That flexibility
// is a query concern, so it can only be verified against a real database.
func TestLogin_AcceptsUsernameOrEmail(t *testing.T) {
	svc, _, _ := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	mustRegister(t, svc, "bob", "bob@example.com", "a-good-password")

	for _, login := range []string{"bob", "bob@example.com"} {
		if _, err := svc.Login(ctx, &LoginRequest{Login: login, Password: "a-good-password"}); err != nil {
			t.Errorf("login with %q failed: %v", login, err)
		}
	}
}

// Five failed attempts must lock the account, and the lock must survive a
// subsequent attempt with the *correct* password — otherwise it is not a lock.
func TestLogin_LocksAccountAfterFailedAttempts(t *testing.T) {
	svc, repo, _ := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	user := mustRegister(t, svc, "carol", "carol@example.com", "the-real-password")

	for i := 0; i < 5; i++ {
		if _, err := svc.Login(ctx, &LoginRequest{Login: "carol", Password: "wrong"}); err == nil {
			t.Fatalf("attempt %d: expected failure with a wrong password", i+1)
		}
	}

	stored, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if stored.FailedLoginAttempts < 5 {
		t.Errorf("FailedLoginAttempts = %d, want >= 5", stored.FailedLoginAttempts)
	}
	if stored.LockedUntil == nil || !stored.LockedUntil.After(time.Now()) {
		t.Fatalf("expected the account to be locked, LockedUntil = %v", stored.LockedUntil)
	}

	if _, err := svc.Login(ctx, &LoginRequest{Login: "carol", Password: "the-real-password"}); err == nil {
		t.Error("expected a locked account to reject even the correct password")
	}
}

func TestLogin_ResetsFailureCountOnSuccess(t *testing.T) {
	svc, repo, _ := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	user := mustRegister(t, svc, "dave", "dave@example.com", "the-real-password")

	for i := 0; i < 3; i++ {
		_, _ = svc.Login(ctx, &LoginRequest{Login: "dave", Password: "wrong"})
	}
	if _, err := svc.Login(ctx, &LoginRequest{Login: "dave", Password: "the-real-password"}); err != nil {
		t.Fatalf("login with the correct password failed: %v", err)
	}

	stored, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if stored.FailedLoginAttempts != 0 {
		t.Errorf("FailedLoginAttempts = %d after a successful login, want 0", stored.FailedLoginAttempts)
	}
	if stored.LastLoginAt == nil {
		t.Error("expected LastLoginAt to be recorded")
	}
}

// Refresh tokens rotate: presenting one must invalidate it. Replaying a rotated
// token is the signal of a stolen credential, so it must not succeed.
func TestRefreshToken_RotatesAndRevokes(t *testing.T) {
	svc, _, _ := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	mustRegister(t, svc, "erin", "erin@example.com", "a-good-password")
	tokens, err := svc.Login(ctx, &LoginRequest{Login: "erin", Password: "a-good-password"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	rotated, err := svc.RefreshToken(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if rotated.RefreshToken == tokens.RefreshToken {
		t.Error("expected a new refresh token, got the same one back")
	}

	if _, err := svc.RefreshToken(ctx, tokens.RefreshToken); err == nil {
		t.Error("expected the rotated-away refresh token to be rejected on reuse")
	}
}

func TestLogout_RevokesRefreshToken(t *testing.T) {
	svc, _, _ := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	mustRegister(t, svc, "frank", "frank@example.com", "a-good-password")
	tokens, err := svc.Login(ctx, &LoginRequest{Login: "frank", Password: "a-good-password"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := svc.Logout(ctx, tokens.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	if _, err := svc.RefreshToken(ctx, tokens.RefreshToken); err == nil {
		t.Error("expected a revoked refresh token to be unusable")
	}
}

// The unique constraints live in the migration, so only a real database proves
// they are actually enforced.
func TestRegister_RejectsDuplicates(t *testing.T) {
	svc, _, _ := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	mustRegister(t, svc, "grace", "grace@example.com", "a-good-password")

	_, err := svc.Register(ctx, &RegisterRequest{
		Username: "grace", Email: "different@example.com",
		Password: "a-good-password", PasswordConfirm: "a-good-password",
	})
	if err == nil {
		t.Error("expected a duplicate username to be rejected")
	}

	_, err = svc.Register(ctx, &RegisterRequest{
		Username: "different", Email: "grace@example.com",
		Password: "a-good-password", PasswordConfirm: "a-good-password",
	})
	if err == nil {
		t.Error("expected a duplicate email to be rejected")
	}
}

// The RBAC check walks users → user_roles → role_permissions. That is a
// three-table join built at runtime, which no unit test can exercise.
func TestRBAC_PermissionEvaluation(t *testing.T) {
	svc, _, db := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	user := mustRegister(t, svc, "heidi", "heidi@example.com", "a-good-password")

	role, err := svc.CreateRole(ctx, &CreateRoleRequest{Name: "editor", Description: "May edit users"})
	if err != nil {
		t.Fatalf("create role failed: %v", err)
	}

	evaluator := authz.NewEvaluator(db)
	subject := authz.Subject{UserID: user.ID}

	// Before anything is granted.
	allowed, err := evaluator.IsAuthorized(ctx, subject, authz.RequirePermissionPolicy{Permission: PermUsersUpdate})
	if err != nil {
		t.Fatalf("authorization check failed: %v", err)
	}
	if allowed {
		t.Error("a user with no roles must not hold any permission")
	}

	if err := svc.AddRolePermission(ctx, role.ID, PermUsersUpdate); err != nil {
		t.Fatalf("attaching permission to role failed: %v", err)
	}

	// The role holds the permission, but the user does not hold the role yet.
	allowed, _ = evaluator.IsAuthorized(ctx, subject, authz.RequirePermissionPolicy{Permission: PermUsersUpdate})
	if allowed {
		t.Error("permission leaked to a user who was never assigned the role")
	}

	if err := svc.AssignUserRole(ctx, user.ID, role.ID); err != nil {
		t.Fatalf("assigning role to user failed: %v", err)
	}

	allowed, err = evaluator.IsAuthorized(ctx, subject, authz.RequirePermissionPolicy{Permission: PermUsersUpdate})
	if err != nil {
		t.Fatalf("authorization check failed: %v", err)
	}
	if !allowed {
		t.Error("expected the permission to be granted through the assigned role")
	}

	// A permission the role was never given must still be refused.
	allowed, _ = evaluator.IsAuthorized(ctx, subject, authz.RequirePermissionPolicy{Permission: PermUsersDelete})
	if allowed {
		t.Error("expected an ungranted permission to be refused")
	}

	// Revoking the role revokes the permission with it.
	if err := svc.RemoveUserRole(ctx, user.ID, role.ID); err != nil {
		t.Fatalf("removing role failed: %v", err)
	}
	allowed, _ = evaluator.IsAuthorized(ctx, subject, authz.RequirePermissionPolicy{Permission: PermUsersUpdate})
	if allowed {
		t.Error("expected permission to be revoked along with the role")
	}
}

// A super user bypasses permission checks entirely — worth pinning down, because
// this flag now travels in the JWT rather than being read from the database.
func TestRBAC_SuperUserBypassesChecks(t *testing.T) {
	svc, _, db := newTestService(t)
	if svc == nil {
		return
	}
	ctx := context.Background()

	user := mustRegister(t, svc, "ivan", "ivan@example.com", "a-good-password")

	evaluator := authz.NewEvaluator(db)
	superSubject := authz.Subject{UserID: user.ID, IsSuperUser: true}

	allowed, err := evaluator.IsAuthorized(ctx, superSubject, authz.RequirePermissionPolicy{Permission: PermUsersDelete})
	if err != nil {
		t.Fatalf("authorization check failed: %v", err)
	}
	if !allowed {
		t.Error("expected a super user to bypass permission checks")
	}
}

// The synchroniser must be idempotent: running it twice is routine (every deploy)
// and must not duplicate rows or error.
func TestPermissionSync_IsIdempotent(t *testing.T) {
	db := testdb.Connect(t)
	if db == nil {
		return
	}
	ctx := context.Background()

	sync := authz.NewSynchronizer(db, authz.DefaultRegistry())

	first, err := sync.Sync(ctx, false)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}
	if first.Inserted == 0 {
		t.Error("expected the first sync to insert the code-declared permissions")
	}

	second, err := sync.Sync(ctx, false)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}
	if second.Inserted != 0 {
		t.Errorf("second sync inserted %d permissions, want 0 — sync is not idempotent", second.Inserted)
	}

	var count int64
	db.Table("permissions").Count(&count)
	if int(count) != len(authz.DefaultRegistry().All()) {
		t.Errorf("permissions table holds %d rows, want %d", count, len(authz.DefaultRegistry().All()))
	}
}

// A client-generated identifier must survive being written.
//
// This is the property that makes offline-first clients possible: a device
// creates a record while disconnected, mints its own ID, and may already have
// created other records referencing it. If BeforeCreate overwrote that ID on
// arrival, those references would silently point at nothing.
func TestCreate_PreservesClientSuppliedID(t *testing.T) {
	_, repo, _ := newTestService(t)
	if repo == nil {
		return
	}
	ctx := context.Background()

	// The identifier a disconnected client would have generated for itself.
	clientID := id.New()

	user := &User{
		ID:       clientID,
		Username: "offline-client",
		Email:    "offline@example.com",
		Password: "already-hashed-by-the-service",
		IsActive: true,
	}
	if err := repo.CreateUserWithProfile(ctx, user, &UserProfile{}); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if user.ID != clientID {
		t.Fatalf("client-supplied ID was overwritten: got %s, want %s", user.ID, clientID)
	}

	stored, err := repo.FindByID(ctx, clientID)
	if err != nil {
		t.Fatalf("could not read back the row by its client-supplied ID: %v", err)
	}
	if stored.Username != "offline-client" {
		t.Errorf("read back the wrong row: %s", stored.Username)
	}
}

// The complementary case: with no ID supplied, one must be generated.
func TestCreate_GeneratesIDWhenAbsent(t *testing.T) {
	_, repo, _ := newTestService(t)
	if repo == nil {
		return
	}
	ctx := context.Background()

	user := &User{
		Username: "no-id-supplied",
		Email:    "no-id@example.com",
		Password: "already-hashed-by-the-service",
		IsActive: true,
	}
	if err := repo.CreateUserWithProfile(ctx, user, &UserProfile{}); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if id.IsZero(user.ID) {
		t.Fatal("expected an identifier to be generated")
	}
	if user.ID.Version() != 7 {
		t.Errorf("generated ID is UUID version %d, want 7", user.ID.Version())
	}
}
