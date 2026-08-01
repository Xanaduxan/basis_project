package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"

	bcryptadapter "basisProject/internal/adapter/bcrypt"
	jwtadapter "basisProject/internal/adapter/jwt"
	mysqladapter "basisProject/internal/adapter/mysql"
	redisadapter "basisProject/internal/adapter/redis"
	"basisProject/internal/config"
	"basisProject/internal/controller/handler/auth"
	httpcontroller "basisProject/internal/controller/http"
	"basisProject/internal/controller/middleware"
	"basisProject/internal/usecase/auth"
)

func Run(
	ctx context.Context,
	configPath string,
) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf(
			"load application configuration: %w",
			err,
		)
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
			slog.Error("close mysql", "error", err)
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
			slog.Error("close redis", "error", err)
		}
	}()

	userRepository := mysqladapter.NewUserRepository(mysqlDB)
	passwordHasher := bcryptadapter.New()
	tokenManager := jwtadapter.New(
		cfg.JWT.Secret,
		cfg.JWT.Lifetime,
	)

	authUseCase := usecase.NewAuth(
		userRepository,
		passwordHasher,
		tokenManager,
	)

	authHandler := handler.NewAuth(authUseCase)
	authMiddleware := middleware.NewAuth(tokenManager)

	router := httpcontroller.NewRouter()

	httpcontroller.RegisterRoutes(
		router,
		authHandler,
		authMiddleware.Authenticate,
	)

	rootHandler := middleware.Recovery(router)

	server := httpcontroller.NewServer(
		httpcontroller.ServerConfig{
			Host:              cfg.App.Host,
			Port:              cfg.App.Port,
			ReadHeaderTimeout: cfg.App.ReadHeaderTimeout,
			ReadTimeout:       cfg.App.ReadTimeout,
			WriteTimeout:      cfg.App.WriteTimeout,
			IdleTimeout:       cfg.App.IdleTimeout,
		},
		rootHandler,
	)

	slog.Info(
		"application dependencies initialized",
	)

	slog.Info(
		"http server started",
		"address", server.Addr,
	)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, nethttp.ErrServerClosed) {
		return fmt.Errorf("run http server: %w", err)
	}

	return nil
}
