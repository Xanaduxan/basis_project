package http

import (
	"log/slog"
	nethttp "net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteError(
	w nethttp.ResponseWriter,
	status int,
	message string,
) {
	response := ErrorResponse{
		Error: message,
	}

	if err := WriteJSON(w, status, response); err != nil {
		slog.Error("write HTTP error response", "error", err)
	}
}
