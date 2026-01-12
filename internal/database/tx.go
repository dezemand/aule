package database

import (
	"context"
	"database/sql"
	"errors"
)

type TxManager struct {
	DB *sql.DB
}

func (t *TxManager) StartTx(ctx context.Context) (context.Context, error) {
	tx, err := t.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelDefault,
	})
	if err != nil {
		return nil, err
	}

	return context.WithValue(ctx, "tx", tx), nil
}

func (t *TxManager) EndTx(ctx context.Context, rollback bool) error {
	tx := GetTx(ctx)
	if tx == nil {
		return errors.New("no tx found")
	}

	// Rollback if an error occurred
	if ctx.Err() != nil || rollback {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return errors.New("rollbacked")
	}

	return tx.Commit()
}

func (t *TxManager) Tx(ctx context.Context, txFunc func(ctx context.Context) error) error {
	nextCtx, err := t.StartTx(ctx)
	if err != nil {
		return err
	}

	// Execute the tx
	err = txFunc(nextCtx)

	if endErr := t.EndTx(nextCtx, err != nil); endErr != nil {
		return endErr
	}

	return err
}

func GetTx(ctx context.Context) *sql.Tx {
	tx := ctx.Value("tx")
	if tx == nil {
		return nil
	}
	txT, ok := tx.(*sql.Tx)
	if !ok {
		return nil
	}
	return txT
}

func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	tx := GetTx(ctx)
	if tx != nil {
		return tx.QueryContext(ctx, query, args...)
	} else {
		return db.DB.QueryContext(ctx, query, args...)
	}
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	tx := GetTx(ctx)
	if tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	} else {
		return db.DB.QueryRowContext(ctx, query, args...)
	}
}
