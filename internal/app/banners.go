package app

import (
	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	au "bannersrv/external/auth/usecase"
	"bannersrv/internal/app/config"
	"bannersrv/internal/pkg/metrics/prometheus"
	"bannersrv/pkg/logger"
	"bannersrv/pkg/server"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	v1 "bannersrv/internal/app/delivery/http/v1"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	bp "bannersrv/internal/banner/repository/postgres"
	bu "bannersrv/internal/banner/usecase"
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type databases struct {
	pg  *pgxpool.Pool
	rds *redis.Client
}

func initDatabases(cfg *config.Config, l logger.Interface) *databases {
	// Postgres
	cfx, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		l.Fatal("[App] Init - postgres.New: %s", err)
	}

	cfx.MaxConns = int32(cfg.Postgres.MaxConnections)
	cfx.MinConns = int32(cfg.Postgres.MinConnections)
	cfx.MaxConnIdleTime = time.Duration(cfg.Postgres.TTLIDleConnections) * time.Millisecond

	pg, err := pgxpool.NewWithConfig(context.Background(), cfx)
	if err != nil {
		l.Fatal("[App] Init - postgres.New: %s", err)
	}

	if err = pg.Ping(context.Background()); err != nil {
		pg.Close()
		l.Fatal("[App] Init - can't check connection to sql with error %s", err)
	}

	l.Info("[App] Init - success check connection to postgresql")

	// Redis
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		pg.Close()
		l.Fatal("[App] Init  - redis - redis.New: %s", err)
	}

	rds := redis.NewClient(opt)

	if err = rds.Ping(context.Background()).Err(); err != nil {
		pg.Close()
		l.Fatal("[App] Init - can't check connection to redis with error: %s", err)
	}

	l.Info("[App] Init - success check connection to redis")

	return &databases{
		pg:  pg,
		rds: rds,
	}
}

func initRoutes(mode config.Mode, dbs *databases, l logger.Interface) (*gin.Engine, error) {
	// metrics
	metricsManager := prometheus.NewPrometheusMetrics("main")
	if err := metricsManager.SetupMonitoring(); err != nil {
		l.Fatal("[App] Init - can't register metrics: %s", err)
	}

	// Repository
	bannerRepository := bp.NewBannerRepository(dbs.pg)
	cacheRepository := cr.NewCashRedis(dbs.rds)

	// Use-cases
	bannerUsecase := bu.NewBannerUsecase(bannerRepository)
	cacheManager := cm.NewCacheManager(cacheRepository)
	authService := au.NewAuthUsecase()

	// Handlers
	bannerHandlers := bh.NewBannerHandlers(bannerUsecase, cacheManager)
	authHandlers := ah.NewAuthHandlers(authService)

	// routes
	routes := PrepareRoutes(bannerHandlers, cacheManager, authService, authHandlers)

	return v1.NewRouter("/api", routes, mode, l, metricsManager)
}

func Run(cfg *config.Config) {
	// Logger
	l, logFile := prepareLogger(cfg.LoggerInfo)

	defer func() {
		_ = l.Sync() // nolint: errcheck //нет смысла логировать ошибку, при выключении сервер

		if logFile != nil {
			_ = logFile.Close() // nolint: errcheck // нет смысла логировать ошибку закрытия лога,
			// при выключении сервера
		}
	}()

	// Databases
	dbs := initDatabases(cfg, l)
	defer dbs.pg.Close()

	// Routes
	router, err := initRoutes(cfg.Mode, dbs, l)
	if err != nil {
		l.Fatal("[App] Init - init handler error: %s", err)
	}

	httpServer := server.New(router, server.Port(cfg.Port))

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	l.Info("[App] Start - server started")

	select {
	case s := <-interrupt:
		l.Info("[App] Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		l.Error(fmt.Errorf("[App] Run - httpServer.Notify: %w", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Fatal(fmt.Errorf("[App] Stop - httpServer.Shutdown: %w", err))
	}

	l.Info("[App] Stop - server stopped")
}
