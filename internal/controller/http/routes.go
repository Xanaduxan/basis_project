package http

import (
	nethttp "net/http"
)

func RegisterRoutes(router *Router) {
	notImplemented := nethttp.HandlerFunc(
		func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			WriteError(
				w,
				nethttp.StatusNotImplemented,
				"not implemented",
			)
		},
	)

	router.Handle(
		nethttp.MethodPost,
		"/register",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodPost,
		"/login",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodPost,
		"/teams",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodGet,
		"/teams",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodPost,
		"/teams/{id}/invite",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodPost,
		"/tasks",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodGet,
		"/tasks",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodPut,
		"/tasks/{id}",
		notImplemented,
	)

	router.Handle(
		nethttp.MethodGet,
		"/tasks/{id}/history",
		notImplemented,
	)
}
