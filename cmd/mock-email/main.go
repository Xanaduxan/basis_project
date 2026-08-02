package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"
	"os"
	"strings"
	"time"
)

type sendRequest struct {
	To       string `json:"to"`
	TeamName string `json:"team_name"`
}

func main() {
	mux := nethttp.NewServeMux()

	mux.HandleFunc(
		nethttp.MethodPost+" /send",
		handleSend,
	)

	server := &nethttp.Server{
		Addr:              ":8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info(
		"mock email service started",
		"address", server.Addr,
	)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, nethttp.ErrServerClosed) {
		slog.Error(
			"mock email service stopped",
			"error", err,
		)
		os.Exit(1)
	}
}

func handleSend(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	var request sendRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		nethttp.Error(
			w,
			"invalid JSON",
			nethttp.StatusBadRequest,
		)
		return
	}

	request.To = strings.TrimSpace(request.To)
	request.TeamName = strings.TrimSpace(request.TeamName)

	if request.To == "" || request.TeamName == "" {
		nethttp.Error(
			w,
			"to and team_name are required",
			nethttp.StatusBadRequest,
		)
		return
	}

	slog.Info(
		"invitation email sent",
		"to", request.To,
		"team", request.TeamName,
	)

	w.WriteHeader(nethttp.StatusNoContent)
}
