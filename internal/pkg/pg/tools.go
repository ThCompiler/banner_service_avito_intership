package pg

import (
	"context"
	"time"

	"bannersrv/pkg/logger"

	"github.com/ThCompiler/sdi"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type Config struct {
	URL                string
	MaxConnections     int
	MinConnections     int
	TTLIDleConnections uint64
}

type ProviderDeps struct {
	Config Config
	Logger logger.Interface
}

func NewPool(ctx context.Context, cfg Config, l logger.Interface) (*pgxpool.Pool, error) {
	cfx, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, errors.Wrap(err, "postgres parse config")
	}

	//nolint:gosec // G115: config bounds are trusted
	cfx.MaxConns = int32(cfg.MaxConnections)
	//nolint:gosec // G115: config bounds are trusted
	cfx.MinConns = int32(cfg.MinConnections)
	//nolint:gosec // G115
	cfx.MaxConnIdleTime = time.Duration(cfg.TTLIDleConnections) * time.Millisecond

	pool, err := pgxpool.NewWithConfig(ctx, cfx)
	if err != nil {
		return nil, errors.Wrap(err, "postgres create pool")
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, errors.Wrap(err, "check connection to sql")
	}

	l.Info("[App] Init - success check connection to postgresql")

	return pool, nil
}

func NewProvider() sdi.Provider[*pgxpool.Pool, ProviderDeps] {
	return sdi.ProviderFuncWithCleanup(
		func(ctx context.Context, deps ProviderDeps) (*pgxpool.Pool, error) {
			return NewPool(ctx, deps.Config, deps.Logger)
		},
		func(_ context.Context, pool *pgxpool.Pool) error {
			if pool != nil {
				pool.Close()
			}

			return nil
		},
	)
}

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
