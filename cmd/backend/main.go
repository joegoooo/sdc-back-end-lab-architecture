package main

import (
	"context"
	"net/http"

	"sdclab/databaseutil"
	"sdclab/internal/form"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const databaseURL = "postgresql://postgres:password@localhost:5432/postgres?sslmode=disable"

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("Starting backend service")

	if err := databaseutil.MigrationUp("file://internal/database/migrations", databaseURL, logger); err != nil {
		logger.Fatal("Failed to run database migration", zap.Error(err))
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		logger.Fatal("Failed to parse database URL", zap.Error(err))
	}

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		logger.Fatal("Failed to create database connection pool", zap.Error(err))
	}
	defer dbPool.Close()

	formService := form.NewService(logger, form.New(dbPool))
	formHandler := form.NewHandler(logger, formService)

	mux := http.NewServeMux()
	formHandler.RegisterRoutes(mux)

	server := &http.Server{Addr: ":8080", Handler: mux}
	logger.Info("Backend started on :8080")

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}
