package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

func WithTransaction(db *pgxpool.Pool, transaction func(tx pgx.Tx) error) error {
	tx, err := db.Begin(context.Background())
	if err != nil {
		return errors.Wrap(err, "can't begin transaction")
	}

	if err := transaction(tx); err != nil {
		errRollback := tx.Rollback(context.Background())
		if errRollback != nil {
			return errors.Wrapf(err, "can't rollback with error %s", errRollback)
		}

		return err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return errors.Wrapf(err, "can't commit transaction")
	}

	return nil
}
