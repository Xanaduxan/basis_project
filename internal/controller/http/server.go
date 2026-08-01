package http

import (
	"net"
	nethttp "net/http"
	"strconv"
	"time"
)

type ServerConfig struct {
	Host              string
	Port              int
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func NewServer(
	cfg ServerConfig,
	handler nethttp.Handler,
) *nethttp.Server {
	address := net.JoinHostPort(
		cfg.Host,
		strconv.Itoa(cfg.Port),
	)

	return &nethttp.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
}
