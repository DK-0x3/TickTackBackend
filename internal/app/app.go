package app

import (
	"context"
	"fmt"
	"time"

	"github.com/DK-0x3/TickTackBackend/internal/config"
	"github.com/DK-0x3/TickTackBackend/internal/modules/tasks"
	"github.com/DK-0x3/TickTackBackend/internal/modules/taskviews"
	"github.com/DK-0x3/TickTackBackend/internal/platform/httpserver"
	loggerplatform "github.com/DK-0x3/TickTackBackend/internal/platform/logger"
	"github.com/DK-0x3/TickTackBackend/internal/platform/postgres"
	dbsqlc "github.com/DK-0x3/TickTackBackend/internal/platform/postgres/sqlc"
)

func Run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := loggerplatform.New(string(cfg.Environment))

	database, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer database.Close()

	logger.Info("postgres connected")

	queries := dbsqlc.New(database)
	taskRepository := tasks.NewRepository(queries)
	taskService := tasks.NewService(taskRepository)
	viewRepository := taskviews.NewRepository(queries)
	viewService := taskviews.NewService(viewRepository, cfg.DevelopmentLocation, time.Now)

	router := newRouter(cfg.Environment, database, taskService, viewService, cfg.DevelopmentUserID)
	server := httpserver.New(cfg.HTTPAddress, router, logger)

	return server.Run(ctx)
}
