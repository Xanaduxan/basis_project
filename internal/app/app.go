package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	nethttp "net/http"

	bcryptadapter "basisProject/internal/adapter/bcrypt"
	emailadapter "basisProject/internal/adapter/email"
	jwtadapter "basisProject/internal/adapter/jwt"
	mysqladapter "basisProject/internal/adapter/mysql"
	prometheusadapter "basisProject/internal/adapter/prometheus"
	redisadapter "basisProject/internal/adapter/redis"
	"basisProject/internal/config"
	authhandler "basisProject/internal/controller/handler/auth"
	taskhandler "basisProject/internal/controller/handler/task"
	teamhandler "basisProject/internal/controller/handler/team"
	httpcontroller "basisProject/internal/controller/http"
	"basisProject/internal/controller/middleware"
	authusecase "basisProject/internal/usecase/auth"
	taskusecase "basisProject/internal/usecase/task"
	teamusecase "basisProject/internal/usecase/team"
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
		return fmt.Errorf(
			"initialize mysql: %w",
			err,
		)
	}

	defer func() {
		if err := mysqlDB.Close(); err != nil {
			slog.Error("close mysql", "error", err)
			return
		}

		slog.Info("mysql connection closed")
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
		return fmt.Errorf(
			"initialize redis: %w",
			err,
		)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error("close redis", "error", err)
			return
		}

		slog.Info("redis connection closed")
	}()

	userRepository := mysqladapter.NewUserRepository(mysqlDB)
	teamRepository := mysqladapter.NewTeamRepository(mysqlDB)
	taskRepository := mysqladapter.NewTaskRepository(mysqlDB)

	taskCache := redisadapter.NewTaskCache(
		redisClient,
		cfg.Redis.TaskTTL,
	)

	rateLimiter := redisadapter.NewRateLimiter(
		redisClient,
		cfg.RateLimit.Requests,
		cfg.RateLimit.Window,
	)

	httpMetrics := prometheusadapter.NewHTTPMetrics()

	passwordHasher := bcryptadapter.New()

	tokenManager := jwtadapter.New(
		cfg.JWT.Secret,
		cfg.JWT.Lifetime,
	)

	emailClient := emailadapter.New(
		cfg.Email.BaseURL,
		cfg.Email.Timeout,
	)

	defer func() {
		emailClient.CloseIdleConnections()
		slog.Info("email idle connections closed")
	}()

	authUseCase := authusecase.NewAuth(
		userRepository,
		passwordHasher,
		tokenManager,
	)

	teamsUseCase := teamusecase.NewTeams(
		teamRepository,
		userRepository,
		emailClient,
	)

	tasksUseCase := taskusecase.NewTasks(
		taskRepository,
		taskCache,
	)

	authHandler := authhandler.NewAuth(authUseCase)
	teamHandler := teamhandler.NewTeam(teamsUseCase)
	taskHandler := taskhandler.NewTask(tasksUseCase)

	authMiddleware := middleware.NewAuth(tokenManager)
	rateLimitMiddleware := middleware.NewRateLimit(rateLimiter)

	router := httpcontroller.NewRouter()

	httpcontroller.RegisterRoutes(
		router,
		authHandler,
		teamHandler,
		taskHandler,
		httpMetrics.Handler(),
		authMiddleware.Authenticate,
		rateLimitMiddleware.Limit,
	)

	rootHandler := middleware.Metrics(
		httpMetrics,
		middleware.Recovery(router),
	)

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

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info(
			"http server started",
			"address", server.Addr,
		)

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			serverErrors <- fmt.Errorf(
				"run http server: %w",
				err,
			)
			return
		}

		serverErrors <- nil
	}()

	select {
	case err := <-serverErrors:
		return err

	case <-ctx.Done():
		slog.Info(
			"shutdown signal received",
			"timeout", cfg.App.ShutdownTimeout,
		)
	}

	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		cfg.App.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		slog.Error(
			"graceful http shutdown failed",
			"error", err,
		)

		if closeErr := server.Close(); closeErr != nil {
			slog.Error(
				"force close http server",
				"error", closeErr,
			)
		}

		return fmt.Errorf(
			"shutdown http server: %w",
			err,
		)
	}

	if err := <-serverErrors; err != nil {
		return err
	}

	slog.Info("http server stopped gracefully")

	return nil
}
