package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"stockflow/backend/internal/config"
	"stockflow/backend/internal/db"
	"stockflow/backend/internal/order"
	"stockflow/backend/internal/product"
	"stockflow/backend/internal/server"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	products := product.NewHandler(product.NewStore(pool))
	orders := order.NewHandler(order.NewStore(pool))
	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           server.New(products, orders),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("stockflow api listening on %s", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-shutdownCtx.Done()
	stop()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(timeoutCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("stockflow api stopped")
}
