package session

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/v2/pkg/id"
)

// Session is one signed-in browser.
//
// The token the browser holds is deliberately absent. Only its SHA-256 digest is
// stored, so a database dump — a backup on a laptop, a leaked replica, an
// over-broad support query — does not hand over live sessions. This mirrors how
// Tusk stores refresh tokens, and is the same reason password hashes are not
// passwords.
type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`

	// SubjectKind separates audiences that share a database. An application with
	// both an operator console and ordinary user sessions gives them different
	// kinds, so a cookie from one can never resolve to a session in the other.
	SubjectKind string `gorm:"size:64;not null;index:idx_sessions_subject,priority:1"`

	// SubjectID identifies who is signed in. There is intentionally no foreign
	// key: the subject may live in any table — Tusk's own users, or an
	// application's separate operators table — and the package has no business
	// deciding which.
	SubjectID uuid.UUID `gorm:"type:uuid;not null;index:idx_sessions_subject,priority:2"`

	IP        string `gorm:"size:45;not null"`
	UserAgent string `gorm:"size:255;not null"`

	CreatedAt  time.Time `gorm:"not null"`
	LastSeenAt time.Time `gorm:"not null"`
	ExpiresAt  time.Time `gorm:"not null;index"`
}

func (Session) TableName() string { return "sessions" }

func (s *Session) BeforeCreate(*gorm.DB) error {
	if id.IsZero(s.ID) {
		s.ID = id.New()
	}
	return nil
}

// ErrNotFound reports that no live session matches. It deliberately does not
// distinguish "no such token" from "expired" — the caller's response is the same
// either way, and telling a client which it was reveals whether a token was ever
// valid.
var ErrNotFound = errors.New("session: not found")

// Store persists sessions.
//
// An interface because sessions are the one part of an admin panel that
// reasonably lives somewhere other than the main database — Redis, most often,
// when session volume outgrows a table. GormStore is the default.
type Store interface {
	Create(ctx context.Context, s *Session) error

	// Find returns the live session for a token digest, or ErrNotFound. It must
	// not return expired sessions.
	Find(ctx context.Context, kind, tokenHash string, now time.Time) (*Session, error)

	Touch(ctx context.Context, sessionID uuid.UUID, at time.Time) error
	Delete(ctx context.Context, sessionID uuid.UUID) error
	DeleteBySubject(ctx context.Context, kind string, subjectID uuid.UUID) error

	// DeleteExpired removes rows past their expiry and reports how many.
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// GormStore stores sessions in the application database.
type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

var _ Store = (*GormStore)(nil)

func (s *GormStore) Create(ctx context.Context, sess *Session) error {
	return s.db.WithContext(ctx).Create(sess).Error
}

func (s *GormStore) Find(ctx context.Context, kind, tokenHash string, now time.Time) (*Session, error) {
	var sess Session

	// Expiry is filtered in the query rather than checked after loading. A row
	// read and then rejected in Go is one refactor away from being read and used.
	err := s.db.WithContext(ctx).
		Where("subject_kind = ? AND token_hash = ? AND expires_at > ?", kind, tokenHash, now).
		First(&sess).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *GormStore) Touch(ctx context.Context, sessionID uuid.UUID, at time.Time) error {
	return s.db.WithContext(ctx).
		Model(&Session{}).
		Where("id = ?", sessionID).
		Update("last_seen_at", at).Error
}

func (s *GormStore) Delete(ctx context.Context, sessionID uuid.UUID) error {
	return s.db.WithContext(ctx).Where("id = ?", sessionID).Delete(&Session{}).Error
}

func (s *GormStore) DeleteBySubject(ctx context.Context, kind string, subjectID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Where("subject_kind = ? AND subject_id = ?", kind, subjectID).
		Delete(&Session{}).Error
}

func (s *GormStore) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := s.db.WithContext(ctx).Where("expires_at <= ?", now).Delete(&Session{})
	return res.RowsAffected, res.Error
}
