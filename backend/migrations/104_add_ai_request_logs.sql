CREATE TABLE IF NOT EXISTS ai_request_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id VARCHAR(255) NOT NULL DEFAULT '',
    client_request_id VARCHAR(255) NOT NULL DEFAULT '',
    user_id BIGINT,
    api_key_id BIGINT,
    account_id BIGINT,
    group_id BIGINT,
    platform VARCHAR(32) NOT NULL DEFAULT '',
    model VARCHAR(255) NOT NULL DEFAULT '',
    request_path VARCHAR(255) NOT NULL DEFAULT '',
    inbound_endpoint VARCHAR(128) NOT NULL DEFAULT '',
    upstream_endpoint VARCHAR(128) NOT NULL DEFAULT '',
    method VARCHAR(16) NOT NULL DEFAULT 'POST',
    status_code INTEGER NOT NULL DEFAULT 0,
    stream BOOLEAN NOT NULL DEFAULT FALSE,
    request_body TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER,
    content_type VARCHAR(255) NOT NULL DEFAULT '',
    response_content_type VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ai_request_logs_created_at ON ai_request_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_request_logs_request_id ON ai_request_logs (request_id);
CREATE INDEX IF NOT EXISTS idx_ai_request_logs_client_request_id ON ai_request_logs (client_request_id);
CREATE INDEX IF NOT EXISTS idx_ai_request_logs_platform_created_at ON ai_request_logs (platform, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_request_logs_user_created_at ON ai_request_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_request_logs_group_created_at ON ai_request_logs (group_id, created_at DESC);
