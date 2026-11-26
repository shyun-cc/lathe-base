package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

type InputError struct{ Message string }

func (e InputError) Error() string { return e.Message }
func Invalid(message string) error { return InputError{Message: message} }

type ConflictError struct{ Message string }

func (e ConflictError) Error() string { return e.Message }
func Conflict(message string) error   { return ConflictError{Message: message} }

type ID struct {
	ID int64 `json:"id"`
}

type Queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func Insert(ctx context.Context, db Queryer, query string, args ...any) (ID, error) {
	var id ID
	err := db.QueryRowContext(ctx, query, args...).Scan(&id.ID)
	return id, err
}

func JSON(ctx context.Context, db Queryer, query string, args ...any) (json.RawMessage, error) {
	var data json.RawMessage
	err := db.QueryRowContext(ctx, query, args...).Scan(&data)
	return data, err
}

func Transaction[T any](ctx context.Context, db *sql.DB, write func(*sql.Tx) (T, error)) (T, error) {
	var zero T
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	result, err := write(tx)
	if err != nil {
		return zero, err
	}
	if err := tx.Commit(); err != nil {
		return zero, err
	}
	return result, nil
}

// RequireReference turns a missing or inactive business reference into an input error.
func RequireReference(err error, message string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return Invalid(message)
	}
	return err
}

// List wraps a bounded query ordered by a unique id for cursor pagination.
func List(db Queryer, query string) func(context.Context, int64, int64) (json.RawMessage, error) {
	return func(ctx context.Context, after, limit int64) (json.RawMessage, error) {
		return JSON(ctx, db, `SELECT coalesce(jsonb_agg(page ORDER BY id), '[]') FROM (`+query+`) page`, after, limit)
	}
}
