package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sq "github.com/Masterminds/squirrel"
	"github.com/vasti/gdc-backend-test/internal/model"
)

// UserRepository handles database operations for users.
type UserRepository struct {
	pool *pgxpool.Pool
	sb   sq.StatementBuilderType
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// Create inserts a new user into the database and populates ID, CreatedAt, UpdatedAt.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	sqlStr, args, err := r.sb.Insert("users").
		Columns("email", "password", "name", "team_id").
		Values(user.Email, user.Password, user.Name, user.TeamID).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// FindByEmail looks up a user by email address. Returns nil, nil if not found.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	sqlStr, args, err := r.sb.Select("id", "email", "password", "name", "team_id", "created_at", "updated_at").
		From("users").
		Where(sq.Eq{"email": email}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var user model.User
	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name, &user.TeamID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

// FindByID looks up a user by UUID. Returns nil, nil if not found.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	sqlStr, args, err := r.sb.Select("id", "email", "password", "name", "team_id", "created_at", "updated_at").
		From("users").
		Where(sq.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var user model.User
	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(
		&user.ID, &user.Email, &user.Password, &user.Name, &user.TeamID, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return &user, nil
}
