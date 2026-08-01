package main

import (
	"context"
	"log/slog"
	"os"

	"basisProject/internal/app"
)

const configPath = "configs/config.yaml"

func main() {
	if err := app.Run(context.Background(), configPath); err != nil {
		slog.Error("application startup failed", "error", err)
		os.Exit(1)
	}
}
