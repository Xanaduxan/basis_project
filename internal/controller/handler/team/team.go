package handler

import (
	"errors"
	"log/slog"
	nethttp "net/http"
	"strconv"

	"basisProject/internal/controller/dto/team"
	httpcontroller "basisProject/internal/controller/http"
	"basisProject/internal/controller/middleware"
	"basisProject/internal/domain"
	"basisProject/internal/usecase/team"
)

type Team struct {
	teams *usecase.Teams
}

func NewTeam(teams *usecase.Teams) *Team {
	return &Team{
		teams: teams,
	}
}

func (h *Team) Create(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	var request dto.CreateTeamRequest

	if err := httpcontroller.ReadJSON(r, &request); err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid JSON",
		)
		return
	}

	team, err := h.teams.Create(
		r.Context(),
		userID,
		request.Name,
	)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			httpcontroller.WriteError(
				w,
				nethttp.StatusBadRequest,
				"invalid team data",
			)
			return
		}

		slog.Error("create team", "error", err)

		httpcontroller.WriteError(
			w,
			nethttp.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	response := teamResponse(*team)

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusCreated,
		response,
	); err != nil {
		slog.Error("write create team response", "error", err)
	}
}

func (h *Team) List(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	teams, err := h.teams.List(r.Context(), userID)
	if err != nil {
		slog.Error("list user teams", "error", err)

		httpcontroller.WriteError(
			w,
			nethttp.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	response := make([]dto.TeamResponse, 0, len(teams))

	for _, team := range teams {
		response = append(response, teamResponse(team))
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusOK,
		response,
	); err != nil {
		slog.Error("write teams response", "error", err)
	}
}

func (h *Team) Invite(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	teamID, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || teamID <= 0 {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid team id",
		)
		return
	}

	var request dto.InviteTeamMemberRequest

	if err := httpcontroller.ReadJSON(r, &request); err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid JSON",
		)
		return
	}

	result, err := h.teams.Invite(
		r.Context(),
		usecase.InviteInput{
			TeamID:    teamID,
			InvitedBy: userID,
			Email:     request.Email,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			httpcontroller.WriteError(
				w,
				nethttp.StatusBadRequest,
				"email is required",
			)

		case errors.Is(err, domain.ErrForbidden):
			httpcontroller.WriteError(
				w,
				nethttp.StatusForbidden,
				"insufficient permissions",
			)

		case errors.Is(err, domain.ErrUserNotFound):
			httpcontroller.WriteError(
				w,
				nethttp.StatusNotFound,
				"user not found",
			)

		case errors.Is(
			err,
			domain.ErrTeamMemberAlreadyExists,
		):
			httpcontroller.WriteError(
				w,
				nethttp.StatusConflict,
				"user is already a team member",
			)

		default:
			slog.Error("invite team member", "error", err)

			httpcontroller.WriteError(
				w,
				nethttp.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	response := dto.InviteTeamMemberResponse{
		TeamID: result.TeamID,
		UserID: result.UserID,
		Role:   string(result.Role),
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusCreated,
		response,
	); err != nil {
		slog.Error("write invite response", "error", err)
	}
}

func teamResponse(team domain.TeamWithRole) dto.TeamResponse {
	return dto.TeamResponse{
		ID:        team.Team.ID,
		Name:      team.Team.Name,
		CreatedBy: team.Team.CreatedBy,
		Role:      string(team.Role),
	}
}

func authenticatedUserID(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) (int64, bool) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		httpcontroller.WriteError(
			w,
			nethttp.StatusUnauthorized,
			"unauthorized",
		)
		return 0, false
	}

	return userID, true
}
