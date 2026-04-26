SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS model_vendors (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon        VARCHAR(512) NOT NULL DEFAULT '',
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_model_vendors_name ON model_vendors (name);
CREATE INDEX IF NOT EXISTS idx_model_vendors_status ON model_vendors (status);

CREATE TABLE IF NOT EXISTS model_catalog (
    id            BIGSERIAL PRIMARY KEY,
    model_name    VARCHAR(128) NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    icon          VARCHAR(512) NOT NULL DEFAULT '',
    tags          VARCHAR(512) NOT NULL DEFAULT '',
    vendor_id     BIGINT REFERENCES model_vendors(id) ON DELETE SET NULL,
    endpoints     JSONB NOT NULL DEFAULT '[]',
    status        VARCHAR(20) NOT NULL DEFAULT 'active',
    sync_official BOOLEAN NOT NULL DEFAULT TRUE,
    name_rule     INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_model_catalog_model_name_deleted_at
    ON model_catalog (model_name, COALESCE(deleted_at, 'infinity'::timestamptz));
CREATE INDEX IF NOT EXISTS idx_model_catalog_vendor_id ON model_catalog (vendor_id);
CREATE INDEX IF NOT EXISTS idx_model_catalog_status ON model_catalog (status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_model_catalog_name_rule ON model_catalog (name_rule) WHERE deleted_at IS NULL;
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_trgm extension not created: %', SQLERRM;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
        CREATE INDEX IF NOT EXISTS idx_model_catalog_model_name_trgm
            ON model_catalog USING gin (model_name gin_trgm_ops) WHERE deleted_at IS NULL;
    END IF;
END $$;

COMMENT ON TABLE model_vendors IS 'Model vendor metadata used by the admin model catalog.';
COMMENT ON TABLE model_catalog IS 'Admin model catalog metadata; auxiliary only and not used for gateway routing.';
COMMENT ON COLUMN model_catalog.name_rule IS '0 exact, 1 prefix, 2 contains, 3 suffix.';
