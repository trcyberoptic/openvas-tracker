package queries

import (
	"context"
	"time"
)

type RiskAcceptRule struct {
	ID          string     `json:"id"`
	Fingerprint string     `json:"fingerprint"`
	HostPattern string     `json:"host_pattern"`
	Reason      string     `json:"reason"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateRiskAcceptRuleParams struct {
	ID          string
	Fingerprint string
	HostPattern string
	Reason      string
	ExpiresAt   *time.Time
	CreatedBy   string
}

func (q *Queries) CreateRiskAcceptRule(ctx context.Context, arg CreateRiskAcceptRuleParams) error {
	_, err := q.db.ExecContext(ctx, `INSERT INTO risk_accept_rules (id, fingerprint, host_pattern, reason, expires_at, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
		arg.ID, arg.Fingerprint, arg.HostPattern, arg.Reason, arg.ExpiresAt, arg.CreatedBy)
	return err
}

func (q *Queries) ListRiskAcceptRules(ctx context.Context) ([]RiskAcceptRule, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, fingerprint, host_pattern, reason, expires_at, created_by, created_at FROM risk_accept_rules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RiskAcceptRule
	for rows.Next() {
		var r RiskAcceptRule
		if err := rows.Scan(&r.ID, &r.Fingerprint, &r.HostPattern, &r.Reason, &r.ExpiresAt, &r.CreatedBy, &r.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func (q *Queries) DeleteRiskAcceptRule(ctx context.Context, id string) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM risk_accept_rules WHERE id = ?`, id)
	return err
}

// MatchRiskAcceptRule checks if a finding (by fingerprint + host) matches any active rule.
func (q *Queries) MatchRiskAcceptRule(ctx context.Context, fingerprint, host string) (*RiskAcceptRule, error) {
	row := q.db.QueryRowContext(ctx, `SELECT id, fingerprint, host_pattern, reason, expires_at, created_by, created_at FROM risk_accept_rules WHERE fingerprint = ? AND (host_pattern = '*' OR host_pattern = ?) AND (expires_at IS NULL OR expires_at >= CURDATE()) ORDER BY created_at DESC LIMIT 1`, fingerprint, host)
	var r RiskAcceptRule
	if err := row.Scan(&r.ID, &r.Fingerprint, &r.HostPattern, &r.Reason, &r.ExpiresAt, &r.CreatedBy, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// ApplyRuleToExistingTickets sets matching open tickets to risk_accepted.
func (q *Queries) ApplyRuleToExistingTickets(ctx context.Context, fingerprint, hostPattern string, expiresAt *time.Time) ([]string, error) {
	var query string
	var args []any
	if hostPattern == "*" {
		query = "SELECT t.id FROM tickets t JOIN vulnerabilities v ON t.vulnerability_id = v.id WHERE t.status = 'open' AND (v.cve_id = ? OR (v.cve_id IS NULL AND CONCAT('title:', v.title) = ?))"
		args = []any{fingerprint, fingerprint}
	} else {
		query = "SELECT t.id FROM tickets t JOIN vulnerabilities v ON t.vulnerability_id = v.id WHERE t.status = 'open' AND v.affected_host = ? AND (v.cve_id = ? OR (v.cve_id IS NULL AND CONCAT('title:', v.title) = ?))"
		args = []any{hostPattern, fingerprint, fingerprint}
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, id := range ids {
		q.db.ExecContext(ctx, "UPDATE tickets SET status = 'risk_accepted', risk_accepted_until = ?, updated_at = NOW() WHERE id = ?", expiresAt, id)
	}
	return ids, nil
}
