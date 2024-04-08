package pg

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func WithTransaction(db *sqlx.DB, transaction func(tx *sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return errors.Wrap(err, "can't begin transaction")
	}

	if err := transaction(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "can't commit transaction")
	}

	return nil
}
