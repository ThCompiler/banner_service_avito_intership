package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Run(configPath string) {
	runtime, cleanup, err := BuildRuntime(context.Background(), configPath)
	if err != nil {
		log.Fatalf("[App] Init - build runtime error: %s", err) //nolint:revive // fatal init error
	}

	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			log.Printf("[App] Cleanup - dependencies: %v", cleanupErr)
		}
	}()

	l := runtime.Logger

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	l.Info("[App] Start - server started")

	select {
	case s := <-interrupt:
		l.Info("[App] Run - signal: " + s.String())
	case err = <-runtime.Server.Notify():
		l.Error(fmt.Errorf("[App] Run - httpServer.Notify: %w", err))
	}

	// Shutdown
	err = runtime.Server.Shutdown()
	if err != nil {
		l.Error(fmt.Errorf("[App] Stop - httpServer.Shutdown: %w", err))

		return
	}

	l.Info("[App] Stop - server stopped")
}
