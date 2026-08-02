package redis

import (
	"context"
	"fmt"
	"time"

	redisclient "github.com/redis/go-redis/v9"
)

var incrementRateLimitScript = redisclient.NewScript(`
local count = redis.call("INCR", KEYS[1])

if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end

return count
`)

type RateLimiter struct {
	client *redisclient.Client
	limit  int64
	window time.Duration
}

func NewRateLimiter(
	client *redisclient.Client,
	limit int,
	window time.Duration,
) *RateLimiter {
	return &RateLimiter{
		client: client,
		limit:  int64(limit),
		window: window,
	}
}

func (l *RateLimiter) Allow(
	ctx context.Context,
	userID int64,
) (bool, error) {
	key := fmt.Sprintf("rate_limit:user:%d", userID)

	count, err := incrementRateLimitScript.Run(
		ctx,
		l.client,
		[]string{key},
		l.window.Milliseconds(),
	).Int64()
	if err != nil {
		return false, fmt.Errorf(
			"increment rate limit counter: %w",
			err,
		)
	}

	return count <= l.limit, nil
}
