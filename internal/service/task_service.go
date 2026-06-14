package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/repository"
)

// TaskService handles business logic for task CRUD operations.
type TaskService struct {
	taskRepo *repository.TaskRepository
}

// NewTaskService creates a new TaskService.
func NewTaskService(taskRepo *repository.TaskRepository) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
	}
}

// Create adds a new task for the given user.
func (s *TaskService) Create(ctx context.Context, userID uuid.UUID, req model.CreateTaskRequest) (*model.Task, error) {
	if req.Title == "" {
		return nil, model.ErrValidation("title is required", nil)
	}

	status := "pending"
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	task := &model.Task{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, model.ErrInternal("failed to create task", err)
	}

	return task, nil
}

// List retrieves paginated tasks for the given user.
func (s *TaskService) List(ctx context.Context, userID uuid.UUID, status, search string, page, limit int) (*model.TaskListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	result, err := s.taskRepo.List(ctx, userID, status, search, page, limit)
	if err != nil {
		return nil, model.ErrInternal("failed to list tasks", err)
	}

	return &model.TaskListResponse{
		Data: result.Tasks,
		Meta: model.Meta{
			Page:  page,
			Limit: limit,
			Total: result.Total,
		},
	}, nil
}

// GetByID retrieves a single task by ID, verifying the user is the owner or assignee.
func (s *TaskService) GetByID(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*model.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, model.ErrInternal("failed to get task", err)
	}
	if task == nil {
		return nil, model.ErrNotFound("task not found", nil)
	}

	// Validate ownership: task must belong to userID OR userID must be assignee
	if task.UserID != userID {
		if task.AssigneeID == nil || *task.AssigneeID != userID {
			return nil, model.ErrForbidden("you do not have access to this task", nil)
		}
	}

	return task, nil
}

// Update applies partial updates to a task, validating ownership first.
func (s *TaskService) Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req model.UpdateTaskRequest) (*model.Task, error) {
	// Verify ownership: must be the task owner
	task, err := s.taskRepo.GetByIDAndUser(ctx, taskID, userID)
	if err != nil {
		return nil, model.ErrInternal("failed to get task", err)
	}
	if task == nil {
		return nil, model.ErrNotFound("task not found or not owned by user", nil)
	}

	// Build updates map with non-nil fields
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) == 0 {
		return task, nil
	}

	updatedAt, err := s.taskRepo.Update(ctx, taskID, updates)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("task not found", err)
		}
		return nil, model.ErrInternal("failed to update task", err)
	}

	// Re-fetch the task to return the full updated record
	updatedTask, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, model.ErrInternal("failed to fetch updated task", err)
	}
	if updatedTask == nil {
		return nil, model.ErrNotFound("task not found after update", nil)
	}
	// Preserve the updated_at returned by Update
	updatedTask.UpdatedAt = updatedAt

	return updatedTask, nil
}

// Delete removes a task, validating ownership first.
func (s *TaskService) Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	// Verify ownership
	task, err := s.taskRepo.GetByIDAndUser(ctx, taskID, userID)
	if err != nil {
		return model.ErrInternal("failed to get task", err)
	}
	if task == nil {
		return model.ErrNotFound("task not found or not owned by user", nil)
	}

	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return model.ErrInternal("failed to delete task", err)
	}

	return nil
}
