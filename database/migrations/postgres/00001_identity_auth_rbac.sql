-- Identity, authentication and RBAC schema.
--
-- Primary keys are UUIDv7, generated in Go by pkg/id rather than by the database.
-- Two reasons there is no DEFAULT here:
--
--   1. PostgreSQL only gained a native uuidv7() in version 18, and Tusk supports
--      earlier versions. gen_random_uuid() would produce v4, losing the index
--      locality that motivated v7 in the first place.
--   2. More fundamentally, an offline-capable client must be able to create a row
--      and know its identifier before the database has ever seen it. A server-side
--      default cannot serve that case, so the application is the right place to
--      mint identifiers regardless of server version.
--
-- Other notes on this schema:
--   * TIMESTAMPTZ throughout. A wall-clock time with no zone is a latent bug the
--     moment a server, a client, or a DST boundary disagrees.
--   * No redundant indexes on UNIQUE columns — PostgreSQL already creates a
--     B-tree to enforce every UNIQUE constraint. Only non-unique columns that are
--     actually queried get an explicit index.
--   * VARCHAR(191) is retained to match the GORM model tags. The length is a
--     MySQL utf8mb4 index-limit artefact and carries no meaning here.

-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id                    UUID PRIMARY KEY,
    username              VARCHAR(191) NOT NULL UNIQUE,
    email                 VARCHAR(191) NOT NULL UNIQUE,
    phone                 VARCHAR(50)  UNIQUE,
    password              VARCHAR(255) NOT NULL,
    is_super_user         BOOLEAN      NOT NULL DEFAULT FALSE,
    is_active             BOOLEAN      NOT NULL DEFAULT TRUE,
    is_verified           BOOLEAN      NOT NULL DEFAULT FALSE,
    failed_login_attempts INTEGER      NOT NULL DEFAULT 0,
    locked_until          TIMESTAMPTZ,
    last_login_at         TIMESTAMPTZ,
    created_at            TIMESTAMPTZ,
    updated_at            TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL DEFAULT '',
    last_name  VARCHAR(100) NOT NULL DEFAULT '',
    avatar     VARCHAR(255) NOT NULL DEFAULT '',
    bio        TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ
);

-- user_id is not unique (a user may hold several active sessions), so this index
-- does real work: revoking every session for a user is a lookup by user_id.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);

CREATE TABLE IF NOT EXISTS permissions (
    name        VARCHAR(191) PRIMARY KEY,
    description TEXT,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY,
    name        VARCHAR(191) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id         UUID         NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_name VARCHAR(191) NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ,
    PRIMARY KEY (role_id, permission_name)
);

-- The composite primary key already indexes (role_id, permission_name), covering
-- lookups by role. The reverse — "which roles grant this permission?" — is not,
-- so it gets its own index.
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_name);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);

-- Same reasoning: the primary key covers user_id, this covers "who holds this role?"
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);

-- +goose Down
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS users;
