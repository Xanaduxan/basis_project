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

	Update(
		w nethttp.ResponseWriter,
		r *nethttp.Request,
	)

	History(
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
	rateLimit func(nethttp.Handler) nethttp.Handler,
) {
	protected := func(handler nethttp.Handler) nethttp.Handler {
		return authenticate(rateLimit(handler))
	}

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
		protected(
			nethttp.HandlerFunc(teamHandler.Create),
		),
	)

	router.Handle(
		nethttp.MethodGet,
		"/teams",
		protected(
			nethttp.HandlerFunc(teamHandler.List),
		),
	)

	router.Handle(
		nethttp.MethodPost,
		"/teams/{id}/invite",
		protected(
			nethttp.HandlerFunc(teamHandler.Invite),
		),
	)

	router.Handle(
		nethttp.MethodPost,
		"/tasks",
		protected(
			nethttp.HandlerFunc(taskHandler.Create),
		),
	)

	router.Handle(
		nethttp.MethodGet,
		"/tasks",
		protected(
			nethttp.HandlerFunc(taskHandler.List),
		),
	)

	router.Handle(
		nethttp.MethodPut,
		"/tasks/{id}",
		protected(
			nethttp.HandlerFunc(taskHandler.Update),
		),
	)

	router.Handle(
		nethttp.MethodGet,
		"/tasks/{id}/history",
		protected(
			nethttp.HandlerFunc(taskHandler.History),
		),
	)
}
