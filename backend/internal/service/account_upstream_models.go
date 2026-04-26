package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	accountUpstreamSyncKeyEnabled     = "upstream_model_sync_enabled"
	accountUpstreamSyncKeyAutoAdd     = "upstream_model_sync_auto_add"
	accountUpstreamSyncKeyIgnored     = "upstream_model_sync_ignored_models"
	accountUpstreamSyncKeyDetected    = "upstream_model_sync_last_detected_models"
	accountUpstreamSyncKeyRemoved     = "upstream_model_sync_last_removed_models"
	accountUpstreamSyncKeyLastCheck   = "upstream_model_sync_last_check_time"
	accountUpstreamSyncKeyCompatType  = "upstream_compat_type"
	accountUpstreamSyncKeyGroupName   = "upstream_group_name"
	accountUpstreamSyncCompatNewAPI   = "newapi"
	accountUpstreamSyncRequestTimeout = 15 * time.Second
)

type DiscoveredUpstreamModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Provider    string `json:"provider"`
	OwnedBy     string `json:"owned_by,omitempty"`
	Source      string `json:"source"`
}

type AccountUpstreamModelPreviewRequest struct {
	AccountID         int64             `json:"account_id"`
	BaseURL           string            `json:"base_url"`
	APIKey            string            `json:"api_key"`
	Platform          string            `json:"platform"`
	CompatType        string            `json:"compat_type"`
	GroupIDs          []int64           `json:"group_ids"`
	Credentials       map[string]any    `json:"credentials"`
	ExistingMapping   map[string]string `json:"existing_mapping"`
	UpstreamGroupName string            `json:"upstream_group_name"`
}

type AccountUpstreamModelDiff struct {
	AccountID      int64                     `json:"account_id,omitempty"`
	GroupIDs       []int64                   `json:"group_ids"`
	Models         []DiscoveredUpstreamModel `json:"models"`
	AddModels      []DiscoveredUpstreamModel `json:"add_models"`
	ExistingModels []DiscoveredUpstreamModel `json:"existing_models"`
	RemoveModels   []string                  `json:"remove_models"`
	IgnoredModels  []string                  `json:"ignored_models"`
	LastCheckTime  int64                     `json:"last_check_time"`
}

type AccountUpstreamModelApplyRequest struct {
	AddModels          []string `json:"add_models"`
	RemoveModels       []string `json:"remove_models"`
	IgnoreModels       []string `json:"ignore_models"`
	SyncToModelCatalog *bool    `json:"sync_to_model_catalog"`
	UpstreamGroupName  string   `json:"upstream_group_name"`
}

type AccountUpstreamModelApplyResult struct {
	AddedModels     []string `json:"added_models"`
	RemovedModels   []string `json:"removed_models"`
	IgnoredModels   []string `json:"ignored_models"`
	RemainingModels []string `json:"remaining_models"`
	CreatedModels   int      `json:"created_models"`
	CreatedVendors  int      `json:"created_vendors"`
}

type AccountUpstreamModelImportCatalogRequest struct {
	Models []DiscoveredUpstreamModel `json:"models"`
}

type AccountUpstreamModelImportCatalogResult struct {
	CreatedModels  int `json:"created_models"`
	CreatedVendors int `json:"created_vendors"`
}

type AccountUpstreamModelService struct {
	accountRepo AccountRepository
	capability  *AccountModelCapabilityService
	catalog     *ModelCatalogService
	client      *http.Client
}

func NewAccountUpstreamModelService(accountRepo AccountRepository, capability *AccountModelCapabilityService, catalog *ModelCatalogService) *AccountUpstreamModelService {
	return &AccountUpstreamModelService{
		accountRepo: accountRepo,
		capability:  capability,
		catalog:     catalog,
		client:      &http.Client{Timeout: accountUpstreamSyncRequestTimeout},
	}
}

func (s *AccountUpstreamModelService) Preview(ctx context.Context, req AccountUpstreamModelPreviewRequest) (*AccountUpstreamModelDiff, error) {
	if s == nil {
		return nil, infraerrors.InternalServer("UPSTREAM_MODELS_SERVICE_UNAVAILABLE", "upstream model service unavailable")
	}
	compat := strings.TrimSpace(req.CompatType)
	if compat == "" {
		compat = accountUpstreamSyncCompatNewAPI
	}
	if compat != accountUpstreamSyncCompatNewAPI {
		return nil, infraerrors.BadRequest("UPSTREAM_MODELS_COMPAT_UNSUPPORTED", "compat_type must be newapi")
	}
	baseURL, apiKey := reqBaseAndKey(req)
	if baseURL == "" || apiKey == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_MODELS_CREDENTIALS_REQUIRED", "base_url and api_key are required")
	}
	models, err := s.fetchNewAPIModels(ctx, baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	mapping := normalizeMapping(req.ExistingMapping)
	diff := buildUpstreamModelDiff(models, mapping, nil)
	diff.GroupIDs = normalizeInt64List(req.GroupIDs)
	return diff, nil
}

func (s *AccountUpstreamModelService) Detect(ctx context.Context, accountID int64) (*AccountUpstreamModelDiff, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if baseURL == "" || apiKey == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_MODELS_CREDENTIALS_REQUIRED", "account base_url and api_key are required")
	}
	models, err := s.fetchNewAPIModels(ctx, baseURL, apiKey)
	if err != nil {
		return nil, err
	}
	ignored := stringSliceFromAny(account.Extra[accountUpstreamSyncKeyIgnored])
	diff := buildUpstreamModelDiff(models, account.GetModelMapping(), ignored)
	diff.AccountID = account.ID
	diff.GroupIDs = normalizeInt64List(account.GroupIDs)
	diff.LastCheckTime = time.Now().Unix()
	_ = s.accountRepo.UpdateExtra(ctx, account.ID, map[string]any{
		accountUpstreamSyncKeyDetected:   modelIDs(diff.AddModels),
		accountUpstreamSyncKeyRemoved:    diff.RemoveModels,
		accountUpstreamSyncKeyLastCheck:  diff.LastCheckTime,
		accountUpstreamSyncKeyCompatType: accountUpstreamSyncCompatNewAPI,
	})
	return diff, nil
}

func (s *AccountUpstreamModelService) Apply(ctx context.Context, accountID int64, req AccountUpstreamModelApplyRequest) (*AccountUpstreamModelApplyResult, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	addModels := normalizeStringList(req.AddModels)
	removeModels := normalizeStringList(req.RemoveModels)
	ignoreModels := normalizeStringList(req.IgnoreModels)
	mapping := account.GetModelMapping()
	if mapping == nil {
		mapping = map[string]string{}
	}
	for _, model := range addModels {
		if _, exists := mapping[model]; !exists {
			mapping[model] = model
		}
	}
	removed := make([]string, 0, len(removeModels))
	removeSet := stringSet(removeModels)
	for src, dst := range mapping {
		if _, ok := removeSet[src]; ok {
			delete(mapping, src)
			removed = append(removed, src)
			continue
		}
		if _, ok := removeSet[dst]; ok {
			delete(mapping, src)
			removed = append(removed, src)
		}
	}
	credentials := cloneCredentials(account.Credentials)
	credentials["model_mapping"] = mapping
	if err := persistAccountCredentials(ctx, s.accountRepo, account, credentials); err != nil {
		return nil, err
	}
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	existingIgnored := stringSliceFromAny(account.Extra[accountUpstreamSyncKeyIgnored])
	nextIgnored := mergeStrings(existingIgnored, ignoreModels)
	nextIgnored = subtractStrings(nextIgnored, addModels)
	extra := map[string]any{
		accountUpstreamSyncKeyIgnored:    nextIgnored,
		accountUpstreamSyncKeyDetected:   []string{},
		accountUpstreamSyncKeyRemoved:    []string{},
		accountUpstreamSyncKeyLastCheck:  time.Now().Unix(),
		accountUpstreamSyncKeyCompatType: accountUpstreamSyncCompatNewAPI,
	}
	if strings.TrimSpace(req.UpstreamGroupName) != "" {
		extra[accountUpstreamSyncKeyGroupName] = strings.TrimSpace(req.UpstreamGroupName)
	}
	_ = s.accountRepo.UpdateExtra(ctx, account.ID, extra)
	providers := map[string]string{}
	for _, model := range addModels {
		providers[model] = InferModelProvider(model, account.GetCredential("base_url"))
	}
	if s.capability != nil {
		_ = s.capability.SyncAccountMapping(ctx, account, AccountModelCapabilitySourceNewAPI, providers)
	}
	createdModels, createdVendors := 0, 0
	if req.SyncToModelCatalog == nil || *req.SyncToModelCatalog {
		inputs := make([]DiscoveredModelCatalogInput, 0, len(addModels))
		for _, model := range addModels {
			inputs = append(inputs, DiscoveredModelCatalogInput{ModelName: model, Provider: providers[model], Source: "newapi"})
		}
		cm, cv, err := s.catalog.EnsureDiscoveredModels(ctx, inputs)
		if err != nil {
			return nil, err
		}
		createdModels, createdVendors = cm, cv
	}
	return &AccountUpstreamModelApplyResult{
		AddedModels:     addModels,
		RemovedModels:   removed,
		IgnoredModels:   nextIgnored,
		RemainingModels: []string{},
		CreatedModels:   createdModels,
		CreatedVendors:  createdVendors,
	}, nil
}

func (s *AccountUpstreamModelService) ImportCatalog(ctx context.Context, req AccountUpstreamModelImportCatalogRequest) (*AccountUpstreamModelImportCatalogResult, error) {
	if s == nil || s.catalog == nil {
		return nil, infraerrors.InternalServer("UPSTREAM_MODELS_SERVICE_UNAVAILABLE", "upstream model service unavailable")
	}
	inputs := make([]DiscoveredModelCatalogInput, 0, len(req.Models))
	for _, model := range req.Models {
		name := strings.TrimSpace(model.ID)
		if name == "" {
			continue
		}
		provider := normalizeDiscoveredProvider(name, model.Provider, model.OwnedBy, "")
		inputs = append(inputs, DiscoveredModelCatalogInput{ModelName: name, Provider: provider, Source: "newapi"})
	}
	createdModels, createdVendors, err := s.catalog.EnsureDiscoveredModels(ctx, inputs)
	if err != nil {
		return nil, err
	}
	return &AccountUpstreamModelImportCatalogResult{CreatedModels: createdModels, CreatedVendors: createdVendors}, nil
}

func (s *AccountUpstreamModelService) fetchNewAPIModels(ctx context.Context, baseURL, apiKey string) ([]DiscoveredUpstreamModel, error) {
	url := newAPIModelsURL(baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, infraerrors.BadRequest("UPSTREAM_MODELS_INVALID_BASE_URL", "invalid upstream base_url")
	}
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "UPSTREAM_MODELS_FETCH_FAILED", "failed to fetch upstream models")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, infraerrors.New(resp.StatusCode, "UPSTREAM_MODELS_FETCH_FAILED", fmt.Sprintf("upstream returned status %d", resp.StatusCode))
	}
	var payload struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			OwnedBy     string `json:"owned_by"`
			Provider    string `json:"provider"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "UPSTREAM_MODELS_INVALID_RESPONSE", "invalid upstream models response")
	}
	if len(payload.Data) == 0 {
		return nil, infraerrors.New(http.StatusBadGateway, "UPSTREAM_MODELS_EMPTY", "upstream returned no models")
	}
	models := make([]DiscoveredUpstreamModel, 0, len(payload.Data))
	seen := map[string]struct{}{}
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		provider := normalizeDiscoveredProvider(id, item.Provider, item.OwnedBy, baseURL)
		display := strings.TrimSpace(item.DisplayName)
		if display == "" {
			display = id
		}
		models = append(models, DiscoveredUpstreamModel{ID: id, DisplayName: display, Provider: provider, OwnedBy: item.OwnedBy, Source: AccountModelCapabilitySourceNewAPI})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return models, nil
}

func normalizeDiscoveredProvider(modelName, provider, ownedBy, fallbackHost string) string {
	provider = strings.TrimSpace(provider)
	ownedBy = strings.TrimSpace(ownedBy)
	if isGenericUpstreamProvider(provider) {
		provider = ""
	}
	if isGenericUpstreamProvider(ownedBy) {
		ownedBy = ""
	}
	inferred := InferModelProvider(modelName, fallbackHost)
	if provider == "" && ownedBy == "" {
		return inferred
	}
	if provider != "" {
		return provider
	}
	if ownedBy != "" {
		return ownedBy
	}
	return inferred
}

func isGenericUpstreamProvider(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "newapi", "new-api", "oneapi", "one-api", "custom", "codex", "default", "system":
		return true
	default:
		return false
	}
}

func reqBaseAndKey(req AccountUpstreamModelPreviewRequest) (string, string) {
	baseURL := strings.TrimSpace(req.BaseURL)
	apiKey := strings.TrimSpace(req.APIKey)
	if baseURL == "" {
		baseURL = stringFromMap(req.Credentials, "base_url")
	}
	if apiKey == "" {
		apiKey = stringFromMap(req.Credentials, "api_key")
	}
	return baseURL, apiKey
}

func newAPIModelsURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/models"
	}
	return trimmed + "/v1/models"
}

func buildUpstreamModelDiff(models []DiscoveredUpstreamModel, mapping map[string]string, ignored []string) *AccountUpstreamModelDiff {
	mapping = normalizeMapping(mapping)
	ignored = normalizeStringList(ignored)
	if ignored == nil {
		ignored = []string{}
	}
	ignoredSet := stringSet(ignored)
	covered := map[string]struct{}{}
	for src, dst := range mapping {
		covered[src] = struct{}{}
		if dst != "" {
			covered[dst] = struct{}{}
		}
	}
	upstreamSet := map[string]DiscoveredUpstreamModel{}
	for _, m := range models {
		upstreamSet[m.ID] = m
	}
	diff := &AccountUpstreamModelDiff{
		Models:         models,
		AddModels:      []DiscoveredUpstreamModel{},
		ExistingModels: []DiscoveredUpstreamModel{},
		RemoveModels:   []string{},
		IgnoredModels:  ignored,
	}
	for _, m := range models {
		if _, ok := covered[m.ID]; ok {
			diff.ExistingModels = append(diff.ExistingModels, m)
			continue
		}
		if _, ok := ignoredSet[m.ID]; ok {
			continue
		}
		diff.AddModels = append(diff.AddModels, m)
	}
	for src, dst := range mapping {
		if src != dst {
			continue
		}
		if _, ok := upstreamSet[dst]; !ok {
			diff.RemoveModels = append(diff.RemoveModels, src)
		}
	}
	sort.Strings(diff.RemoveModels)
	return diff
}

func normalizeMapping(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			continue
		}
		if v == "" {
			v = k
		}
		out[k] = v
	}
	return out
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func modelIDs(models []DiscoveredUpstreamModel) []string {
	out := make([]string, 0, len(models))
	for _, m := range models {
		out = append(out, m.ID)
	}
	return out
}

func stringSliceFromAny(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return normalizeStringList(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return normalizeStringList(out)
	default:
		return nil
	}
}

func stringSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, v := range normalizeStringList(values) {
		out[v] = struct{}{}
	}
	return out
}

func mergeStrings(a, b []string) []string {
	set := stringSet(a)
	for _, v := range normalizeStringList(b) {
		set[v] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func subtractStrings(a, b []string) []string {
	remove := stringSet(b)
	out := make([]string, 0, len(a))
	for _, v := range normalizeStringList(a) {
		if _, ok := remove[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}
