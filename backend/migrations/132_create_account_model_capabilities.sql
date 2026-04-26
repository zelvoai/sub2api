-- Internal account model capabilities discovered from upstream accounts.
-- Accounts remain admin-only; this table links backend accounts to user-visible groups/models.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS account_model_capabilities (
    id                  BIGSERIAL   PRIMARY KEY,
    account_id          BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    group_id            BIGINT      NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    model_name          TEXT        NOT NULL,
    upstream_model_name TEXT        NOT NULL,
    provider            TEXT        NOT NULL DEFAULT '',
    source              TEXT        NOT NULL DEFAULT 'manual',
    status              TEXT        NOT NULL DEFAULT 'active',
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_model_capabilities_unique
    ON account_model_capabilities (account_id, group_id, model_name, upstream_model_name, source);
CREATE INDEX IF NOT EXISTS idx_account_model_capabilities_group_model
    ON account_model_capabilities (group_id, model_name) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_account_model_capabilities_account
    ON account_model_capabilities (account_id);
CREATE INDEX IF NOT EXISTS idx_account_model_capabilities_last_seen
    ON account_model_capabilities (last_seen_at);

COMMENT ON TABLE account_model_capabilities IS 'Internal account-to-group model capabilities discovered from upstream accounts. Not exposed to end users.';
COMMENT ON COLUMN account_model_capabilities.source IS 'Capability source, e.g. newapi or manual.';
