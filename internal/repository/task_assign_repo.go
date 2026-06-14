package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AssignInTransaction assigns a task to a user and logs the assignment in a single transaction.
// It operates on an existing pgx.Tx; the caller is responsible for committing or rolling back.
func AssignInTransaction(ctx context.Context, tx pgx.Tx, taskID, assigneeID, changedBy uuid.UUID) error {
	// Step 1: Update the task's assignee
	result, err := tx.Exec(ctx,
		`UPDATE tasks SET assignee_id = $1, updated_at = NOW() WHERE id = $2`,
		assigneeID, taskID,
	)
	if err != nil {
		return fmt.Errorf("failed to update task assignee: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %w", pgx.ErrNoRows)
	}

	// Step 2: Insert an audit log entry
	oldValue := `null`
	newValue := fmt.Sprintf(`"%s"`, assigneeID.String())

	_, err = tx.Exec(ctx,
		`INSERT INTO task_logs (task_id, changed_by, action, old_value, new_value) VALUES ($1, $2, 'assigned', $3::jsonb, $4::jsonb)`,
		taskID, changedBy, oldValue, newValue,
	)
	if err != nil {
		return fmt.Errorf("failed to insert task log: %w", err)
	}

	return nil
}
