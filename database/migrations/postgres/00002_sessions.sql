-- +goose Up

-- Browser sessions for server-rendered pages, separate from the JWT path the API
-- uses. A session is a database row so that deleting the row ends access — the
-- thing a stateless token cannot do, and the reason an operator console needs
-- this table rather than another kind of token.
CREATE TABLE IF NOT EXISTS sessions (
    id           UUID         PRIMARY KEY,

    -- The SHA-256 digest of the token, never the token. A database dump
    -- therefore does not hand over live sessions. 64 hex characters.
    token_hash   VARCHAR(64)  NOT NULL UNIQUE,

    -- Which audience the session belongs to. An application with both an
    -- operator console and ordinary user sessions gives them different kinds, so
    -- a cookie from one can never resolve against the other.
    subject_kind VARCHAR(64)  NOT NULL,

    -- Who is signed in. Deliberately no foreign key: the subject may live in
    -- Tusk's users table or in an application's own operators table, and this
    -- table has no business deciding which.
    subject_id   UUID         NOT NULL,

    -- Recorded for an operator reviewing active sessions. Never used to make an
    -- access decision: behind a proxy both values are supplied by someone else.
    ip           VARCHAR(45)  NOT NULL DEFAULT '',
    user_agent   VARCHAR(255) NOT NULL DEFAULT '',

    created_at   TIMESTAMPTZ  NOT NULL,
    last_seen_at TIMESTAMPTZ  NOT NULL,
    expires_at   TIMESTAMPTZ  NOT NULL
);

-- Covers "every session for this person", which is what signing someone out
-- everywhere has to do.
CREATE INDEX IF NOT EXISTS idx_sessions_subject ON sessions(subject_kind, subject_id);

-- Covers the periodic sweep of expired rows. Nothing else deletes them.
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS sessions;
