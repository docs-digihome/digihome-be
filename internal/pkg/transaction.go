package pkg

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}
type txManager struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewTxManager(db *pgxpool.Pool, logger slog.Logger) TxManager {
	return &txManager{db: db, logger: &logger}
}

func (tm *txManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := tm.db.Begin(ctx)
	if err != nil {
		tm.logger.Error("begin transaction failed", "error", err)
		return err
	}

	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(ctx); err != nil {
				tm.logger.Error("failed to rollback transaction", "error", err)
			}
		}
	}()

	if err := fn(tx); err != nil {
		tm.logger.Error("fn inside transaction error", "error", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		tm.logger.Error("commit transaction failed", "error", err)
		return err
	}

	committed = true
	return nil
}
