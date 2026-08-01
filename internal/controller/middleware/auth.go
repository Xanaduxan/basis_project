package middleware

import (
	"context"
	nethttp "net/http"
	"strings"

	httpcontroller "basisProject/internal/controller/http"
)

type TokenParser interface {
	Parse(token string) (int64, error)
}

type Auth struct {
	tokens TokenParser
}

type contextKey string

const userIDKey contextKey = "user_id"

func NewAuth(tokens TokenParser) *Auth {
	return &Auth{
		tokens: tokens,
	}
}

func (m *Auth) Authenticate(
	next nethttp.Handler,
) nethttp.Handler {
	return nethttp.HandlerFunc(
		func(w nethttp.ResponseWriter, r *nethttp.Request) {
			authorization := r.Header.Get("Authorization")
			parts := strings.Fields(authorization)

			if len(parts) != 2 ||
				!strings.EqualFold(parts[0], "Bearer") {
				httpcontroller.WriteError(
					w,
					nethttp.StatusUnauthorized,
					"unauthorized",
				)
				return
			}

			userID, err := m.tokens.Parse(parts[1])
			if err != nil {
				httpcontroller.WriteError(
					w,
					nethttp.StatusUnauthorized,
					"unauthorized",
				)
				return
			}

			ctx := context.WithValue(
				r.Context(),
				userIDKey,
				userID,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

func UserIDFromContext(
	ctx context.Context,
) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}
