package app

import (
	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	au "bannersrv/external/auth/usecase"
	"bannersrv/internal/app/config"
	v1 "bannersrv/internal/app/delivery/http/v1"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	bp "bannersrv/internal/banner/repository/postgres"
	bu "bannersrv/internal/banner/usecase"
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"
	"bannersrv/internal/pkg/metrics/prometheus"
	"context"
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"github.com/redis/go-redis/v9"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"

	"bannersrv/pkg/server"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
)

func Run(cfg *config.Config) {
	// Logger
	l, logFile := prepareLogger(cfg.LoggerInfo)

	defer func() {
		if logFile != nil {
			_ = logFile.Close()
		}
		_ = l.Sync()
	}()

	// Postgres
	pg, err := sqlx.Open("pgx", cfg.Postgres.URL)
	if err != nil {
		l.Fatal("[App] Init - postgres.New: %s", err)
	}
	defer pg.Close()

	pg.SetConnMaxIdleTime(time.Duration(cfg.Postgres.TTLIdleConnections) * time.Millisecond)
	pg.SetMaxOpenConns(cfg.Postgres.MaxConnections)
	pg.SetMaxIdleConns(cfg.Postgres.MaxIdleConnections)

	if err := pg.Ping(); err != nil {
		l.Fatal("[App] Init - can't check connection to sql with error %s", err)
	}
	l.Info("[App] Init - success check connection to postgresql")

	// Redis
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		l.Fatal("[App] Init  - redis - redis.New: %s", err)
	}
	rds := redis.NewClient(opt)

	if err := rds.Ping(context.Background()).Err(); err != nil {
		l.Fatal("[App] Init - can't check connection to redis with error: %s", err)
	}

	l.Info("[App] Init - success check connection to redis")

	// Cron
	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		l.Fatal("[App] Init - start cronScheduler error: %s", err)
	}

	// metrics
	metricsManager := prometheus.NewPrometheusMetrics("main")
	if err := metricsManager.SetupMonitoring(); err != nil {
		l.Fatal("[App] Init - can't register metrics: %s", err)
	}

	// Repository
	bannerRepository, err := bp.NewBannerRepository(pg, cronScheduler, l)
	if err != nil {
		l.Fatal("[App] Init - initialize BannerRepository error: %s", err)
	}

	cacheRepository := cr.NewCashRedis(rds)

	// Use-cases
	bannerUsecase := bu.NewBannerUsecase(bannerRepository)
	cacheManager := cm.NewCacheManager(cacheRepository)
	authService := au.NewAuthUsecase()

	// Handlers
	bannerHandlers := bh.NewBannerHandlers(bannerUsecase, cacheManager)
	authHandlers := ah.NewAuthHandlers(authService)

	// routes
	routes := prepareRoutes(bannerHandlers, cacheManager, authService, authHandlers)
	router, err := v1.NewRouter("/api", routes, cfg.Mode, l, metricsManager)
	if err != nil {
		l.Fatal("[App] Init - init handler error: %s", err)
	}

	httpServer := server.New(router, server.Port(cfg.Port))

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	cronScheduler.Start()

	l.Info("[App] Start - server started")

	select {
	case s := <-interrupt:
		l.Info("[App] Run - signal: " + s.String())
	case err := <-httpServer.Notify():
		l.Error(fmt.Errorf("[App] Run - httpServer.Notify: %s", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Fatal(fmt.Errorf("[App] Stop - httpServer.Shutdown: %s", err))
	}

	err = cronScheduler.Shutdown()
	if err != nil {
		l.Fatal(fmt.Errorf("[App] Stop - cronScheduler.Shutdown: %s", err))
	}

	l.Info("[App] Stop - server stopped")
}
