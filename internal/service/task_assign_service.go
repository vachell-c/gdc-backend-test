package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/repository"
)

// TaskAssignService handles task assignment business logic.
type TaskAssignService struct {
	pool     *pgxpool.Pool
	userRepo *repository.UserRepository
	taskRepo *repository.TaskRepository
}

// NewTaskAssignService creates a new TaskAssignService.
func NewTaskAssignService(pool *pgxpool.Pool, userRepo *repository.UserRepository, taskRepo *repository.TaskRepository) *TaskAssignService {
	return &TaskAssignService{
		pool:     pool,
		userRepo: userRepo,
		taskRepo: taskRepo,
	}
}

// Assign assigns a task to a user within the same team.
func (s *TaskAssignService) Assign(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req model.AssignRequest) (*model.Task, error) {
	// 1. Get task by ID (must exist)
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, model.ErrInternal("failed to get task", err)
	}
	if task == nil {
		return nil, model.ErrNotFound("task not found", nil)
	}

	// 2. Verify userID is the task owner
	if task.UserID != userID {
		return nil, model.ErrForbidden("only the task owner can assign tasks", nil)
	}

	// 3. Get assignee user by ID (must exist)
	assignee, err := s.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, model.ErrInternal("failed to get assignee user", err)
	}
	if assignee == nil {
		return nil, model.ErrNotFound("assignee user not found", nil)
	}

	// 4. Verify same team: both user and assignee must have non-nil team_id, and they must match
	if task.UserID == req.UserID {
		return nil, model.ErrForbidden("cannot assign task to yourself", nil)
	}

	// Get the owner user to check team
	owner, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, model.ErrInternal("failed to get owner user", err)
	}
	if owner == nil {
		return nil, model.ErrNotFound("owner user not found", nil)
	}

	if owner.TeamID == nil || assignee.TeamID == nil {
		return nil, model.ErrForbidden("both users must belong to a team", nil)
	}
	if *owner.TeamID != *assignee.TeamID {
		return nil, model.ErrForbidden("users must be in the same team", nil)
	}

	// 5. Begin transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, model.ErrInternal("failed to begin transaction", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 6. Call AssignInTransaction
	if err := repository.AssignInTransaction(ctx, tx, taskID, req.UserID, userID); err != nil {
		return nil, model.ErrInternal("failed to assign task", err)
	}

	// 7. Log notification
	slog.Info("notification",
		"type", "task_assigned",
		"task_id", taskID,
		"assigned_by", userID,
		"assigned_to", req.UserID,
	)

	// 8. Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, model.ErrInternal("failed to commit transaction", err)
	}

	// 9. Re-fetch and return updated task
	updatedTask, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, model.ErrInternal("failed to fetch updated task", err)
	}
	if updatedTask == nil {
		return nil, model.ErrNotFound("task not found after assignment", nil)
	}

	return updatedTask, nil
}
