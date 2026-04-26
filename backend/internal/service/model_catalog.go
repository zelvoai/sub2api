package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	ModelStatusActive   = "active"
	ModelStatusDisabled = "disabled"

	ModelNameRuleExact    = 0
	ModelNameRulePrefix   = 1
	ModelNameRuleContains = 2
	ModelNameRuleSuffix   = 3
)

var (
	ErrModelVendorNotFound = infraerrors.NotFound("MODEL_VENDOR_NOT_FOUND", "model vendor not found")
	ErrModelVendorExists   = infraerrors.Conflict("MODEL_VENDOR_EXISTS", "model vendor name already exists")
	ErrModelNotFound       = infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	ErrModelExists         = infraerrors.Conflict("MODEL_EXISTS", "model name already exists")
)

type ModelVendor struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ModelBoundChannel struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Platform string   `json:"platform"`
	GroupIDs []int64  `json:"group_ids"`
	Groups   []string `json:"groups"`
}

type ModelCatalog struct {
	ID              int64               `json:"id"`
	ModelName       string              `json:"model_name"`
	Description     string              `json:"description"`
	Icon            string              `json:"icon"`
	Tags            string              `json:"tags"`
	VendorID        *int64              `json:"vendor_id"`
	VendorName      string              `json:"vendor_name"`
	VendorIcon      string              `json:"vendor_icon"`
	Endpoints       []string            `json:"endpoints"`
	Status          string              `json:"status"`
	SyncOfficial    bool                `json:"sync_official"`
	NameRule        int                 `json:"name_rule"`
	BoundChannels   []ModelBoundChannel `json:"bound_channels,omitempty"`
	EnableGroups    []string            `json:"enable_groups,omitempty"`
	QuotaTypes      []string            `json:"quota_types,omitempty"`
	MatchedModels   []string            `json:"matched_models,omitempty"`
	MatchedCount    int                 `json:"matched_count,omitempty"`
	AccountCount    int                 `json:"account_count,omitempty"`
	AvailableGroups []string            `json:"available_groups,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type MissingModel struct {
	ModelName    string              `json:"model_name"`
	NameRule     int                 `json:"name_rule"`
	Sources      []string            `json:"sources"`
	Platforms    []string            `json:"platforms"`
	Channels     []ModelBoundChannel `json:"channels"`
	MatchedCount int                 `json:"matched_count"`
}

type ModelSyncConflictField struct {
	Field    string `json:"field"`
	Local    any    `json:"local"`
	Upstream any    `json:"upstream"`
}

type ModelSyncConflict struct {
	ModelName string                   `json:"model_name"`
	Fields    []ModelSyncConflictField `json:"fields"`
}

type ModelSyncPreview struct {
	Missing   []upstreamModelCatalog `json:"missing"`
	Conflicts []ModelSyncConflict    `json:"conflicts"`
}

type ModelSyncOverwriteField struct {
	ModelName string   `json:"model_name"`
	Fields    []string `json:"fields"`
}

type ModelSyncRequest struct {
	Locale    string                    `json:"locale"`
	Overwrite []ModelSyncOverwriteField `json:"overwrite"`
}

type ModelSyncResult struct {
	CreatedModels    int      `json:"created_models"`
	CreatedVendors   int      `json:"created_vendors"`
	UpdatedModels    int      `json:"updated_models"`
	SkippedModels    []string `json:"skipped_models"`
	ConflictModels   []string `json:"conflict_models"`
	UpstreamModelURL string   `json:"upstream_model_url"`
}

type DiscoveredModelCatalogInput struct {
	ModelName string
	Provider  string
	Source    string
}

type ModelCatalogRepository interface {
	ListVendors(ctx context.Context, params pagination.PaginationParams, search, status string) ([]ModelVendor, *pagination.PaginationResult, error)
	GetVendorByID(ctx context.Context, id int64) (*ModelVendor, error)
	GetVendorByName(ctx context.Context, name string) (*ModelVendor, error)
	CreateVendor(ctx context.Context, vendor *ModelVendor) error
	UpdateVendor(ctx context.Context, vendor *ModelVendor) error
	DeleteVendor(ctx context.Context, id int64) error

	ListModels(ctx context.Context, params pagination.PaginationParams, filters ModelCatalogFilters) ([]ModelCatalog, *pagination.PaginationResult, error)
	GetModelByID(ctx context.Context, id int64) (*ModelCatalog, error)
	GetModelByName(ctx context.Context, name string) (*ModelCatalog, error)
	CreateModel(ctx context.Context, model *ModelCatalog) error
	UpdateModel(ctx context.Context, model *ModelCatalog) error
	UpdateModelStatus(ctx context.Context, id int64, status string) error
	DeleteModel(ctx context.Context, id int64) error
	BatchDeleteModels(ctx context.Context, ids []int64) error
	ExistingModelNames(ctx context.Context) (map[string]ModelCatalog, error)
	FindReferencedModels(ctx context.Context) ([]MissingModel, error)
	EnrichModels(ctx context.Context, models []ModelCatalog) ([]ModelCatalog, error)
}

type ModelCatalogFilters struct {
	Search   string
	VendorID *int64
	Status   string
	NameRule *int
}

type ModelVendorService struct {
	repo ModelCatalogRepository
}

func NewModelVendorService(repo ModelCatalogRepository) *ModelVendorService {
	return &ModelVendorService{repo: repo}
}

func (s *ModelVendorService) List(ctx context.Context, params pagination.PaginationParams, search, status string) ([]ModelVendor, *pagination.PaginationResult, error) {
	return s.repo.ListVendors(ctx, params, search, normalizeStatus(status))
}

func (s *ModelVendorService) Create(ctx context.Context, vendor *ModelVendor) error {
	normalizeVendor(vendor)
	if vendor.Name == "" {
		return infraerrors.BadRequest("MODEL_VENDOR_NAME_REQUIRED", "vendor name is required")
	}
	return s.repo.CreateVendor(ctx, vendor)
}

func (s *ModelVendorService) Update(ctx context.Context, vendor *ModelVendor) error {
	normalizeVendor(vendor)
	if vendor.ID <= 0 {
		return infraerrors.BadRequest("MODEL_VENDOR_ID_REQUIRED", "vendor id is required")
	}
	if vendor.Name == "" {
		return infraerrors.BadRequest("MODEL_VENDOR_NAME_REQUIRED", "vendor name is required")
	}
	return s.repo.UpdateVendor(ctx, vendor)
}

func (s *ModelVendorService) Delete(ctx context.Context, id int64) error {
	return s.repo.DeleteVendor(ctx, id)
}

type ModelCatalogService struct {
	repo   ModelCatalogRepository
	client *http.Client
}

func NewModelCatalogService(repo ModelCatalogRepository) *ModelCatalogService {
	return &ModelCatalogService{repo: repo, client: newModelSyncHTTPClient()}
}

func (s *ModelCatalogService) List(ctx context.Context, params pagination.PaginationParams, filters ModelCatalogFilters) ([]ModelCatalog, *pagination.PaginationResult, error) {
	filters.Status = normalizeStatus(filters.Status)
	models, pr, err := s.repo.ListModels(ctx, params, filters)
	if err != nil {
		return nil, nil, err
	}
	models, err = s.repo.EnrichModels(ctx, models)
	return models, pr, err
}

func (s *ModelCatalogService) Get(ctx context.Context, id int64) (*ModelCatalog, error) {
	m, err := s.repo.GetModelByID(ctx, id)
	if err != nil {
		return nil, err
	}
	enriched, err := s.repo.EnrichModels(ctx, []ModelCatalog{*m})
	if err != nil {
		return nil, err
	}
	if len(enriched) == 0 {
		return m, nil
	}
	return &enriched[0], nil
}

func (s *ModelCatalogService) Create(ctx context.Context, model *ModelCatalog) error {
	normalizeModel(model)
	if err := validateModelCatalog(model); err != nil {
		return err
	}
	return s.repo.CreateModel(ctx, model)
}

func (s *ModelCatalogService) Update(ctx context.Context, model *ModelCatalog) error {
	normalizeModel(model)
	if model.ID <= 0 {
		return infraerrors.BadRequest("MODEL_ID_REQUIRED", "model id is required")
	}
	if err := validateModelCatalog(model); err != nil {
		return err
	}
	return s.repo.UpdateModel(ctx, model)
}

func (s *ModelCatalogService) UpdateStatus(ctx context.Context, id int64, status string) error {
	status = normalizeStatus(status)
	if status == "" {
		return infraerrors.BadRequest("MODEL_STATUS_INVALID", "model status is invalid")
	}
	return s.repo.UpdateModelStatus(ctx, id, status)
}

func (s *ModelCatalogService) Delete(ctx context.Context, id int64) error {
	return s.repo.DeleteModel(ctx, id)
}

func (s *ModelCatalogService) BatchDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return s.repo.BatchDeleteModels(ctx, ids)
}

func (s *ModelCatalogService) Missing(ctx context.Context) ([]MissingModel, error) {
	existing, err := s.repo.ExistingModelNames(ctx)
	if err != nil {
		return nil, err
	}
	referenced, err := s.repo.FindReferencedModels(ctx)
	if err != nil {
		return nil, err
	}
	var missing []MissingModel
	for _, item := range referenced {
		if _, ok := existing[strings.ToLower(item.ModelName)]; !ok {
			missing = append(missing, item)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i].ModelName < missing[j].ModelName })
	return missing, nil
}

func (s *ModelCatalogService) EnsureDiscoveredModels(ctx context.Context, inputs []DiscoveredModelCatalogInput) (int, int, error) {
	if s == nil || s.repo == nil || len(inputs) == 0 {
		return 0, 0, nil
	}
	existing, err := s.repo.ExistingModelNames(ctx)
	if err != nil {
		return 0, 0, err
	}
	createdModels := 0
	createdVendors := 0
	for _, in := range inputs {
		name := strings.TrimSpace(in.ModelName)
		if name == "" {
			continue
		}
		provider := strings.TrimSpace(in.Provider)
		if provider == "" {
			provider = InferModelProvider(name, "")
		}
		if local, ok := existing[strings.ToLower(name)]; ok {
			if shouldRefreshDiscoveredModel(local, provider, in.Source) {
				vendorID, created, err := s.ensureVendor(ctx, provider, nil)
				if err != nil {
					return createdModels, createdVendors, err
				}
				if created {
					createdVendors++
				}
				local.VendorID = vendorID
				if strings.TrimSpace(local.Tags) == "" {
					local.Tags = strings.TrimSpace(in.Source)
				}
				if err := s.repo.UpdateModel(ctx, &local); err != nil {
					return createdModels, createdVendors, err
				}
				existing[strings.ToLower(name)] = local
			}
			continue
		}
		vendorID, created, err := s.ensureVendor(ctx, provider, nil)
		if err != nil {
			return createdModels, createdVendors, err
		}
		if created {
			createdVendors++
		}
		model := ModelCatalog{
			ModelName:    name,
			VendorID:     vendorID,
			Status:       ModelStatusActive,
			SyncOfficial: false,
			NameRule:     ModelNameRuleExact,
			Tags:         strings.TrimSpace(in.Source),
			Endpoints:    []string{},
		}
		if err := s.repo.CreateModel(ctx, &model); err != nil {
			if errors.Is(err, ErrModelExists) {
				continue
			}
			return createdModels, createdVendors, err
		}
		existing[strings.ToLower(name)] = model
		createdModels++
	}
	return createdModels, createdVendors, nil
}

func shouldRefreshDiscoveredModel(model ModelCatalog, provider, source string) bool {
	provider = strings.TrimSpace(provider)
	if provider == "" || isGenericUpstreamProvider(provider) {
		return false
	}
	currentVendor := strings.TrimSpace(model.VendorName)
	if currentVendor == "" || isGenericUpstreamProvider(currentVendor) {
		return true
	}
	return strings.TrimSpace(model.Tags) == "" && strings.TrimSpace(source) != ""
}

type upstreamEnvelope[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    []T    `json:"data"`
}

type upstreamModelCatalog struct {
	ModelName   string          `json:"model_name"`
	Description string          `json:"description"`
	Icon        string          `json:"icon"`
	Tags        string          `json:"tags"`
	VendorName  string          `json:"vendor_name"`
	Endpoints   json.RawMessage `json:"endpoints"`
	Status      int             `json:"status"`
	NameRule    int             `json:"name_rule"`
}

type upstreamVendorCatalog struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Status      int    `json:"status"`
}

var modelSyncCache = struct {
	sync.RWMutex
	etag map[string]string
	body map[string][]byte
}{etag: map[string]string{}, body: map[string][]byte{}}

func (s *ModelCatalogService) SyncPreview(ctx context.Context, locale string) (*ModelSyncPreview, error) {
	upModels, _, _, err := s.fetchUpstream(ctx, locale)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.ExistingModelNames(ctx)
	if err != nil {
		return nil, err
	}
	preview := &ModelSyncPreview{}
	for _, up := range upModels {
		up.ModelName = strings.TrimSpace(up.ModelName)
		if up.ModelName == "" {
			continue
		}
		local, ok := existing[strings.ToLower(up.ModelName)]
		if !ok {
			preview.Missing = append(preview.Missing, up)
			continue
		}
		fields := diffUpstreamModel(local, up)
		if len(fields) > 0 {
			preview.Conflicts = append(preview.Conflicts, ModelSyncConflict{ModelName: up.ModelName, Fields: fields})
		}
	}
	return preview, nil
}

func (s *ModelCatalogService) SyncUpstream(ctx context.Context, req ModelSyncRequest) (*ModelSyncResult, error) {
	upModels, upVendors, modelsURL, err := s.fetchUpstream(ctx, req.Locale)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.ExistingModelNames(ctx)
	if err != nil {
		return nil, err
	}
	vendorByName := make(map[string]upstreamVendorCatalog, len(upVendors))
	for _, v := range upVendors {
		if strings.TrimSpace(v.Name) != "" {
			vendorByName[strings.TrimSpace(v.Name)] = v
		}
	}
	overwrite := map[string]map[string]bool{}
	for _, ow := range req.Overwrite {
		m := strings.ToLower(strings.TrimSpace(ow.ModelName))
		if m == "" {
			continue
		}
		overwrite[m] = map[string]bool{}
		for _, f := range ow.Fields {
			overwrite[m][strings.TrimSpace(f)] = true
		}
	}
	res := &ModelSyncResult{UpstreamModelURL: modelsURL}
	for _, up := range upModels {
		name := strings.TrimSpace(up.ModelName)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		local, exists := existing[key]
		if !exists {
			vendorID, created, err := s.ensureVendor(ctx, up.VendorName, vendorByName)
			if err != nil {
				return nil, err
			}
			if created {
				res.CreatedVendors++
			}
			model := upstreamToModel(up, vendorID)
			if err := s.repo.CreateModel(ctx, &model); err != nil {
				return nil, err
			}
			res.CreatedModels++
			continue
		}
		fields := overwrite[key]
		if len(fields) == 0 {
			if len(diffUpstreamModel(local, up)) > 0 {
				res.ConflictModels = append(res.ConflictModels, name)
			} else {
				res.SkippedModels = append(res.SkippedModels, name)
			}
			continue
		}
		if !local.SyncOfficial {
			res.SkippedModels = append(res.SkippedModels, name)
			continue
		}
		updated := local
		applyUpstreamFields(&updated, up, fields)
		if fields["vendor"] {
			vendorID, created, err := s.ensureVendor(ctx, up.VendorName, vendorByName)
			if err != nil {
				return nil, err
			}
			if created {
				res.CreatedVendors++
			}
			updated.VendorID = vendorID
		}
		if err := s.repo.UpdateModel(ctx, &updated); err != nil {
			return nil, err
		}
		res.UpdatedModels++
	}
	sort.Strings(res.SkippedModels)
	sort.Strings(res.ConflictModels)
	return res, nil
}

func (s *ModelCatalogService) ensureVendor(ctx context.Context, name string, upstream map[string]upstreamVendorCatalog) (*int64, bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, false, nil
	}
	v, err := s.repo.GetVendorByName(ctx, name)
	if err == nil {
		return &v.ID, false, nil
	}
	if !errors.Is(err, ErrModelVendorNotFound) && infraerrors.Code(err) != 404 {
		return nil, false, err
	}
	var uv upstreamVendorCatalog
	if upstream != nil {
		uv = upstream[name]
	}
	v = &ModelVendor{Name: name, Description: strings.TrimSpace(uv.Description), Icon: strings.TrimSpace(uv.Icon), Status: upstreamStatus(uv.Status)}
	if err := s.repo.CreateVendor(ctx, v); err != nil {
		if errors.Is(err, ErrModelVendorExists) {
			v, err = s.repo.GetVendorByName(ctx, name)
			if err != nil {
				return nil, false, err
			}
			return &v.ID, false, nil
		}
		return nil, false, err
	}
	return &v.ID, true, nil
}

func (s *ModelCatalogService) fetchUpstream(ctx context.Context, locale string) ([]upstreamModelCatalog, []upstreamVendorCatalog, string, error) {
	modelsURL, vendorsURL := modelSyncURLs(locale)
	var modelsEnv upstreamEnvelope[upstreamModelCatalog]
	if err := s.fetchJSON(ctx, modelsURL, &modelsEnv); err != nil {
		return nil, nil, modelsURL, err
	}
	var vendorsEnv upstreamEnvelope[upstreamVendorCatalog]
	if err := s.fetchJSON(ctx, vendorsURL, &vendorsEnv); err != nil {
		return nil, nil, modelsURL, err
	}
	return modelsEnv.Data, vendorsEnv.Data, modelsURL, nil
}

func (s *ModelCatalogService) fetchJSON(ctx context.Context, url string, out any) error {
	attempts := envInt("SYNC_HTTP_RETRY", 3)
	if attempts < 1 {
		attempts = 1
	}
	maxBytes := int64(envInt("SYNC_HTTP_MAX_MB", 10)) << 20
	var lastErr error
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		modelSyncCache.RLock()
		if et := modelSyncCache.etag[url]; et != "" {
			req.Header.Set("If-None-Match", et)
		}
		modelSyncCache.RUnlock()
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			sleepBackoff(i)
			continue
		}
		func() {
			defer resp.Body.Close()
			var body []byte
			if resp.StatusCode == http.StatusNotModified {
				modelSyncCache.RLock()
				body = append([]byte(nil), modelSyncCache.body[url]...)
				modelSyncCache.RUnlock()
				if len(body) == 0 {
					lastErr = fmt.Errorf("sync upstream cache miss for %s", url)
					return
				}
			} else if resp.StatusCode == http.StatusOK {
				body, err = io.ReadAll(io.LimitReader(resp.Body, maxBytes))
				if err != nil {
					lastErr = err
					return
				}
				modelSyncCache.Lock()
				if et := resp.Header.Get("ETag"); et != "" {
					modelSyncCache.etag[url] = et
				}
				modelSyncCache.body[url] = append([]byte(nil), body...)
				modelSyncCache.Unlock()
			} else {
				lastErr = fmt.Errorf("sync upstream %s: %s", url, resp.Status)
				return
			}
			if err := decodeUpstream(body, out); err != nil {
				lastErr = err
				return
			}
			lastErr = nil
		}()
		if lastErr == nil {
			return nil
		}
		sleepBackoff(i)
	}
	return lastErr
}

func decodeUpstream(body []byte, out any) error {
	if err := json.Unmarshal(body, out); err == nil {
		return nil
	}
	switch target := out.(type) {
	case *upstreamEnvelope[upstreamModelCatalog]:
		var arr []upstreamModelCatalog
		if err := json.Unmarshal(body, &arr); err != nil {
			return err
		}
		target.Success = true
		target.Data = arr
	case *upstreamEnvelope[upstreamVendorCatalog]:
		var arr []upstreamVendorCatalog
		if err := json.Unmarshal(body, &arr); err != nil {
			return err
		}
		target.Success = true
		target.Data = arr
	default:
		return json.Unmarshal(body, out)
	}
	return nil
}

func normalizeVendor(v *ModelVendor) {
	if v == nil {
		return
	}
	v.Name = strings.TrimSpace(v.Name)
	v.Description = strings.TrimSpace(v.Description)
	v.Icon = strings.TrimSpace(v.Icon)
	v.Status = normalizeStatus(v.Status)
	if v.Status == "" {
		v.Status = ModelStatusActive
	}
}

func normalizeModel(m *ModelCatalog) {
	if m == nil {
		return
	}
	m.ModelName = strings.TrimSpace(m.ModelName)
	m.Description = strings.TrimSpace(m.Description)
	m.Icon = strings.TrimSpace(m.Icon)
	m.Tags = strings.TrimSpace(m.Tags)
	m.Status = normalizeStatus(m.Status)
	if m.Status == "" {
		m.Status = ModelStatusActive
	}
	m.Endpoints = normalizeStringList(m.Endpoints)
}

func validateModelCatalog(m *ModelCatalog) error {
	if m.ModelName == "" {
		return infraerrors.BadRequest("MODEL_NAME_REQUIRED", "model name is required")
	}
	if m.NameRule < ModelNameRuleExact || m.NameRule > ModelNameRuleSuffix {
		return infraerrors.BadRequest("MODEL_NAME_RULE_INVALID", "model name rule is invalid")
	}
	return nil
}

func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "all":
		return ""
	case ModelStatusActive, "1", "enabled":
		return ModelStatusActive
	case ModelStatusDisabled, "0":
		return ModelStatusDisabled
	default:
		return ""
	}
}

func normalizeStringList(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func upstreamStatus(status int) string {
	if status == 0 {
		return ModelStatusDisabled
	}
	return ModelStatusActive
}

func upstreamEndpoints(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return normalizeStringList(list)
	}
	var nums []int
	if err := json.Unmarshal(raw, &nums); err == nil {
		out := make([]string, 0, len(nums))
		for _, n := range nums {
			out = append(out, strconv.Itoa(n))
		}
		return out
	}
	return []string{}
}

func upstreamToModel(up upstreamModelCatalog, vendorID *int64) ModelCatalog {
	return ModelCatalog{
		ModelName:    strings.TrimSpace(up.ModelName),
		Description:  strings.TrimSpace(up.Description),
		Icon:         strings.TrimSpace(up.Icon),
		Tags:         strings.TrimSpace(up.Tags),
		VendorID:     vendorID,
		Endpoints:    upstreamEndpoints(up.Endpoints),
		Status:       upstreamStatus(up.Status),
		SyncOfficial: true,
		NameRule:     up.NameRule,
	}
}

func diffUpstreamModel(local ModelCatalog, up upstreamModelCatalog) []ModelSyncConflictField {
	var fields []ModelSyncConflictField
	if local.Description != strings.TrimSpace(up.Description) {
		fields = append(fields, ModelSyncConflictField{Field: "description", Local: local.Description, Upstream: strings.TrimSpace(up.Description)})
	}
	if local.Icon != strings.TrimSpace(up.Icon) {
		fields = append(fields, ModelSyncConflictField{Field: "icon", Local: local.Icon, Upstream: strings.TrimSpace(up.Icon)})
	}
	if local.Tags != strings.TrimSpace(up.Tags) {
		fields = append(fields, ModelSyncConflictField{Field: "tags", Local: local.Tags, Upstream: strings.TrimSpace(up.Tags)})
	}
	if strings.Join(local.Endpoints, ",") != strings.Join(upstreamEndpoints(up.Endpoints), ",") {
		fields = append(fields, ModelSyncConflictField{Field: "endpoints", Local: local.Endpoints, Upstream: upstreamEndpoints(up.Endpoints)})
	}
	if local.NameRule != up.NameRule {
		fields = append(fields, ModelSyncConflictField{Field: "name_rule", Local: local.NameRule, Upstream: up.NameRule})
	}
	if local.VendorName != strings.TrimSpace(up.VendorName) {
		fields = append(fields, ModelSyncConflictField{Field: "vendor", Local: local.VendorName, Upstream: strings.TrimSpace(up.VendorName)})
	}
	return fields
}

func applyUpstreamFields(local *ModelCatalog, up upstreamModelCatalog, fields map[string]bool) {
	if fields["description"] {
		local.Description = strings.TrimSpace(up.Description)
	}
	if fields["icon"] {
		local.Icon = strings.TrimSpace(up.Icon)
	}
	if fields["tags"] {
		local.Tags = strings.TrimSpace(up.Tags)
	}
	if fields["endpoints"] {
		local.Endpoints = upstreamEndpoints(up.Endpoints)
	}
	if fields["name_rule"] {
		local.NameRule = up.NameRule
	}
}

func modelSyncURLs(locale string) (string, string) {
	base := strings.TrimRight(os.Getenv("SYNC_UPSTREAM_BASE"), "/")
	if base == "" {
		base = "https://basellm.github.io/llm-metadata"
	}
	switch strings.TrimSpace(locale) {
	case "en", "zh-CN", "zh-TW", "ja":
		return base + "/api/i18n/" + locale + "/newapi/models.json", base + "/api/i18n/" + locale + "/newapi/vendors.json"
	default:
		return base + "/api/newapi/models.json", base + "/api/newapi/vendors.json"
	}
}

func envInt(key string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key))); err == nil {
		return v
	}
	return fallback
}

func newModelSyncHTTPClient() *http.Client {
	timeout := time.Duration(envInt("SYNC_HTTP_TIMEOUT_SECONDS", 10)) * time.Second
	dialer := &net.Dialer{Timeout: timeout}
	return &http.Client{
		Timeout: timeout + 5*time.Second,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			MaxIdleConns:          50,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
		},
	}
}

func sleepBackoff(attempt int) {
	base := 200 * time.Millisecond
	time.Sleep(base*time.Duration(1<<attempt) + time.Duration(rand.Intn(120))*time.Millisecond)
}
