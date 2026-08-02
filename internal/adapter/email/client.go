package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/sony/gobreaker/v2"
)

const (
	breakerMaxRequests      uint32 = 1
	breakerFailureThreshold uint32 = 3
	breakerOpenTimeout             = 30 * time.Second
)

type Client struct {
	baseURL    string
	httpClient *nethttp.Client
	breaker    *gobreaker.CircuitBreaker[struct{}]
}

type invitationRequest struct {
	To       string `json:"to"`
	TeamName string `json:"team_name"`
}

func New(
	baseURL string,
	timeout time.Duration,
) *Client {
	breaker := gobreaker.NewCircuitBreaker[struct{}](
		gobreaker.Settings{
			Name:        "email-service",
			MaxRequests: breakerMaxRequests,
			Timeout:     breakerOpenTimeout,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >=
					breakerFailureThreshold
			},
			OnStateChange: func(
				name string,
				from gobreaker.State,
				to gobreaker.State,
			) {
				slog.Warn(
					"circuit breaker state changed",
					"name", name,
					"from", from.String(),
					"to", to.String(),
				)
			},
		},
	)

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &nethttp.Client{
			Timeout: timeout,
		},
		breaker: breaker,
	}
}

func (c *Client) SendInvitation(
	ctx context.Context,
	email string,
	teamName string,
) error {
	payload, err := json.Marshal(
		invitationRequest{
			To:       email,
			TeamName: teamName,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"marshal email request: %w",
			err,
		)
	}

	_, err = c.breaker.Execute(
		func() (struct{}, error) {
			request, err := nethttp.NewRequestWithContext(
				ctx,
				nethttp.MethodPost,
				c.baseURL+"/send",
				bytes.NewReader(payload),
			)
			if err != nil {
				return struct{}{}, fmt.Errorf(
					"create email request: %w",
					err,
				)
			}

			request.Header.Set(
				"Content-Type",
				"application/json",
			)

			response, err := c.httpClient.Do(request)
			if err != nil {
				return struct{}{}, fmt.Errorf(
					"send email request: %w",
					err,
				)
			}
			defer response.Body.Close()

			_, _ = io.Copy(io.Discard, response.Body)

			if response.StatusCode < 200 ||
				response.StatusCode >= 300 {
				return struct{}{}, fmt.Errorf(
					"email service returned status %d",
					response.StatusCode,
				)
			}

			return struct{}{}, nil
		},
	)
	if err != nil {
		return fmt.Errorf(
			"execute email request: %w",
			err,
		)
	}

	return nil
}

func (c *Client) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}
