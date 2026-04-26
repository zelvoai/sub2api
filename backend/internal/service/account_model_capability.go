package service

import (
	"context"
	"sort"
	"strings"
	"time"
)

const (
	AccountModelCapabilitySourceNewAPI = "newapi"
	AccountModelCapabilityStatusActive = "active"
)

type AccountModelCapability struct {
	ID                int64     `json:"id"`
	AccountID         int64     `json:"account_id"`
	GroupID           int64     `json:"group_id"`
	GroupName         string    `json:"group_name,omitempty"`
	ModelName         string    `json:"model_name"`
	UpstreamModelName string    `json:"upstream_model_name"`
	Provider          string    `json:"provider"`
	Source            string    `json:"source"`
	Status            string    `json:"status"`
	LastSeenAt        time.Time `json:"last_seen_at"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type AccountModelCapabilityInput struct {
	AccountID         int64
	GroupID           int64
	ModelName         string
	UpstreamModelName string
	Provider          string
	Source            string
	Status            string
}

type GroupAvailableModel struct {
	ModelName    string   `json:"model_name"`
	Provider     string   `json:"provider"`
	VendorName   string   `json:"vendor_name"`
	VendorIcon   string   `json:"vendor_icon"`
	AccountCount int      `json:"account_count"`
	GroupIDs     []int64  `json:"group_ids"`
	Groups       []string `json:"groups"`
	Priced       bool     `json:"priced"`
}

type AccountModelCapabilitySummary struct {
	AccountID  int64
	ModelCount int
	LastSeenAt *time.Time
}

type AccountModelCapabilityRepository interface {
	ReplaceForAccount(ctx context.Context, accountID int64, source string, caps []AccountModelCapabilityInput) error
	ListByGroupIDs(ctx context.Context, groupIDs []int64, search string, limit int) ([]GroupAvailableModel, error)
	CountByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]int, error)
	SummariesByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]AccountModelCapabilitySummary, error)
}

type AccountModelCapabilityService struct {
	repo AccountModelCapabilityRepository
}

func NewAccountModelCapabilityService(repo AccountModelCapabilityRepository) *AccountModelCapabilityService {
	return &AccountModelCapabilityService{repo: repo}
}

func (s *AccountModelCapabilityService) SyncAccountMapping(ctx context.Context, account *Account, source string, providers map[string]string) error {
	if s == nil || s.repo == nil || account == nil {
		return nil
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = AccountModelCapabilitySourceNewAPI
	}
	groupIDs := normalizeInt64List(account.GroupIDs)
	mapping := account.GetModelMapping()
	caps := make([]AccountModelCapabilityInput, 0, len(groupIDs)*len(mapping))
	for _, gid := range groupIDs {
		for modelName, upstreamName := range mapping {
			modelName = strings.TrimSpace(modelName)
			upstreamName = strings.TrimSpace(upstreamName)
			if modelName == "" {
				continue
			}
			if upstreamName == "" {
				upstreamName = modelName
			}
			provider := strings.TrimSpace(providers[upstreamName])
			if provider == "" {
				provider = InferModelProvider(upstreamName, account.GetCredential("base_url"))
			}
			caps = append(caps, AccountModelCapabilityInput{
				AccountID:         account.ID,
				GroupID:           gid,
				ModelName:         modelName,
				UpstreamModelName: upstreamName,
				Provider:          provider,
				Source:            source,
				Status:            AccountModelCapabilityStatusActive,
			})
		}
	}
	return s.repo.ReplaceForAccount(ctx, account.ID, source, caps)
}

func (s *AccountModelCapabilityService) ListGroupModels(ctx context.Context, groupIDs []int64, search string, limit int) ([]GroupAvailableModel, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListByGroupIDs(ctx, normalizeInt64List(groupIDs), search, limit)
}

func (s *AccountModelCapabilityService) SummariesByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]AccountModelCapabilitySummary, error) {
	if s == nil || s.repo == nil {
		return map[int64]AccountModelCapabilitySummary{}, nil
	}
	return s.repo.SummariesByAccountIDs(ctx, normalizeInt64List(accountIDs))
}

func normalizeInt64List(values []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(values))
	for _, v := range values {
		if v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
