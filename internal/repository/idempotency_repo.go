package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sq "github.com/Masterminds/squirrel"
)

// IdempotencyRecord represents a stored idempotency key entry.
type IdempotencyRecord struct {
	Key          uuid.UUID       `json:"key"`
	Method       string          `json:"method"`
	Path         string          `json:"path"`
	ResponseBody json.RawMessage `json:"response_body"`
	ResponseCode int             `json:"response_code"`
	CreatedAt    time.Time       `json:"created_at"`
}

// IdempotencyRepository handles database operations for idempotency keys.
type IdempotencyRepository struct {
	pool *pgxpool.Pool
	sb   sq.StatementBuilderType
}

// NewIdempotencyRepository creates a new IdempotencyRepository.
func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{
		pool: pool,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// Save inserts a new idempotency key. If the key already exists it does nothing (ON CONFLICT DO NOTHING).
func (r *IdempotencyRepository) Save(ctx context.Context, key uuid.UUID, method, path string, responseCode int, responseBody []byte) error {
	sqlStr, args, err := r.sb.Insert("idempotency_keys").
		Columns("key", "method", "path", "response_code", "response_body").
		Values(key, method, path, responseCode, json.RawMessage(responseBody)).
		Suffix("ON CONFLICT (key) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = r.pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to save idempotency key: %w", err)
	}
	return nil
}

// GetByKey retrieves an idempotency key record if it exists and was created within the last 24 hours.
// Returns nil, nil if not found or expired.
func (r *IdempotencyRepository) GetByKey(ctx context.Context, key uuid.UUID) (*IdempotencyRecord, error) {
	sqlStr, args, err := r.sb.Select("key", "method", "path", "response_body", "response_code", "created_at").
		From("idempotency_keys").
		Where(sq.Eq{"key": key}).
		Where("created_at > NOW() - INTERVAL '24 hours'").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	var rec IdempotencyRecord
	err = r.pool.QueryRow(ctx, sqlStr, args...).Scan(
		&rec.Key, &rec.Method, &rec.Path, &rec.ResponseBody, &rec.ResponseCode, &rec.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get idempotency key: %w", err)
	}
	return &rec, nil
}
