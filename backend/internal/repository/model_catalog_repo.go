package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type modelCatalogRepository struct {
	db *sql.DB
}

func NewModelCatalogRepository(db *sql.DB) service.ModelCatalogRepository {
	return &modelCatalogRepository{db: db}
}

func (r *modelCatalogRepository) ListVendors(ctx context.Context, params pagination.PaginationParams, search, status string) ([]service.ModelVendor, *pagination.PaginationResult, error) {
	where, args := []string{"1=1"}, []any{}
	idx := 1
	if search != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", idx, idx))
		args = append(args, "%"+escapeLike(search)+"%")
		idx++
	}
	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", idx))
		args = append(args, status)
		idx++
	}
	clause := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_vendors WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count model vendors: %w", err)
	}
	limit := params.Limit()
	offset := params.Offset()
	query := fmt.Sprintf("SELECT id, name, description, icon, status, created_at, updated_at FROM model_vendors WHERE %s ORDER BY name ASC LIMIT $%d OFFSET $%d", clause, idx, idx+1)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("list model vendors: %w", err)
	}
	defer rows.Close()
	var vendors []service.ModelVendor
	for rows.Next() {
		var v service.ModelVendor
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.Icon, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, nil, err
		}
		vendors = append(vendors, v)
	}
	return vendors, pageResult(total, params.Page, limit), rows.Err()
}

func (r *modelCatalogRepository) GetVendorByID(ctx context.Context, id int64) (*service.ModelVendor, error) {
	return r.getVendor(ctx, "id = $1", id)
}

func (r *modelCatalogRepository) GetVendorByName(ctx context.Context, name string) (*service.ModelVendor, error) {
	return r.getVendor(ctx, "name = $1", name)
}

func (r *modelCatalogRepository) getVendor(ctx context.Context, where string, args ...any) (*service.ModelVendor, error) {
	var v service.ModelVendor
	err := r.db.QueryRowContext(ctx, "SELECT id, name, description, icon, status, created_at, updated_at FROM model_vendors WHERE "+where, args...).
		Scan(&v.ID, &v.Name, &v.Description, &v.Icon, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, service.ErrModelVendorNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *modelCatalogRepository) CreateVendor(ctx context.Context, vendor *service.ModelVendor) error {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO model_vendors (name, description, icon, status) VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		vendor.Name, vendor.Description, vendor.Icon, vendor.Status,
	).Scan(&vendor.ID, &vendor.CreatedAt, &vendor.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrModelVendorExists
		}
		return fmt.Errorf("create model vendor: %w", err)
	}
	return nil
}

func (r *modelCatalogRepository) UpdateVendor(ctx context.Context, vendor *service.ModelVendor) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE model_vendors SET name = $1, description = $2, icon = $3, status = $4, updated_at = NOW() WHERE id = $5`,
		vendor.Name, vendor.Description, vendor.Icon, vendor.Status, vendor.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrModelVendorExists
		}
		return fmt.Errorf("update model vendor: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrModelVendorNotFound
	}
	return nil
}

func (r *modelCatalogRepository) DeleteVendor(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM model_vendors WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete model vendor: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrModelVendorNotFound
	}
	return nil
}

func (r *modelCatalogRepository) ListModels(ctx context.Context, params pagination.PaginationParams, filters service.ModelCatalogFilters) ([]service.ModelCatalog, *pagination.PaginationResult, error) {
	where, args := []string{"m.deleted_at IS NULL"}, []any{}
	idx := 1
	if filters.Search != "" {
		where = append(where, fmt.Sprintf("(m.model_name ILIKE $%d OR m.description ILIKE $%d OR m.tags ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+escapeLike(filters.Search)+"%")
		idx++
	}
	if filters.VendorID != nil {
		where = append(where, fmt.Sprintf("m.vendor_id = $%d", idx))
		args = append(args, *filters.VendorID)
		idx++
	}
	if filters.Status != "" {
		where = append(where, fmt.Sprintf("m.status = $%d", idx))
		args = append(args, filters.Status)
		idx++
	}
	if filters.NameRule != nil {
		where = append(where, fmt.Sprintf("m.name_rule = $%d", idx))
		args = append(args, *filters.NameRule)
		idx++
	}
	clause := strings.Join(where, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_catalog m WHERE "+clause, args...).Scan(&total); err != nil {
		return nil, nil, fmt.Errorf("count models: %w", err)
	}
	limit := params.Limit()
	offset := params.Offset()
	query := fmt.Sprintf(`SELECT m.id, m.model_name, m.description, m.icon, m.tags, m.vendor_id, COALESCE(v.name, ''), COALESCE(v.icon, ''),
		m.endpoints, m.status, m.sync_official, m.name_rule, m.created_at, m.updated_at
		FROM model_catalog m LEFT JOIN model_vendors v ON v.id = m.vendor_id
		WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`, clause, modelCatalogOrderBy(params), idx, idx+1)
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("list models: %w", err)
	}
	defer rows.Close()
	models, err := scanModels(rows)
	return models, pageResult(total, params.Page, limit), err
}

func (r *modelCatalogRepository) GetModelByID(ctx context.Context, id int64) (*service.ModelCatalog, error) {
	return r.getModel(ctx, "m.id = $1", id)
}

func (r *modelCatalogRepository) GetModelByName(ctx context.Context, name string) (*service.ModelCatalog, error) {
	return r.getModel(ctx, "LOWER(m.model_name) = LOWER($1)", name)
}

func (r *modelCatalogRepository) getModel(ctx context.Context, where string, args ...any) (*service.ModelCatalog, error) {
	row := r.db.QueryRowContext(ctx, `SELECT m.id, m.model_name, m.description, m.icon, m.tags, m.vendor_id, COALESCE(v.name, ''), COALESCE(v.icon, ''),
		m.endpoints, m.status, m.sync_official, m.name_rule, m.created_at, m.updated_at
		FROM model_catalog m LEFT JOIN model_vendors v ON v.id = m.vendor_id
		WHERE m.deleted_at IS NULL AND `+where, args...)
	m, err := scanModel(row)
	if err == sql.ErrNoRows {
		return nil, service.ErrModelNotFound
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *modelCatalogRepository) CreateModel(ctx context.Context, model *service.ModelCatalog) error {
	endpoints, _ := json.Marshal(model.Endpoints)
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO model_catalog (model_name, description, icon, tags, vendor_id, endpoints, status, sync_official, name_rule)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at, updated_at`,
		model.ModelName, model.Description, model.Icon, model.Tags, model.VendorID, endpoints, model.Status, model.SyncOfficial, model.NameRule,
	).Scan(&model.ID, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrModelExists
		}
		return fmt.Errorf("create model: %w", err)
	}
	return nil
}

func (r *modelCatalogRepository) UpdateModel(ctx context.Context, model *service.ModelCatalog) error {
	endpoints, _ := json.Marshal(model.Endpoints)
	result, err := r.db.ExecContext(ctx,
		`UPDATE model_catalog SET model_name = $1, description = $2, icon = $3, tags = $4, vendor_id = $5, endpoints = $6,
		 status = $7, sync_official = $8, name_rule = $9, updated_at = NOW() WHERE id = $10 AND deleted_at IS NULL`,
		model.ModelName, model.Description, model.Icon, model.Tags, model.VendorID, endpoints, model.Status, model.SyncOfficial, model.NameRule, model.ID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrModelExists
		}
		return fmt.Errorf("update model: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrModelNotFound
	}
	return nil
}

func (r *modelCatalogRepository) UpdateModelStatus(ctx context.Context, id int64, status string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE model_catalog SET status = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`, status, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrModelNotFound
	}
	return nil
}

func (r *modelCatalogRepository) DeleteModel(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE model_catalog SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrModelNotFound
	}
	return nil
}

func (r *modelCatalogRepository) BatchDeleteModels(ctx context.Context, ids []int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE model_catalog SET deleted_at = NOW(), updated_at = NOW() WHERE id = ANY($1) AND deleted_at IS NULL`, pq.Array(ids))
	return err
}

func (r *modelCatalogRepository) ExistingModelNames(ctx context.Context) (map[string]service.ModelCatalog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT m.id, m.model_name, m.description, m.icon, m.tags, m.vendor_id, COALESCE(v.name, ''), COALESCE(v.icon, ''),
		m.endpoints, m.status, m.sync_official, m.name_rule, m.created_at, m.updated_at
		FROM model_catalog m LEFT JOIN model_vendors v ON v.id = m.vendor_id WHERE m.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	models, err := scanModels(rows)
	if err != nil {
		return nil, err
	}
	out := make(map[string]service.ModelCatalog, len(models))
	for _, m := range models {
		out[strings.ToLower(m.ModelName)] = m
	}
	return out, nil
}

func (r *modelCatalogRepository) FindReferencedModels(ctx context.Context) ([]service.MissingModel, error) {
	found := map[string]*service.MissingModel{}
	rows, err := r.db.QueryContext(ctx, `SELECT c.id, c.name, p.platform, p.models FROM channel_model_pricing p JOIN channels c ON c.id = p.channel_id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var channelID int64
		var channel, platform string
		var raw []byte
		if err := rows.Scan(&channelID, &channel, &platform, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		var models []string
		_ = json.Unmarshal(raw, &models)
		for _, m := range models {
			addReferenced(found, m, "pricing:"+channel, channelID, channel, platform)
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	rows, err = r.db.QueryContext(ctx, `SELECT id, name, model_mapping FROM channels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var channelID int64
		var channel string
		var raw []byte
		if err := rows.Scan(&channelID, &channel, &raw); err != nil {
			return nil, err
		}
		var nested map[string]map[string]string
		if err := json.Unmarshal(raw, &nested); err == nil {
			for platform, mapping := range nested {
				for src, dst := range mapping {
					addReferenced(found, src, "mapping-source:"+channel+":"+platform, channelID, channel, platform)
					addReferenced(found, dst, "mapping-target:"+channel+":"+platform, channelID, channel, platform)
				}
			}
			continue
		}
		var flat map[string]string
		if err := json.Unmarshal(raw, &flat); err == nil {
			for src, dst := range flat {
				addReferenced(found, src, "mapping-source:"+channel, channelID, channel, "")
				addReferenced(found, dst, "mapping-target:"+channel, channelID, channel, "")
			}
		}
	}
	rows.Close()
	rows, err = r.db.QueryContext(ctx, `
		SELECT a.id, a.name, COALESCE(g.platform, ''), kv.key, kv.value
		FROM accounts a
		JOIN account_groups ag ON ag.account_id = a.id
		JOIN groups g ON g.id = ag.group_id
		CROSS JOIN LATERAL jsonb_each_text(COALESCE(a.credentials->'model_mapping', '{}'::jsonb)) AS kv(key, value)
		WHERE a.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var accountID int64
		var account, platform, src, dst string
		if err := rows.Scan(&accountID, &account, &platform, &src, &dst); err != nil {
			return nil, err
		}
		addReferenced(found, src, "account-mapping-source:"+account, accountID, account, platform)
		addReferenced(found, dst, "account-mapping-target:"+account, accountID, account, platform)
	}
	out := make([]service.MissingModel, 0, len(found))
	for _, item := range found {
		sort.Strings(item.Sources)
		sort.Strings(item.Platforms)
		sort.Slice(item.Channels, func(i, j int) bool {
			if item.Channels[i].Name == item.Channels[j].Name {
				return item.Channels[i].Platform < item.Channels[j].Platform
			}
			return item.Channels[i].Name < item.Channels[j].Name
		})
		item.MatchedCount = len(item.Sources)
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *modelCatalogRepository) EnrichModels(ctx context.Context, models []service.ModelCatalog) ([]service.ModelCatalog, error) {
	if len(models) == 0 {
		return models, nil
	}
	referenced, err := r.FindReferencedModels(ctx)
	if err != nil {
		return nil, err
	}
	referencedNames := make([]string, 0, len(referenced))
	for _, ref := range referenced {
		referencedNames = append(referencedNames, ref.ModelName)
	}
	for i := range models {
		matched := matchModelRule(models[i], referencedNames)
		models[i].MatchedModels = matched
		models[i].MatchedCount = len(matched)
		channels, groups, quotaTypes, accountCount, accountGroups, err := r.modelBindings(ctx, models[i], matched)
		if err != nil {
			return nil, err
		}
		models[i].BoundChannels = channels
		models[i].EnableGroups = groups
		models[i].QuotaTypes = quotaTypes
		models[i].AccountCount = accountCount
		models[i].AvailableGroups = accountGroups
	}
	return models, nil
}

func (r *modelCatalogRepository) modelBindings(ctx context.Context, model service.ModelCatalog, matched []string) ([]service.ModelBoundChannel, []string, []string, int, []string, error) {
	names := matched
	if len(names) == 0 && model.NameRule == service.ModelNameRuleExact {
		names = []string{model.ModelName}
	}
	if len(names) == 0 {
		return nil, nil, nil, 0, nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT c.id, c.name, p.platform, cg.group_id, COALESCE(g.name, '')
		FROM channel_model_pricing p
		JOIN channels c ON c.id = p.channel_id
		LEFT JOIN channel_groups cg ON cg.channel_id = c.id
		LEFT JOIN groups g ON g.id = cg.group_id
	WHERE p.models ?| $1::text[]`, pq.Array(names))
	if err != nil {
		return nil, nil, nil, 0, nil, err
	}
	defer rows.Close()
	byID := map[int64]*service.ModelBoundChannel{}
	groupSet := map[string]struct{}{}
	quotaSet := map[string]struct{}{}
	for rows.Next() {
		var id, gid sql.NullInt64
		var name, platform, groupName string
		if err := rows.Scan(&id, &name, &platform, &gid, &groupName); err != nil {
			return nil, nil, nil, 0, nil, err
		}
		ch := byID[id.Int64]
		if ch == nil {
			ch = &service.ModelBoundChannel{ID: id.Int64, Name: name, Platform: platform}
			byID[id.Int64] = ch
		}
		if gid.Valid {
			ch.GroupIDs = append(ch.GroupIDs, gid.Int64)
		}
		if groupName != "" {
			ch.Groups = append(ch.Groups, groupName)
			groupSet[groupName] = struct{}{}
		}
		if platform != "" {
			quotaSet[platform] = struct{}{}
		}
	}
	channels := make([]service.ModelBoundChannel, 0, len(byID))
	for _, ch := range byID {
		channels = append(channels, *ch)
	}
	sort.Slice(channels, func(i, j int) bool { return channels[i].Name < channels[j].Name })
	if err := rows.Err(); err != nil {
		return nil, nil, nil, 0, nil, err
	}
	accountCount, accountGroups, err := r.accountCapabilityBindings(ctx, names)
	if err != nil {
		return nil, nil, nil, 0, nil, err
	}
	for _, g := range accountGroups {
		groupSet[g] = struct{}{}
	}
	return channels, setToSorted(groupSet), setToSorted(quotaSet), accountCount, accountGroups, nil
}

func (r *modelCatalogRepository) accountCapabilityBindings(ctx context.Context, names []string) (int, []string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT COUNT(DISTINCT amc.account_id), COALESCE(ARRAY_AGG(DISTINCT g.name) FILTER (WHERE g.name IS NOT NULL), ARRAY[]::text[])
		FROM account_model_capabilities amc
		JOIN accounts a ON a.id = amc.account_id
		JOIN groups g ON g.id = amc.group_id
		WHERE amc.status = 'active' AND a.status = 'active' AND a.schedulable = TRUE AND amc.model_name = ANY($1)`, pq.Array(names))
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var count int
		var groups []string
		if err := rows.Scan(&count, pq.Array(&groups)); err != nil {
			return 0, nil, err
		}
		sort.Strings(groups)
		return count, groups, nil
	}
	return 0, nil, rows.Err()
}

type modelScanner interface {
	Scan(dest ...any) error
}

func scanModel(row modelScanner) (*service.ModelCatalog, error) {
	var m service.ModelCatalog
	var vendorID sql.NullInt64
	var endpointsRaw []byte
	if err := row.Scan(&m.ID, &m.ModelName, &m.Description, &m.Icon, &m.Tags, &vendorID, &m.VendorName, &m.VendorIcon, &endpointsRaw, &m.Status, &m.SyncOfficial, &m.NameRule, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	if vendorID.Valid {
		m.VendorID = &vendorID.Int64
	}
	_ = json.Unmarshal(endpointsRaw, &m.Endpoints)
	if m.Endpoints == nil {
		m.Endpoints = []string{}
	}
	return &m, nil
}

func scanModels(rows *sql.Rows) ([]service.ModelCatalog, error) {
	var models []service.ModelCatalog
	for rows.Next() {
		m, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, *m)
	}
	return models, rows.Err()
}

func addReferenced(found map[string]*service.MissingModel, raw, source string, channelID int64, channelName, platform string) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return
	}
	rule := service.ModelNameRuleExact
	if strings.HasSuffix(name, "*") {
		name = strings.TrimSuffix(name, "*")
		rule = service.ModelNameRulePrefix
	}
	if name == "" {
		return
	}
	key := strings.ToLower(name)
	item := found[key]
	if item == nil {
		item = &service.MissingModel{ModelName: name, NameRule: rule}
		found[key] = item
	}
	platform = strings.TrimSpace(platform)
	if platform != "" && !containsString(item.Platforms, platform) {
		item.Platforms = append(item.Platforms, platform)
	}
	if channelID > 0 && !containsChannel(item.Channels, channelID, platform) {
		item.Channels = append(item.Channels, service.ModelBoundChannel{ID: channelID, Name: channelName, Platform: platform})
	}
	for _, existing := range item.Sources {
		if existing == source {
			return
		}
	}
	item.Sources = append(item.Sources, source)
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func containsChannel(channels []service.ModelBoundChannel, id int64, platform string) bool {
	for _, ch := range channels {
		if ch.ID == id && ch.Platform == platform {
			return true
		}
	}
	return false
}

func matchModelRule(model service.ModelCatalog, names []string) []string {
	var matched []string
	needle := strings.ToLower(model.ModelName)
	for _, name := range names {
		lower := strings.ToLower(name)
		ok := false
		switch model.NameRule {
		case service.ModelNameRulePrefix:
			ok = strings.HasPrefix(lower, needle)
		case service.ModelNameRuleContains:
			ok = strings.Contains(lower, needle)
		case service.ModelNameRuleSuffix:
			ok = strings.HasSuffix(lower, needle)
		default:
			ok = lower == needle
		}
		if ok {
			matched = append(matched, name)
		}
	}
	sort.Strings(matched)
	return matched
}

func modelCatalogOrderBy(params pagination.PaginationParams) string {
	order := params.NormalizedSortOrder(pagination.SortOrderDesc)
	switch params.SortBy {
	case "model_name":
		return "m.model_name " + order
	case "status":
		return "m.status " + order + ", m.id DESC"
	case "vendor":
		return "v.name " + order + " NULLS LAST, m.id DESC"
	case "updated_at":
		return "m.updated_at " + order
	default:
		return "m.created_at " + order + ", m.id DESC"
	}
}

func pageResult(total int64, page, pageSize int) *pagination.PaginationResult {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}
	return &pagination.PaginationResult{Total: total, Page: page, PageSize: pageSize, Pages: pages}
}

func setToSorted(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
