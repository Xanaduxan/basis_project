package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"basisProject/internal/app"
)

const configPath = "configs/config.yaml"

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.Run(ctx, configPath); err != nil {
		slog.Error(
			"application stopped with error",
			"error", err,
		)
		os.Exit(1)
	}

	slog.Info("application stopped")
}
