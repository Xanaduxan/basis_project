package http

import (
	nethttp "net/http"
)

const apiPrefix = "/api/v1"

type Router struct {
	mux *nethttp.ServeMux
}

func NewRouter() *Router {
	return &Router{
		mux: nethttp.NewServeMux(),
	}
}

func (r *Router) Handle(
	method string,
	path string,
	handler nethttp.Handler,
) {
	r.HandleRoot(method, apiPrefix+path, handler)
}

func (r *Router) HandleRoot(
	method string,
	path string,
	handler nethttp.Handler,
) {
	pattern := method + " " + path
	r.mux.Handle(pattern, handler)
}

func (r *Router) ServeHTTP(
	w nethttp.ResponseWriter,
	req *nethttp.Request,
) {
	r.mux.ServeHTTP(w, req)
}
