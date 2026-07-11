package storage

import (
	"database/sql"
	"fmt"
)

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// WithTx runs fn inside a SQLite transaction and commits only when fn succeeds.
func (db *DB) WithTx(fn func(*sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
