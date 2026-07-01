// Package notifications provides CRUD operations for notification rules
// scoped to the tenant via Row Level Security. Rules map event types +
// severities to delivery channels (telegram, email, sms, dashboard).
package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sambawy01/Retail-Compliance-System/pkg/database"
)

// Channel enumerates supported delivery channels. The values must match the
// CHECK constraint in migration 005.
const (
	ChannelTelegram  = "telegram"
	ChannelEmail     = "email"
	ChannelSMS       = "sms"
	ChannelDashboard = "dashboard"
)

// Rule is a notification rule row.
type Rule struct {
	RuleID    uuid.UUID  `json:"rule_id"`
	OrgID     uuid.UUID  `json:"org_id"`
	EventType string     `json:"event_type"`
	Severity  string     `json:"severity"`
	Channel   string     `json:"channel"`
	Target    string     `json:"target"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CreateRuleInput is the payload for creating a rule.
type CreateRuleInput struct {
	EventType string `json:"event_type"`
	Severity  string `json:"severity"`
	Channel   string `json:"channel"`
	Target    string `json:"target"`
}

// UpdateRuleInput is the payload for updating a rule. Pointer fields are
// optional; nil fields leave the column unchanged.
type UpdateRuleInput struct {
	EventType *string `json:"event_type,omitempty"`
	Severity  *string `json:"severity,omitempty"`
	Channel   *string `json:"channel,omitempty"`
	Target    *string `json:"target,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

// ErrRuleNotFound is returned when a rule is not found within the tenant scope.
var ErrRuleNotFound = errors.New("notifications: rule not found")

// Service manages notification rules with tenant-scoped DB access.
type Service struct {
	pool *database.Pool
}

// New creates a notifications Service bound to the given pool.
func New(pool *database.Pool) *Service {
	return &Service{pool: pool}
}

// ListRules returns all notification rules for the tenant in context.
func (s *Service) ListRules(ctx context.Context) ([]Rule, error) {
	var rules []Rule
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			SELECT rule_id, org_id, event_type, severity, channel, target, enabled,
			       created_at, updated_at
			FROM notification_rules
			ORDER BY created_at`
		rows, err := tx.Query(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r Rule
			if err := rows.Scan(
				&r.RuleID, &r.OrgID, &r.EventType, &r.Severity, &r.Channel,
				&r.Target, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
			); err != nil {
				return err
			}
			rules = append(rules, r)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("notifications: list rules: %w", err)
	}
	return rules, nil
}

// CreateRule inserts a new rule scoped to the tenant in context.
func (s *Service) CreateRule(ctx context.Context, in CreateRuleInput) (Rule, error) {
	var r Rule
	err := database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			INSERT INTO notification_rules (org_id, event_type, severity, channel, target)
			VALUES (current_setting('app.current_org_id', true)::uuid, $1, $2, $3, $4)
			RETURNING rule_id, org_id, event_type, severity, channel, target, enabled,
			          created_at, updated_at`
		return tx.QueryRow(ctx, q, in.EventType, in.Severity, in.Channel, in.Target).Scan(
			&r.RuleID, &r.OrgID, &r.EventType, &r.Severity, &r.Channel,
			&r.Target, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
		)
	})
	if err != nil {
		return Rule{}, fmt.Errorf("notifications: create rule: %w", err)
	}
	return r, nil
}

// UpdateRule updates mutable rule fields.
func (s *Service) UpdateRule(ctx context.Context, ruleID string, in UpdateRuleInput) (Rule, error) {
	id, err := uuid.Parse(ruleID)
	if err != nil {
		return Rule{}, fmt.Errorf("notifications: invalid rule id: %w", err)
	}
	var r Rule
	err = database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `
			UPDATE notification_rules
			SET event_type = COALESCE($2, event_type),
			    severity   = COALESCE($3, severity),
			    channel    = COALESCE($4, channel),
			    target     = COALESCE($5, target),
			    enabled    = COALESCE($6, enabled),
			    updated_at = now()
			WHERE rule_id = $1
			RETURNING rule_id, org_id, event_type, severity, channel, target, enabled,
			          created_at, updated_at`
		return tx.QueryRow(ctx, q, id, in.EventType, in.Severity, in.Channel, in.Target, in.Enabled).Scan(
			&r.RuleID, &r.OrgID, &r.EventType, &r.Severity, &r.Channel,
			&r.Target, &r.Enabled, &r.CreatedAt, &r.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Rule{}, ErrRuleNotFound
	}
	if err != nil {
		return Rule{}, fmt.Errorf("notifications: update rule: %w", err)
	}
	return r, nil
}

// DeleteRule removes a rule within the tenant scope.
func (s *Service) DeleteRule(ctx context.Context, ruleID string) error {
	id, err := uuid.Parse(ruleID)
	if err != nil {
		return fmt.Errorf("notifications: invalid rule id: %w", err)
	}
	var tag pgconn.CommandTag
	err = database.TenantTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		const q = `DELETE FROM notification_rules WHERE rule_id = $1`
		var err error
		tag, err = tx.Exec(ctx, q, id)
		return err
	})
	if err != nil {
		return fmt.Errorf("notifications: delete rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRuleNotFound
	}
	return nil
}