package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/DK-0x3/TickTackBackend/internal/app"
	"github.com/joho/godotenv"
)

func main() {
	if err := loadDotEnv(); err != nil {
		slog.Error("load .env", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.Run(ctx); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func loadDotEnv() error {
	err := godotenv.Load()
	if err == nil {
		return nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}
