package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/firmaelectronica/expedientes-oj/internal/config"
	"github.com/firmaelectronica/expedientes-oj/internal/db"
	"github.com/firmaelectronica/expedientes-oj/internal/httpapi"
	"github.com/firmaelectronica/expedientes-oj/internal/migrate"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		if err := migrate.Up(ctx, pool); err != nil {
			log.Fatalf("migraciones: %v", err)
		}
		log.Print("migraciones: ok")
	}

	if err := os.MkdirAll(cfg.StoragePath, 0750); err != nil {
		log.Fatalf("storage: %v", err)
	}

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewRouter(httpapi.RouterDeps{
			Pool:              pool,
			Env:               cfg.Env,
			StoragePath:       cfg.StoragePath,
			CORSOrigins:       cfg.CORSOrigins,
			SignPin:           cfg.SignPin,
			PublicFrontendURL: cfg.PublicFrontendURL,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("servidor escuchando %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
