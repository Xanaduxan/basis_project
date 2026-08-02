package middleware

import (
	"context"
	"log/slog"
	nethttp "net/http"

	httpcontroller "basisProject/internal/controller/http"
)

type UserRateLimiter interface {
	Allow(
		ctx context.Context,
		userID int64,
	) (bool, error)
}

type RateLimit struct {
	limiter UserRateLimiter
}

func NewRateLimit(limiter UserRateLimiter) *RateLimit {
	return &RateLimit{
		limiter: limiter,
	}
}

func (m *RateLimit) Limit(
	next nethttp.Handler,
) nethttp.Handler {
	return nethttp.HandlerFunc(
		func(w nethttp.ResponseWriter, r *nethttp.Request) {
			userID, ok := UserIDFromContext(r.Context())
			if !ok || userID <= 0 {
				httpcontroller.WriteError(
					w,
					nethttp.StatusUnauthorized,
					"unauthorized",
				)
				return
			}

			allowed, err := m.limiter.Allow(
				r.Context(),
				userID,
			)
			if err != nil {
				slog.Warn(
					"rate limiter unavailable, allow request",
					"user_id", userID,
					"error", err,
				)

				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				httpcontroller.WriteError(
					w,
					nethttp.StatusTooManyRequests,
					"rate limit exceeded",
				)
				return
			}

			next.ServeHTTP(w, r)
		},
	)
}
