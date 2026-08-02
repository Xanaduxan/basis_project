package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"basisProject/internal/domain"
)

type TeamRepository interface {
	CreateWithOwner(
		ctx context.Context,
		name string,
		createdBy int64,
	) (*domain.Team, error)

	ListByUser(
		ctx context.Context,
		userID int64,
	) ([]domain.TeamWithRole, error)

	FindForMember(
		ctx context.Context,
		teamID int64,
		userID int64,
	) (*domain.TeamWithRole, error)

	AddMember(
		ctx context.Context,
		teamID int64,
		userID int64,
	) error
}

type TeamUserFinder interface {
	FindByEmail(
		ctx context.Context,
		email string,
	) (*domain.User, error)
}

type InvitationNotifier interface {
	SendInvitation(
		ctx context.Context,
		email string,
		teamName string,
	) error
}

type Teams struct {
	teams    TeamRepository
	users    TeamUserFinder
	notifier InvitationNotifier
}

type InviteInput struct {
	TeamID    int64
	InvitedBy int64
	Email     string
}

type InviteResult struct {
	TeamID int64
	UserID int64
	Role   domain.TeamRole
}

func NewTeams(
	teams TeamRepository,
	users TeamUserFinder,
	notifier InvitationNotifier,
) *Teams {
	return &Teams{
		teams:    teams,
		users:    users,
		notifier: notifier,
	}
}

func (u *Teams) Create(
	ctx context.Context,
	userID int64,
	name string,
) (*domain.TeamWithRole, error) {
	name = strings.TrimSpace(name)

	if userID <= 0 ||
		name == "" ||
		utf8.RuneCountInString(name) > 255 {
		return nil, domain.ErrInvalidInput
	}

	team, err := u.teams.CreateWithOwner(
		ctx,
		name,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create team with owner: %w",
			err,
		)
	}

	return &domain.TeamWithRole{
		Team: *team,
		Role: domain.TeamRoleOwner,
	}, nil
}

func (u *Teams) List(
	ctx context.Context,
	userID int64,
) ([]domain.TeamWithRole, error) {
	if userID <= 0 {
		return nil, domain.ErrInvalidInput
	}

	teams, err := u.teams.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"list user teams: %w",
			err,
		)
	}

	return teams, nil
}

func (u *Teams) Invite(
	ctx context.Context,
	input InviteInput,
) (*InviteResult, error) {
	email := strings.ToLower(
		strings.TrimSpace(input.Email),
	)

	if input.TeamID <= 0 ||
		input.InvitedBy <= 0 ||
		email == "" {
		return nil, domain.ErrInvalidInput
	}

	team, err := u.teams.FindForMember(
		ctx,
		input.TeamID,
		input.InvitedBy,
	)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, err
		}

		return nil, fmt.Errorf(
			"find inviter team: %w",
			err,
		)
	}

	if team.Role != domain.TeamRoleOwner &&
		team.Role != domain.TeamRoleAdmin {
		return nil, domain.ErrForbidden
	}

	invitedUser, err := u.users.FindByEmail(
		ctx,
		email,
	)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, err
		}

		return nil, fmt.Errorf(
			"find invited user: %w",
			err,
		)
	}

	if err := u.teams.AddMember(
		ctx,
		input.TeamID,
		invitedUser.ID,
	); err != nil {
		if errors.Is(
			err,
			domain.ErrTeamMemberAlreadyExists,
		) {
			return nil, err
		}

		return nil, fmt.Errorf(
			"add team member: %w",
			err,
		)
	}

	if err := u.notifier.SendInvitation(
		ctx,
		invitedUser.Email,
		team.Team.Name,
	); err != nil {
		slog.Warn(
			"send invitation notification",
			"team_id", input.TeamID,
			"user_id", invitedUser.ID,
			"error", err,
		)
	}

	return &InviteResult{
		TeamID: input.TeamID,
		UserID: invitedUser.ID,
		Role:   domain.TeamRoleMember,
	}, nil
}
