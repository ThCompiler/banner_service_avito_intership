package main

import (
	"bannersrv/internal/app/config"
	"bannersrv/internal/banner"
	bp "bannersrv/internal/banner/repository/postgres"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultTaskPeriod = 18000

func main() { // nolint: revive // this a small executable file and big length of function is possible
	var configPath string

	var period uint64

	flag.StringVar(&configPath, "config", "./config/localhost-config.yaml", "path to config file")
	flag.Uint64Var(&period, "period", defaultTaskPeriod, "task start period in seconds")
	flag.Parse()

	l := log.New(os.Stderr, "cron-service", log.LUTC)

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		l.Fatal(err)
	}

	// Cron
	cronScheduler, err := gocron.NewScheduler()
	if err != nil {
		l.Fatalf("INIT: start cronScheduler error: %s", err)
	}

	// Postgres
	conf, err := pgxpool.ParseConfig(cfg.Postgres.URL)
	if err != nil {
		l.Fatalf("INIT:- postgres.New: %s", err)
	}

	conf.MaxConns = int32(cfg.Postgres.MaxConnections)
	conf.MinConns = int32(cfg.Postgres.MinConnections)
	conf.MaxConnIdleTime = time.Duration(cfg.Postgres.TTLIDleConnections) * time.Millisecond

	pg, err := pgxpool.NewWithConfig(context.Background(), conf)
	if err != nil {
		l.Fatalf("INIT: - postgres.New: %s", err)
	}
	defer pg.Close()

	if err = pg.Ping(context.Background()); err != nil {
		l.Fatalf("INIT: - can't check connection to sql with error %s", err)
	}

	l.Println("INIT: success check connection to postgresql")

	// Repository
	bannerRepository := bp.NewBannerRepository(pg)

	if _, err = cronScheduler.NewJob(
		gocron.DurationJob(time.Duration(period)*time.Second),
		gocron.NewTask(
			func(rep banner.Repository, l *log.Logger) {
				if err := rep.CleanDeletedBanner(); err != nil {
					l.Printf("ERROR: %s", errors.Wrap(err, "in cron job of cleaning deleted banner"))
				}
				l.Println("INFO: deleted banner was cleaned by cron job")
			},
			bannerRepository,
			l,
		),
	); err != nil {
		l.Fatalf("INIT: setup gocron task: %s", err)
	}

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	cronScheduler.Start()

	l.Println("Start: service started")

	s := <-interrupt
	l.Printf("RUN - signal: %s", s.String())

	// Shutdown
	err = cronScheduler.Shutdown()
	if err != nil {
		l.Fatal(fmt.Errorf("STOP - cronScheduler.Shutdown: %w", err))
	}

	l.Println("Stop - service stopped")
}
