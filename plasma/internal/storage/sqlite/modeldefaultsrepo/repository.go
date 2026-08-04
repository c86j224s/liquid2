package modeldefaultsrepo

import (
	"context"
	"database/sql"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	settingWorkflowGoalModel           = "model_defaults.workflow_goal.model"
	settingWorkflowGoalReasoningEffort = "model_defaults.workflow_goal.reasoning_effort"
)

// Repository executes model-default SQL against a caller-owned DB.
type Repository struct {
	db *sql.DB
}

// New binds model-default persistence to an existing SQLite connection pool.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetModelDefaults reads configured model defaults.
func (r *Repository) GetModelDefaults(ctx context.Context) (app.ModelDefaults, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT key, value
FROM plasma_app_settings
WHERE key IN (?, ?)`,
		settingWorkflowGoalModel,
		settingWorkflowGoalReasoningEffort)
	if err != nil {
		return app.ModelDefaults{}, err
	}
	defer rows.Close()

	var defaults app.ModelDefaults
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return app.ModelDefaults{}, err
		}
		switch key {
		case settingWorkflowGoalModel:
			defaults.WorkflowGoalModel = value
		case settingWorkflowGoalReasoningEffort:
			defaults.WorkflowGoalReasoningEffort = value
		}
	}
	return defaults, rows.Err()
}

// SaveModelDefaults stores configured model defaults in one transaction.
func (r *Repository) SaveModelDefaults(ctx context.Context, defaults app.ModelDefaults) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, setting := range []struct {
		key   string
		value string
	}{
		{settingWorkflowGoalModel, defaults.WorkflowGoalModel},
		{settingWorkflowGoalReasoningEffort, defaults.WorkflowGoalReasoningEffort},
	} {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO plasma_app_settings (key, value, updated_at)
VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
ON CONFLICT(key) DO UPDATE SET
  value = excluded.value,
  updated_at = excluded.updated_at`,
			setting.key, setting.value); err != nil {
			return err
		}
	}
	return tx.Commit()
}
