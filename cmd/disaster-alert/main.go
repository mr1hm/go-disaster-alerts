package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/mr1hm/go-disaster-alerts/internal/api"
	"github.com/mr1hm/go-disaster-alerts/internal/config"
	internalgrpc "github.com/mr1hm/go-disaster-alerts/internal/grpc"
	"github.com/mr1hm/go-disaster-alerts/internal/ingestion"
	"github.com/mr1hm/go-disaster-alerts/internal/logging"
	"github.com/mr1hm/go-disaster-alerts/internal/repository"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		logging.Fatalf("Fatal while loading config: %v", err)
	}
	logging.Setup(cfg.Logging.Level)

	slog.Info("Server starting", "host", cfg.Server.Host, "port", cfg.Server.Port)

	db, err := repository.NewSQLiteDB(cfg.DB.Path)
	if err != nil {
		logging.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create broadcaster for gRPC streaming
	broadcaster := internalgrpc.NewBroadcaster()

	// Start ingestion manager
	mgr := ingestion.NewManager(cfg, db, broadcaster)
	mgr.Start(ctx)

	// Start gRPC server
	grpcServer := internalgrpc.NewServer(db, broadcaster)
	go func() {
		grpcAddr := fmt.Sprintf(":%d", cfg.GRPC.Port)
		if err := grpcServer.Start(grpcAddr); err != nil {
			logging.Fatalf("gRPC server error: %v", err)
		}
	}()

	// Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false, // Set to false when using wildcard origins
	}))
	router.Use(api.RateLimitMiddleware(5)) // 5 req/s global limit

	handler := api.NewHandler(db)
	handler.RegisterRoutes(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")

	cancel()
	mgr.Stop()
	broadcaster.Close() // Close all streams gracefully
	grpcServer.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("shutdown complete")
}
