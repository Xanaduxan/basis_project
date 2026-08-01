package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"

	mysqladapter "basisProject/internal/adapter/mysql"
	redisadapter "basisProject/internal/adapter/redis"
	"basisProject/internal/config"
	httpcontroller "basisProject/internal/controller/http"
	"basisProject/internal/controller/middleware"
)

func Run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load application configuration: %w", err)
	}

	mysqlDB, err := mysqladapter.Open(
		ctx,
		mysqladapter.Config{
			DSN:                   cfg.MySQL.DSN,
			MaxOpenConnections:    cfg.MySQL.MaxOpenConnections,
			MaxIdleConnections:    cfg.MySQL.MaxIdleConnections,
			ConnectionMaxLifetime: cfg.MySQL.ConnectionMaxLifetime,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize mysql: %w", err)
	}

	defer func() {
		if err := mysqlDB.Close(); err != nil {
			slog.Error("close mysql connection pool", "error", err)
		}
	}()

	redisClient, err := redisadapter.Open(
		ctx,
		redisadapter.Config{
			Address:  cfg.Redis.Address,
			Password: cfg.Redis.Password,
			Database: cfg.Redis.Database,
		},
	)
	if err != nil {
		return fmt.Errorf("initialize redis: %w", err)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("close redis client", "error", err)
		}
	}()

	router := httpcontroller.NewRouter()
	httpcontroller.RegisterRoutes(router)

	handler := middleware.Recovery(router)

	server := httpcontroller.NewServer(
		httpcontroller.ServerConfig{
			Host:              cfg.App.Host,
			Port:              cfg.App.Port,
			ReadHeaderTimeout: cfg.App.ReadHeaderTimeout,
			ReadTimeout:       cfg.App.ReadTimeout,
			WriteTimeout:      cfg.App.WriteTimeout,
			IdleTimeout:       cfg.App.IdleTimeout,
		},
		handler,
	)

	slog.Info(
		"HTTP server started",
		"address", server.Addr,
	)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, nethttp.ErrServerClosed) {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	return nil
}
