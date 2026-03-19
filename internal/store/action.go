package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zachbroad/nitrohook/internal/model"
)

type ActionStore struct {
	pool *pgxpool.Pool
}

const actionColumns = `id, source_id, type, target_url, script_body, signing_secret, config, is_active, created_at, updated_at`

func scanAction(scan func(dest ...any) error) (model.Action, error) {
	var a model.Action
	err := scan(&a.ID, &a.SourceID, &a.Type, &a.TargetURL, &a.ScriptBody, &a.SigningSecret, &a.Config, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (s *ActionStore) Create(ctx context.Context, sourceID uuid.UUID, actionType model.ActionType, targetURL *string, signingSecret *string, scriptBody *string, config json.RawMessage) (*model.Action, error) {
	a, err := scanAction(s.pool.QueryRow(ctx,
		`INSERT INTO actions (source_id, type, target_url, signing_secret, script_body, config)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING `+actionColumns,
		sourceID, actionType, targetURL, signingSecret, scriptBody, config,
	).Scan)
	if err != nil {
		return nil, fmt.Errorf("create action: %w", err)
	}
	return &a, nil
}

func (s *ActionStore) List(ctx context.Context, sourceID uuid.UUID) ([]model.Action, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+actionColumns+` FROM actions WHERE source_id = $1 ORDER BY created_at DESC`,
		sourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer rows.Close()

	var actions []model.Action
	for rows.Next() {
		a, err := scanAction(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (s *ActionStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Action, error) {
	a, err := scanAction(s.pool.QueryRow(ctx,
		`SELECT `+actionColumns+` FROM actions WHERE id = $1`,
		id,
	).Scan)
	if err != nil {
		return nil, fmt.Errorf("get action: %w", err)
	}
	return &a, nil
}

func (s *ActionStore) Update(ctx context.Context, id uuid.UUID, targetURL *string, signingSecret *string, isActive *bool, scriptBody *string, config json.RawMessage) (*model.Action, error) {
	a, err := scanAction(s.pool.QueryRow(ctx,
		`UPDATE actions SET
			target_url     = COALESCE($2, target_url),
			signing_secret = COALESCE($3, signing_secret),
			is_active      = COALESCE($4, is_active),
			script_body    = COALESCE($5, script_body),
			config         = COALESCE($6, config),
			updated_at     = $7
		 WHERE id = $1
		 RETURNING `+actionColumns,
		id, targetURL, signingSecret, isActive, scriptBody, config, time.Now(),
	).Scan)
	if err != nil {
		return nil, fmt.Errorf("update action: %w", err)
	}
	return &a, nil
}

func (s *ActionStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM actions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete action: %w", err)
	}
	return nil
}

func (s *ActionStore) ListActiveBySource(ctx context.Context, sourceID uuid.UUID) ([]model.Action, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+actionColumns+` FROM actions WHERE source_id = $1 AND is_active = true`,
		sourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active actions: %w", err)
	}
	defer rows.Close()

	var actions []model.Action
	for rows.Next() {
		a, err := scanAction(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}
