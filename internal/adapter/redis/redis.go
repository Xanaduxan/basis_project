package redis

import (
	"context"
	"fmt"

	redisclient "github.com/redis/go-redis/v9"
)

type Config struct {
	Address  string
	Password string
	Database int
}

func Open(ctx context.Context, cfg Config) (*redisclient.Client, error) {
	client := redisclient.NewClient(&redisclient.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
