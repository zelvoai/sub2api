-- 128_username_login_support.sql
-- Support username-only registration/login: make email nullable, add username unique index.

-- 1. Make email nullable so username-registered users don't need an email.
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

-- 2. Add partial unique index on normalized username for active (non-soft-deleted) users.
--    Uses LOWER(TRIM(username)) for case-insensitive uniqueness on ASCII portion.
--    Excludes empty/whitespace-only usernames and soft-deleted rows.
CREATE UNIQUE INDEX IF NOT EXISTS users_username_unique_active
    ON users (LOWER(TRIM(username)))
    WHERE deleted_at IS NULL
      AND username IS NOT NULL
      AND TRIM(username) <> '';

-- 3. Add 'username' to the signup_source CHECK constraint.
--    Drop and re-create to include the new value.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_signup_source_check;
ALTER TABLE users ADD CONSTRAINT users_signup_source_check
    CHECK (signup_source IN ('email', 'linuxdo', 'wechat', 'oidc', 'username'));
