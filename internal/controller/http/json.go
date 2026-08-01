package http

import (
	"encoding/json"
	"fmt"
	nethttp "net/http"
)

func ReadJSON(r *nethttp.Request, destination any) error {
	if err := json.NewDecoder(r.Body).Decode(destination); err != nil {
		return fmt.Errorf("decode JSON request: %w", err)
	}

	return nil
}

func WriteJSON(
	w nethttp.ResponseWriter,
	status int,
	data any,
) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}

	return nil
}
