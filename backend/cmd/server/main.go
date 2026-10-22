package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/app"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/repository"
	"backend/internal/routes"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.RunMigration(db); err != nil {
		return err
	}
	container, err := app.New(db, cfg)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{
		Addr: ":" + cfg.Port, Handler: routes.NewHandler(container),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: cfg.RequestTimeout,
		WriteTimeout: 2 * cfg.RequestTimeout, IdleTimeout: 2 * cfg.RequestTimeout,
	}
	failures := make(chan error, 1)
	go func() { failures <- server.ListenAndServe() }()
	log.Printf("Agro-Shield listening on port %s", cfg.Port)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	sessions := repository.NewSessionRepository(db)
	for {
		select {
		case err := <-failures:
			if !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		case <-ticker.C:
			cleanup, cancel := context.WithTimeout(ctx, 10*time.Second)
			if err := sessions.DeleteExpired(cleanup); err != nil {
				log.Printf("remove expired sessions: %v", err)
			}
			cancel()
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			return server.Shutdown(shutdown)
		}
	}
}
