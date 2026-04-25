package service

import "time"

type AIRequestLog struct {
	ID                  int64      `json:"id"`
	CreatedAt           time.Time  `json:"created_at"`
	RequestID           string     `json:"request_id"`
	ClientRequestID     string     `json:"client_request_id"`
	UserID              *int64     `json:"user_id,omitempty"`
	APIKeyID            *int64     `json:"api_key_id,omitempty"`
	AccountID           *int64     `json:"account_id,omitempty"`
	GroupID             *int64     `json:"group_id,omitempty"`
	Platform            string     `json:"platform"`
	Model               string     `json:"model"`
	RequestPath         string     `json:"request_path"`
	InboundEndpoint     string     `json:"inbound_endpoint"`
	UpstreamEndpoint    string     `json:"upstream_endpoint"`
	Method              string     `json:"method"`
	StatusCode          int        `json:"status_code"`
	Stream              bool       `json:"stream"`
	RequestBody         string     `json:"request_body"`
	ResponseBody        string     `json:"response_body"`
	ErrorMessage        string     `json:"error_message"`
	DurationMs          *int       `json:"duration_ms,omitempty"`
	ContentType         string     `json:"content_type"`
	ResponseContentType string     `json:"response_content_type"`
	DeletedAt           *time.Time `json:"-"`
}

type AIRequestLogFilter struct {
	Page            int
	PageSize        int
	RequestID       string
	ClientRequestID string
	Platform        string
	Model           string
	StatusCode      *int
	UserID          *int64
	APIKeyID        *int64
	AccountID       *int64
	GroupID         *int64
	Query           string
	StartTime       *time.Time
	EndTime         *time.Time
}

type AIRequestLogList struct {
	Items    []*AIRequestLog `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

type AIRequestLogRetentionSettings struct {
	Enabled               bool `json:"enabled"`
	RetentionHours        int  `json:"retention_hours"`
	CleanupIntervalMinute int  `json:"cleanup_interval_minutes"`
	DeleteBatchSize       int  `json:"delete_batch_size"`
}

func defaultAIRequestLogRetentionSettings() *AIRequestLogRetentionSettings {
	return &AIRequestLogRetentionSettings{
		Enabled:               true,
		RetentionHours:        24,
		CleanupIntervalMinute: 30,
		DeleteBatchSize:       2000,
	}
}

func normalizeAIRequestLogRetentionSettings(cfg *AIRequestLogRetentionSettings) {
	if cfg == nil {
		return
	}
	if cfg.RetentionHours < 6 {
		cfg.RetentionHours = 6
	}
	if cfg.RetentionHours > 168 {
		cfg.RetentionHours = 168
	}
	if cfg.CleanupIntervalMinute < 5 {
		cfg.CleanupIntervalMinute = 5
	}
	if cfg.CleanupIntervalMinute > 180 {
		cfg.CleanupIntervalMinute = 180
	}
	if cfg.DeleteBatchSize < 100 {
		cfg.DeleteBatchSize = 100
	}
	if cfg.DeleteBatchSize > 10000 {
		cfg.DeleteBatchSize = 10000
	}
}
