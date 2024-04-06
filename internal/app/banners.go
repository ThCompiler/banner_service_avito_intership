package app

import (
	ah "bannersrv/external/auth/delivery/http/v1/handlers"
	au "bannersrv/external/auth/usecase"
	v1 "bannersrv/internal/app/delivery/http/v1"
	bh "bannersrv/internal/banner/delivery/http/v1/handlers"
	bp "bannersrv/internal/banner/repository/postgres"
	bu "bannersrv/internal/banner/usecase"
	cm "bannersrv/internal/caches/manager"
	cr "bannersrv/internal/caches/repository/redis"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmoiron/sqlx"

	"bannersrv/config"

	"bannersrv/pkg/server"

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
	pg, err := sqlx.Open("postgres", cfg.Postgres.URL)
	if err != nil {
		l.Fatal("[App] Init - postgres.New: %s", err)
	}
	defer pg.Close()

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

	// Repository
	bannerRepository := bp.NewBannerRepository(pg)
	cacheRepository := cr.NewCashRedis(rds)

	// Use-cases
	bannerUsecase := bu.NewBannerUsecase(bannerRepository)
	cacheManager := cm.NewCacheManager(cacheRepository)
	authService := au.NewAuthUsecase()

	// Handlers
	bannerHandlers := bh.NewBannerHandlers(bannerUsecase, cacheManager)
	authHandlers := ah.NewAuthHandlers(authService)

	// routes
	router, err := v1.NewRouter("/api", l, prepareRoutes(bannerHandlers, cacheManager, authService, authHandlers))
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
	case err := <-httpServer.Notify():
		l.Error(fmt.Errorf("[App] Run - httpServer.Notify: %s", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Error(fmt.Errorf("[App] Stop - httpServer.Shutdown: %s", err))
	}

	l.Info("[App] Stop - server stopped")
}
