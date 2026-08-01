package handler

import (
	"errors"
	"log/slog"
	nethttp "net/http"

	"basisProject/internal/controller/dto/auth"
	httpcontroller "basisProject/internal/controller/http"
	"basisProject/internal/domain"
	"basisProject/internal/usecase/auth"
)

type Auth struct {
	auth *usecase.Auth
}

func NewAuth(auth *usecase.Auth) *Auth {
	return &Auth{
		auth: auth,
	}
}

func (h *Auth) Register(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	var request dto.RegisterRequest

	if err := httpcontroller.ReadJSON(r, &request); err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid JSON",
		)
		return
	}

	user, err := h.auth.Register(
		r.Context(),
		usecase.RegisterInput{
			Name:     request.Name,
			Email:    request.Email,
			Password: request.Password,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			httpcontroller.WriteError(
				w,
				nethttp.StatusBadRequest,
				"invalid registration data",
			)

		case errors.Is(err, domain.ErrEmailAlreadyExists):
			httpcontroller.WriteError(
				w,
				nethttp.StatusConflict,
				"email already exists",
			)

		default:
			slog.Error("register user", "error", err)

			httpcontroller.WriteError(
				w,
				nethttp.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	response := dto.RegisterResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusCreated,
		response,
	); err != nil {
		slog.Error("write registration response", "error", err)
	}
}

func (h *Auth) Login(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	var request dto.LoginRequest

	if err := httpcontroller.ReadJSON(r, &request); err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid JSON",
		)
		return
	}

	token, err := h.auth.Login(
		r.Context(),
		usecase.LoginInput{
			Email:    request.Email,
			Password: request.Password,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			httpcontroller.WriteError(
				w,
				nethttp.StatusBadRequest,
				"email and password are required",
			)

		case errors.Is(err, domain.ErrInvalidCredentials):
			httpcontroller.WriteError(
				w,
				nethttp.StatusUnauthorized,
				"invalid email or password",
			)

		default:
			slog.Error("login user", "error", err)

			httpcontroller.WriteError(
				w,
				nethttp.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	response := dto.LoginResponse{
		AccessToken: token,
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusOK,
		response,
	); err != nil {
		slog.Error("write login response", "error", err)
	}
}
