package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sq "github.com/Masterminds/squirrel"
	"github.com/vasti/gdc-backend-test/internal/model"
)

// TaskRepository handles database operations for tasks.
type TaskRepository struct {
	pool *pgxpool.Pool
	sb   sq.StatementBuilderType
}

// NewTaskRepository creates a new TaskRepository.
func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{
		pool: pool,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// Create inserts a new task into the database and populates ID, CreatedAt, UpdatedAt.
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	sqlStr, args, err := r.sb.Insert("tasks").
		Columns("user_id", "assignee_id", "title", "description", "status").
		Values(task.UserID, task.AssigneeID, task.Title, task.Description, task.Status).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

// ListResult contains the results of a paginated task list query.
type ListResult struct {
	Tasks []model.Task
	Total int
}

// List retrieves paginated tasks with optional filters for user ID, status, and search text.
// search uses PostgreSQL full-text search via to_tsvector / plainto_tsquery.
func (r *TaskRepository) List(ctx context.Context, userID uuid.UUID, status, search string, page, limit int) (*ListResult, error) {
	// -- count query --
	countBuilder := r.sb.Select("COUNT(*)").From("tasks").Where(sq.Eq{"user_id": userID})
	if status != "" {
		countBuilder = countBuilder.Where(sq.Eq{"status": status})
	}
	if search != "" {
		fts := strings.TrimSpace(search)
		countBuilder = countBuilder.Where("to_tsvector('english', title) @@ plainto_tsquery('english', ?)", fts)
	}

	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build count query: %w", err)
	}

	var total int
	err = r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	// -- data query --
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	dataBuilder := r.sb.Select("id", "user_id", "assignee_id", "title", "description", "status", "created_at", "updated_at").
		From("tasks").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	if status != "" {
		dataBuilder = dataBuilder.Where(sq.Eq{"status": status})
	}
	if search != "" {
		fts := strings.TrimSpace(search)
		dataBuilder = dataBuilder.Where("to_tsvector('english', title) @@ plainto_tsquery('english', ?)", fts)
	}

	dataSQL, dataArgs, err := dataBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build data query: %w", err)
	}

	rows, err := r.pool.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		var desc *string
		err := rows.Scan(&t.ID, &t.UserID, &t.AssigneeID, &t.Title, &desc, &t.Status, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}
		t.Description = desc
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	if tasks == nil {
		tasks = []model.Task{}
	}

	return &ListResult{
		Tasks: tasks,
		Total: total,
	}, nil
}

// GetByID retrieves a single task by its ID.
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	sqlStr, args, err := r.sb.Select("id", "user_id", "assignee_id", "title", "description", "status", "created_at", "updated_at").
		From("tasks").
		Where(sq.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var t model.Task
	var desc *string
	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(
		&t.ID, &t.UserID, &t.AssigneeID, &t.Title, &desc, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task by id: %w", err)
	}
	t.Description = desc
	return &t, nil
}

// GetByIDAndUser retrieves a task by its ID and the owning user ID (for ownership checks).
func (r *TaskRepository) GetByIDAndUser(ctx context.Context, taskID, userID uuid.UUID) (*model.Task, error) {
	sqlStr, args, err := r.sb.Select("id", "user_id", "assignee_id", "title", "description", "status", "created_at", "updated_at").
		From("tasks").
		Where(sq.Eq{"id": taskID, "user_id": userID}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var t model.Task
	var desc *string
	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(
		&t.ID, &t.UserID, &t.AssigneeID, &t.Title, &desc, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get task by id and user: %w", err)
	}
	t.Description = desc
	return &t, nil
}

// Update applies partial updates to a task. Returns the updated updated_at timestamp.
func (r *TaskRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (time.Time, error) {
	// Ensure updated_at is always refreshed
	updates["updated_at"] = time.Now()

	sqlStr, args, err := r.sb.Update("tasks").
		SetMap(updates).
		Where(sq.Eq{"id": id}).
		Suffix("RETURNING updated_at").
		ToSql()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to build update query: %w", err)
	}

	var updatedAt time.Time
	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(&updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return time.Time{}, fmt.Errorf("task not found: %w", pgx.ErrNoRows)
		}
		return time.Time{}, fmt.Errorf("failed to update task: %w", err)
	}
	return updatedAt, nil
}

// Delete removes a task by its ID. Does not error if the task does not exist.
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	sqlStr, args, err := r.sb.Delete("tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build delete query: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}
