package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type aiRequestLogRepository struct {
	db *sql.DB
}

func NewAIRequestLogRepository(db *sql.DB) service.AIRequestLogRepository {
	return &aiRequestLogRepository{db: db}
}

func (r *aiRequestLogRepository) Insert(ctx context.Context, input *service.AIRequestLog) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ai request log repository")
	}
	if input == nil {
		return fmt.Errorf("nil input")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO ai_request_logs (
  created_at, request_id, client_request_id, user_id, api_key_id, account_id, group_id,
  platform, model, request_path, inbound_endpoint, upstream_endpoint, method, status_code,
  stream, request_body, response_body, error_message, duration_ms, content_type, response_content_type
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,
  $8,$9,$10,$11,$12,$13,$14,
  $15,$16,$17,$18,$19,$20,$21
)`,
		input.CreatedAt,
		strings.TrimSpace(input.RequestID),
		strings.TrimSpace(input.ClientRequestID),
		aiNullInt64(input.UserID),
		aiNullInt64(input.APIKeyID),
		aiNullInt64(input.AccountID),
		aiNullInt64(input.GroupID),
		strings.TrimSpace(input.Platform),
		strings.TrimSpace(input.Model),
		strings.TrimSpace(input.RequestPath),
		strings.TrimSpace(input.InboundEndpoint),
		strings.TrimSpace(input.UpstreamEndpoint),
		strings.TrimSpace(input.Method),
		input.StatusCode,
		input.Stream,
		input.RequestBody,
		input.ResponseBody,
		input.ErrorMessage,
		aiNullInt(input.DurationMs),
		strings.TrimSpace(input.ContentType),
		strings.TrimSpace(input.ResponseContentType),
	)
	return err
}

func (r *aiRequestLogRepository) List(ctx context.Context, filter *service.AIRequestLogFilter) ([]*service.AIRequestLog, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("nil ai request log repository")
	}
	page := 1
	pageSize := 50
	if filter != nil {
		if filter.Page > 0 {
			page = filter.Page
		}
		if filter.PageSize > 0 {
			pageSize = filter.PageSize
		}
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize

	conditions := make([]string, 0, 16)
	args := make([]any, 0, 16)
	add := func(expr string, values ...any) {
		conditions = append(conditions, expr)
		args = append(args, values...)
	}
	if filter != nil {
		if v := strings.TrimSpace(filter.RequestID); v != "" {
			add(fmt.Sprintf("request_id = $%d", len(args)+1), v)
		}
		if v := strings.TrimSpace(filter.ClientRequestID); v != "" {
			add(fmt.Sprintf("client_request_id = $%d", len(args)+1), v)
		}
		if v := strings.TrimSpace(strings.ToLower(filter.Platform)); v != "" {
			add(fmt.Sprintf("platform = $%d", len(args)+1), v)
		}
		if v := strings.TrimSpace(filter.Model); v != "" {
			add(fmt.Sprintf("model = $%d", len(args)+1), v)
		}
		if filter.StatusCode != nil {
			add(fmt.Sprintf("status_code = $%d", len(args)+1), *filter.StatusCode)
		}
		if filter.UserID != nil {
			add(fmt.Sprintf("user_id = $%d", len(args)+1), *filter.UserID)
		}
		if filter.APIKeyID != nil {
			add(fmt.Sprintf("api_key_id = $%d", len(args)+1), *filter.APIKeyID)
		}
		if filter.AccountID != nil {
			add(fmt.Sprintf("account_id = $%d", len(args)+1), *filter.AccountID)
		}
		if filter.GroupID != nil {
			add(fmt.Sprintf("group_id = $%d", len(args)+1), *filter.GroupID)
		}
		if filter.StartTime != nil {
			add(fmt.Sprintf("created_at >= $%d", len(args)+1), filter.StartTime.UTC())
		}
		if filter.EndTime != nil {
			add(fmt.Sprintf("created_at < $%d", len(args)+1), filter.EndTime.UTC())
		}
		if q := strings.TrimSpace(strings.ToLower(filter.Query)); q != "" {
			like := "%" + q + "%"
			add(fmt.Sprintf("(LOWER(COALESCE(request_id,'')) LIKE $%d OR LOWER(COALESCE(client_request_id,'')) LIKE $%d OR LOWER(COALESCE(model,'')) LIKE $%d OR LOWER(COALESCE(error_message,'')) LIKE $%d)", len(args)+1, len(args)+2, len(args)+3, len(args)+4), like, like, like, like)
		}
	}
	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	countQuery := "SELECT COUNT(1) FROM ai_request_logs " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listQuery := fmt.Sprintf(`
SELECT
  id, created_at, request_id, client_request_id, user_id, api_key_id, account_id, group_id,
  platform, model, request_path, inbound_endpoint, upstream_endpoint, method, status_code,
  stream, request_body, response_body, error_message, duration_ms, content_type, response_content_type
FROM ai_request_logs
%s
ORDER BY created_at DESC, id DESC
LIMIT $%d OFFSET $%d
`, where, len(args)+1, len(args)+2)
	rows, err := r.db.QueryContext(ctx, listQuery, append(args, pageSize, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.AIRequestLog, 0, pageSize)
	for rows.Next() {
		item, err := scanAIRequestLog(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (r *aiRequestLogRepository) GetByID(ctx context.Context, id int64) (*service.AIRequestLog, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ai request log repository")
	}
	row := r.db.QueryRowContext(ctx, `
SELECT
  id, created_at, request_id, client_request_id, user_id, api_key_id, account_id, group_id,
  platform, model, request_path, inbound_endpoint, upstream_endpoint, method, status_code,
  stream, request_body, response_body, error_message, duration_ms, content_type, response_content_type
FROM ai_request_logs
WHERE id = $1
LIMIT 1
`, id)
	return scanAIRequestLog(row)
}

func (r *aiRequestLogRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ai request log repository")
	}
	if limit <= 0 {
		limit = 2000
	}
	res, err := r.db.ExecContext(ctx, `
WITH batch AS (
  SELECT id FROM ai_request_logs
  WHERE created_at < $1
  ORDER BY id
  LIMIT $2
)
DELETE FROM ai_request_logs
WHERE id IN (SELECT id FROM batch)
`, cutoff.UTC(), limit)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type aiRequestLogScanner interface {
	Scan(dest ...any) error
}

func scanAIRequestLog(scanner aiRequestLogScanner) (*service.AIRequestLog, error) {
	var item service.AIRequestLog
	var userID, apiKeyID, accountID, groupID sql.NullInt64
	var durationMs sql.NullInt64
	if err := scanner.Scan(
		&item.ID,
		&item.CreatedAt,
		&item.RequestID,
		&item.ClientRequestID,
		&userID,
		&apiKeyID,
		&accountID,
		&groupID,
		&item.Platform,
		&item.Model,
		&item.RequestPath,
		&item.InboundEndpoint,
		&item.UpstreamEndpoint,
		&item.Method,
		&item.StatusCode,
		&item.Stream,
		&item.RequestBody,
		&item.ResponseBody,
		&item.ErrorMessage,
		&durationMs,
		&item.ContentType,
		&item.ResponseContentType,
	); err != nil {
		return nil, err
	}
	if userID.Valid {
		v := userID.Int64
		item.UserID = &v
	}
	if apiKeyID.Valid {
		v := apiKeyID.Int64
		item.APIKeyID = &v
	}
	if accountID.Valid {
		v := accountID.Int64
		item.AccountID = &v
	}
	if groupID.Valid {
		v := groupID.Int64
		item.GroupID = &v
	}
	if durationMs.Valid {
		v := int(durationMs.Int64)
		item.DurationMs = &v
	}
	return &item, nil
}

func aiNullInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func aiNullInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
