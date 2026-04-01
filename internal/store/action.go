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

// ActionCreateParams holds the fields for creating a new action.
type ActionCreateParams struct {
	SourceID        uuid.UUID
	Type            model.ActionType
	TargetURL       *string
	SigningSecret   *string
	ScriptBody      *string
	Config          json.RawMessage
	TransformScript *string
}

// ActionUpdateParams holds the fields for updating an action.
// nil fields are left unchanged (via COALESCE in SQL).
type ActionUpdateParams struct {
	TargetURL       *string
	SigningSecret   *string
	IsActive        *bool
	ScriptBody      *string
	Config          json.RawMessage
	TransformScript *string
}

const actionColumns = `id, source_id, type, target_url, script_body, signing_secret, config, transform_script, is_active, created_at, updated_at`

func scanAction(scan func(dest ...any) error) (model.Action, error) {
	var a model.Action
	err := scan(&a.ID, &a.SourceID, &a.Type, &a.TargetURL, &a.ScriptBody, &a.SigningSecret, &a.Config, &a.TransformScript, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func (s *ActionStore) Create(ctx context.Context, p ActionCreateParams) (*model.Action, error) {
	a, err := scanAction(s.pool.QueryRow(ctx,
		`INSERT INTO actions (source_id, type, target_url, signing_secret, script_body, config, transform_script)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING `+actionColumns,
		p.SourceID, p.Type, p.TargetURL, p.SigningSecret, p.ScriptBody, p.Config, p.TransformScript,
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

	actions := make([]model.Action, 0)
	for rows.Next() {
		a, err := scanAction(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (s *ActionStore) CountBySource(ctx context.Context, sourceID uuid.UUID) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM actions WHERE source_id = $1`, sourceID).Scan(&count)
	return count, err
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

func (s *ActionStore) Update(ctx context.Context, id uuid.UUID, p ActionUpdateParams) (*model.Action, error) {
	a, err := scanAction(s.pool.QueryRow(ctx,
		`UPDATE actions SET
			target_url       = COALESCE($2, target_url),
			signing_secret   = COALESCE($3, signing_secret),
			is_active        = COALESCE($4, is_active),
			script_body      = COALESCE($5, script_body),
			config           = COALESCE($6, config),
			transform_script = COALESCE($7, transform_script),
			updated_at       = $8
		 WHERE id = $1
		 RETURNING `+actionColumns,
		id, p.TargetURL, p.SigningSecret, p.IsActive, p.ScriptBody, p.Config, p.TransformScript, time.Now(),
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

	actions := make([]model.Action, 0)
	for rows.Next() {
		a, err := scanAction(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}
