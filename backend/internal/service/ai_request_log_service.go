package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const SettingKeyAIRequestLogRetention = "ai_request_log_retention"

type AIRequestLogRepository interface {
	Insert(ctx context.Context, input *AIRequestLog) error
	List(ctx context.Context, filter *AIRequestLogFilter) ([]*AIRequestLog, int64, error)
	GetByID(ctx context.Context, id int64) (*AIRequestLog, error)
	DeleteOlderThan(ctx context.Context, cutoff time.Time, limit int) (int64, error)
}

type AIRequestLogService struct {
	repo       AIRequestLogRepository
	settingRepo SettingRepository
	cfg        *AIRequestLogRetentionSettings
	clock      func() time.Time
}

func NewAIRequestLogService(repo AIRequestLogRepository, settingRepo SettingRepository) *AIRequestLogService {
	return &AIRequestLogService{
		repo:        repo,
		settingRepo: settingRepo,
		cfg:         defaultAIRequestLogRetentionSettings(),
		clock:       time.Now,
	}
}

func (s *AIRequestLogService) Record(ctx context.Context, entry *AIRequestLog) error {
	if s == nil || s.repo == nil || entry == nil {
		return nil
	}
	settings, err := s.GetRetentionSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return nil
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.clock().UTC()
	}
	entry.RequestID = strings.TrimSpace(entry.RequestID)
	entry.ClientRequestID = strings.TrimSpace(entry.ClientRequestID)
	entry.Platform = strings.TrimSpace(strings.ToLower(entry.Platform))
	entry.Model = strings.TrimSpace(entry.Model)
	entry.RequestPath = strings.TrimSpace(entry.RequestPath)
	entry.InboundEndpoint = strings.TrimSpace(entry.InboundEndpoint)
	entry.UpstreamEndpoint = strings.TrimSpace(entry.UpstreamEndpoint)
	entry.Method = strings.TrimSpace(strings.ToUpper(entry.Method))
	entry.ContentType = strings.TrimSpace(entry.ContentType)
	entry.ResponseContentType = strings.TrimSpace(entry.ResponseContentType)
	return s.repo.Insert(ctx, entry)
}

func (s *AIRequestLogService) List(ctx context.Context, filter *AIRequestLogFilter) (*AIRequestLogList, error) {
	if s == nil || s.repo == nil {
		return &AIRequestLogList{Items: []*AIRequestLog{}, Page: 1, PageSize: 50}, nil
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
	copyFilter := &AIRequestLogFilter{Page: page, PageSize: pageSize}
	if filter != nil {
		*copyFilter = *filter
		copyFilter.Page = page
		copyFilter.PageSize = pageSize
	}
	items, total, err := s.repo.List(ctx, copyFilter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*AIRequestLog{}
	}
	return &AIRequestLogList{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *AIRequestLogService) GetByID(ctx context.Context, id int64) (*AIRequestLog, error) {
	if s == nil || s.repo == nil {
		return nil, infraerrors.NotFound("AI_REQUEST_LOG_NOT_FOUND", "ai request log not found")
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return nil, infraerrors.NotFound("AI_REQUEST_LOG_NOT_FOUND", "ai request log not found")
		}
		return nil, err
	}
	if item == nil {
		return nil, infraerrors.NotFound("AI_REQUEST_LOG_NOT_FOUND", "ai request log not found")
	}
	return item, nil
}

func (s *AIRequestLogService) GetRetentionSettings(ctx context.Context) (*AIRequestLogRetentionSettings, error) {
	defaultCfg := defaultAIRequestLogRetentionSettings()
	if s == nil || s.settingRepo == nil {
		return defaultCfg, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyAIRequestLogRetention)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			encoded, mErr := json.Marshal(defaultCfg)
			if mErr == nil {
				_ = s.settingRepo.Set(ctx, SettingKeyAIRequestLogRetention, string(encoded))
			}
			return defaultCfg, nil
		}
		return nil, err
	}
	cfg := defaultAIRequestLogRetentionSettings()
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return defaultCfg, nil
	}
	normalizeAIRequestLogRetentionSettings(cfg)
	return cfg, nil
}

func (s *AIRequestLogService) UpdateRetentionSettings(ctx context.Context, cfg *AIRequestLogRetentionSettings) (*AIRequestLogRetentionSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, errors.New("setting repository not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, errors.New("invalid config")
	}
	normalizeAIRequestLogRetentionSettings(cfg)
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyAIRequestLogRetention, string(raw)); err != nil {
		return nil, err
	}
	out := &AIRequestLogRetentionSettings{}
	_ = json.Unmarshal(raw, out)
	return out, nil
}

func (s *AIRequestLogService) Cleanup(ctx context.Context, limit int) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	settings, err := s.GetRetentionSettings(ctx)
	if err != nil {
		return 0, err
	}
	if !settings.Enabled {
		return 0, nil
	}
	if limit <= 0 {
		limit = settings.DeleteBatchSize
	}
	cutoff := s.clock().UTC().Add(-time.Duration(settings.RetentionHours) * time.Hour)
	return s.repo.DeleteOlderThan(ctx, cutoff, limit)
}
