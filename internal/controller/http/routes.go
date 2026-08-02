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

type TeamHandler interface {
	Create(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)

	List(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)

	Invite(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)
}

type TaskHandler interface {
	Create(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)

	List(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)
}

func RegisterRoutes(
	router *Router,
	authHandler AuthHandler,
	teamHandler TeamHandler,
	taskHandler TaskHandler,
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

	router.Handle(
		nethttp.MethodPost,
		"/teams",
		authenticate(
			nethttp.HandlerFunc(teamHandler.Create),
		),
	)

	router.Handle(
		nethttp.MethodGet,
		"/teams",
		authenticate(
			nethttp.HandlerFunc(teamHandler.List),
		),
	)

	router.Handle(
		nethttp.MethodPost,
		"/teams/{id}/invite",
		authenticate(
			nethttp.HandlerFunc(teamHandler.Invite),
		),
	)

	router.Handle(
		nethttp.MethodPost,
		"/tasks",
		authenticate(
			nethttp.HandlerFunc(taskHandler.Create),
		),
	)

	router.Handle(
		nethttp.MethodGet,
		"/tasks",
		authenticate(
			nethttp.HandlerFunc(taskHandler.List),
		),
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

	protectedNotImplemented := authenticate(notImplemented)

	router.Handle(
		nethttp.MethodPut,
		"/tasks/{id}",
		protectedNotImplemented,
	)

	router.Handle(
		nethttp.MethodGet,
		"/tasks/{id}/history",
		protectedNotImplemented,
	)
}
