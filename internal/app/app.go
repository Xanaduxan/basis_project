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
			slog.Error(
				"close mysql",
				"error", err,
			)
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
		return fmt.Errorf(
			"initialize redis: %w",
			err,
		)
	}

	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Error(
				"close redis",
				"error", err,
			)
		}
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

	passwordHasher := bcryptadapter.New()

	tokenManager := jwtadapter.New(
		cfg.JWT.Secret,
		cfg.JWT.Lifetime,
	)

	emailClient := emailadapter.New(
		cfg.Email.BaseURL,
		cfg.Email.Timeout,
	)

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
		authMiddleware.Authenticate,
		rateLimitMiddleware.Limit,
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

	slog.Info("application dependencies initialized")

	slog.Info(
		"http server started",
		"address", server.Addr,
	)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, nethttp.ErrServerClosed) {
		return fmt.Errorf(
			"run http server: %w",
			err,
		)
	}

	return nil
}
