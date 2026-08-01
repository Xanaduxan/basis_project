package http

import nethttp "net/http"

type AuthHandler interface {
	Register(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)

	Login(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)
}

func RegisterRoutes(
	router *Router,
	authHandler AuthHandler,
	authenticate func(nethttp.Handler) nethttp.Handler,
) {
	router.Handle(
		nethttp.MethodPost,
		"/register",
		nethttp.HandlerFunc(authHandler.Register),
	)

	router.Handle(
		nethttp.MethodPost,
		"/login",
		nethttp.HandlerFunc(authHandler.Login),
	)

	notImplemented := nethttp.HandlerFunc(
		func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			WriteError(
				w,
				nethttp.StatusNotImplemented,
				"not implemented",
			)
		},
	)

	protected := authenticate(notImplemented)

	router.Handle(nethttp.MethodPost, "/teams", protected)
	router.Handle(nethttp.MethodGet, "/teams", protected)
	router.Handle(nethttp.MethodPost, "/teams/{id}/invite", protected)

	router.Handle(nethttp.MethodPost, "/tasks", protected)
	router.Handle(nethttp.MethodGet, "/tasks", protected)
	router.Handle(nethttp.MethodPut, "/tasks/{id}", protected)
	router.Handle(nethttp.MethodGet, "/tasks/{id}/history", protected)
}
