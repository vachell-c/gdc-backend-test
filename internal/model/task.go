package model

import (
	"time"

	"github.com/google/uuid"
)

// Task represents a task in the system.
type Task struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateTaskRequest is the payload for creating a new task.
type CreateTaskRequest struct {
	Title       string  `json:"title" validate:"required"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// UpdateTaskRequest is the payload for updating an existing task.
type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Status      *string `json:"status,omitempty"`
}

// TaskResponse wraps a single task in the response body.
type TaskResponse struct {
	Data Task `json:"data"`
}

// TaskListResponse wraps a list of tasks with pagination metadata.
type TaskListResponse struct {
	Data []Task `json:"data"`
	Meta Meta   `json:"meta"`
}

// Meta contains pagination information.
type Meta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// AssignRequest is the payload for assigning a task to a user.
type AssignRequest struct {
	UserID uuid.UUID `json:"user_id"`
}
