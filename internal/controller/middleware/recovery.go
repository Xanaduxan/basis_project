package middleware

import (
	"log/slog"
	nethttp "net/http"

	httpcontroller "basisProject/internal/controller/http"
)

func Recovery(next nethttp.Handler) nethttp.Handler {
	return nethttp.HandlerFunc(
		func(w nethttp.ResponseWriter, r *nethttp.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				slog.Error(
					"panic recovered",
					"panic", recovered,
					"method", r.Method,
					"path", r.URL.Path,
				)

				httpcontroller.WriteError(
					w,
					nethttp.StatusInternalServerError,
					"internal server error",
				)
			}()

			next.ServeHTTP(w, r)
		},
	)
}
