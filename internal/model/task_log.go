package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskLog represents an audit log entry for a task change.
type TaskLog struct {
	ID        uuid.UUID        `json:"id"`
	TaskID    uuid.UUID        `json:"task_id"`
	ChangedBy uuid.UUID        `json:"changed_by"`
	Action    string           `json:"action"`
	OldValue  *json.RawMessage `json:"old_value,omitempty"`
	NewValue  *json.RawMessage `json:"new_value,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}
