package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type accountModelCapabilityRepository struct {
	db *sql.DB
}

func NewAccountModelCapabilityRepository(db *sql.DB) service.AccountModelCapabilityRepository {
	return &accountModelCapabilityRepository{db: db}
}

func (r *accountModelCapabilityRepository) ReplaceForAccount(ctx context.Context, accountID int64, source string, caps []service.AccountModelCapabilityInput) error {
	if r == nil || r.db == nil || accountID <= 0 {
		return nil
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = service.AccountModelCapabilitySourceNewAPI
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE account_model_capabilities SET status = 'inactive', updated_at = NOW() WHERE account_id = $1 AND source = $2`, accountID, source); err != nil {
		return fmt.Errorf("deactivate account model capabilities: %w", err)
	}
	for _, cap := range caps {
		modelName := strings.TrimSpace(cap.ModelName)
		upstream := strings.TrimSpace(cap.UpstreamModelName)
		if modelName == "" || cap.GroupID <= 0 {
			continue
		}
		if upstream == "" {
			upstream = modelName
		}
		status := strings.TrimSpace(cap.Status)
		if status == "" {
			status = service.AccountModelCapabilityStatusActive
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO account_model_capabilities
				(account_id, group_id, model_name, upstream_model_name, provider, source, status, last_seen_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			ON CONFLICT (account_id, group_id, model_name, upstream_model_name, source)
			DO UPDATE SET provider = EXCLUDED.provider, status = EXCLUDED.status, last_seen_at = NOW(), updated_at = NOW()`,
			accountID, cap.GroupID, modelName, upstream, strings.TrimSpace(cap.Provider), source, status,
		); err != nil {
			return fmt.Errorf("upsert account model capability: %w", err)
		}
	}
	return tx.Commit()
}

func (r *accountModelCapabilityRepository) ListByGroupIDs(ctx context.Context, groupIDs []int64, search string, limit int) ([]service.GroupAvailableModel, error) {
	if r == nil || r.db == nil || len(groupIDs) == 0 {
		return []service.GroupAvailableModel{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	where := []string{"amc.status = 'active'", "a.status = 'active'", "a.schedulable = TRUE", "amc.group_id = ANY($1)"}
	args := []any{pq.Array(groupIDs)}
	idx := 2
	if q := strings.TrimSpace(search); q != "" {
		where = append(where, fmt.Sprintf("(amc.model_name ILIKE $%d OR amc.provider ILIKE $%d OR COALESCE(v.name, '') ILIKE $%d)", idx, idx, idx))
		args = append(args, "%"+escapeLike(q)+"%")
		idx++
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
		SELECT
			amc.model_name,
			COALESCE(NULLIF(MAX(amc.provider), ''), ''),
			COALESCE(MAX(v.name), ''),
			COALESCE(MAX(v.icon), ''),
			COUNT(DISTINCT amc.account_id),
			ARRAY_AGG(DISTINCT amc.group_id),
			ARRAY_AGG(DISTINCT g.name),
			COALESCE(BOOL_OR(p.id IS NOT NULL), FALSE) AS priced
		FROM account_model_capabilities amc
		JOIN accounts a ON a.id = amc.account_id
		JOIN groups g ON g.id = amc.group_id
		LEFT JOIN model_catalog mc ON LOWER(mc.model_name) = LOWER(amc.model_name) AND mc.deleted_at IS NULL
		LEFT JOIN model_vendors v ON v.id = mc.vendor_id
		LEFT JOIN channel_groups cg ON cg.group_id = amc.group_id
		LEFT JOIN channel_model_pricing p ON p.channel_id = cg.channel_id AND p.models ? amc.model_name
		WHERE %s
		GROUP BY amc.model_name
		ORDER BY amc.model_name ASC
		LIMIT $%d`, strings.Join(where, " AND "), idx)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list group available models: %w", err)
	}
	defer rows.Close()
	var out []service.GroupAvailableModel
	for rows.Next() {
		var item service.GroupAvailableModel
		if err := rows.Scan(&item.ModelName, &item.Provider, &item.VendorName, &item.VendorIcon, &item.AccountCount, pq.Array(&item.GroupIDs), pq.Array(&item.Groups), &item.Priced); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *accountModelCapabilityRepository) CountByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	out := map[int64]int{}
	if r == nil || r.db == nil || len(accountIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT account_id, COUNT(DISTINCT model_name)
		FROM account_model_capabilities
		WHERE status = 'active' AND account_id = ANY($1)
		GROUP BY account_id`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		out[id] = count
	}
	return out, rows.Err()
}

func (r *accountModelCapabilityRepository) SummariesByAccountIDs(ctx context.Context, accountIDs []int64) (map[int64]service.AccountModelCapabilitySummary, error) {
	out := map[int64]service.AccountModelCapabilitySummary{}
	if r == nil || r.db == nil || len(accountIDs) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT account_id, COUNT(DISTINCT model_name), MAX(last_seen_at)
		FROM account_model_capabilities
		WHERE status = 'active' AND account_id = ANY($1)
		GROUP BY account_id`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item service.AccountModelCapabilitySummary
		var lastSeen sql.NullTime
		if err := rows.Scan(&item.AccountID, &item.ModelCount, &lastSeen); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			seen := lastSeen.Time.UTC().Truncate(time.Microsecond)
			item.LastSeenAt = &seen
		}
		out[item.AccountID] = item
	}
	return out, rows.Err()
}
