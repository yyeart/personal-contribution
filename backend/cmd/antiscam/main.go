package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yyeart/personal-contribution/backend/internal/logger"
	progress_service "github.com/yyeart/personal-contribution/backend/internal/services/progress"
	users_service "github.com/yyeart/personal-contribution/backend/internal/services/users"
	usersutils "github.com/yyeart/personal-contribution/backend/internal/services/users/utils"
	"github.com/yyeart/personal-contribution/backend/internal/storage/postgres/pool"
	progress_repository "github.com/yyeart/personal-contribution/backend/internal/storage/postgres/progress"
	users_repository "github.com/yyeart/personal-contribution/backend/internal/storage/postgres/user"
	httptransport "github.com/yyeart/personal-contribution/backend/internal/transport/http"
	"go.uber.org/zap"
)

const (
	serverAddress   = ":8080"
	shutdownTimeout = 10 * time.Second
)

func main() {
	loggerConfig := logger.NewConfigMust()
	appLogger, err := logger.NewLogger(loggerConfig)
	if err != nil {
		log.Fatalf("create logger: %v", err)
	}
	defer func() {
		if err := appLogger.Sync(); err != nil {
			log.Printf("sync logger: %v", err)
		}
		appLogger.Close()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbConfig := pool.NewConfigMust()

	db, err := pool.NewPool(ctx, dbConfig)
	if err != nil {
		appLogger.Error("connect to PostgreSQL failed", zap.Error(err))
	}
	defer db.Close()

	usersRepository := users_repository.NewUsersRepository(db)
	usersService := users_service.NewUsersService(
		usersRepository,
		usersutils.BcryptPasswordHasher{},
		usersutils.UUIDGenerator{},
		usersutils.RealClock{},
	)

	progressRepository := progress_repository.NewProgressRepository(db)
	progressService := progress_service.NewProgressService(progressRepository)
	progressHandler := httptransport.NewProgressHandler(progressService)

	server := httptransport.NewServer(usersService, appLogger)

	router := gin.New()
	router.Use(gin.Recovery())

	httptransport.RegisterProgressRoutes(router, progressHandler)

	httpServer := &http.Server{
		Addr:              serverAddress,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("HTTP server stopped with error", zap.Error(err))
			stop()
		}
	}()

	appLogger.Info("AntiScam API started", zap.String("address", serverAddress))

	<-ctx.Done()
	appLogger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("graceful shutdown failed", zap.Error(err))
	}
}
