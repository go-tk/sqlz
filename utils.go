package sqlz

import (
	"context"
	"database/sql"
	"fmt"
)

// BeginTx is the convenient version of `DB.BeginTx()`.
func BeginTx(ctx context.Context, db *sql.DB, txOptions *sql.TxOptions, returnedErr *error) (*sql.Tx, func(), error) {
	tx, err := db.BeginTx(ctx, txOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	closeTx := func() {
		if *returnedErr == nil {
			if err := tx.Commit(); err != nil {
				*returnedErr = fmt.Errorf("commit tx: %w", err)
			}
		} else {
			tx.Rollback()
		}
	}
	return tx, closeTx, nil
}
