package usecase

import (
	"context"
	"errors"
	"testing"

	"basisProject/internal/domain"
)

type teamRepositoryMock struct {
	team      *domain.TeamWithRole
	created   *domain.Team
	addErr    error
	createErr error
	addedUser int64
}

func (m *teamRepositoryMock) CreateWithOwner(_ context.Context, name string, createdBy int64) (*domain.Team, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.created = &domain.Team{ID: 10, Name: name, CreatedBy: createdBy}
	return m.created, nil
}
func (m *teamRepositoryMock) ListByUser(context.Context, int64) ([]domain.TeamWithRole, error) {
	return nil, nil
}
func (m *teamRepositoryMock) FindForMember(context.Context, int64, int64) (*domain.TeamWithRole, error) {
	if m.team == nil {
		return nil, domain.ErrForbidden
	}
	return m.team, nil
}
func (m *teamRepositoryMock) AddMember(_ context.Context, _ int64, userID int64) error {
	m.addedUser = userID
	return m.addErr
}

type teamUserFinderMock struct {
	user *domain.User
	err  error
}

func (m *teamUserFinderMock) FindByEmail(context.Context, string) (*domain.User, error) {
	return m.user, m.err
}

type notifierMock struct {
	called bool
	err    error
}

func (m *notifierMock) SendInvitation(context.Context, string, string) error {
	m.called = true
	return m.err
}

func TestTeamsCreate(t *testing.T) {
	repository := &teamRepositoryMock{}
	service := NewTeams(repository, &teamUserFinderMock{}, &notifierMock{})

	team, err := service.Create(context.Background(), 5, "  Backend  ")
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	if team.Team.Name != "Backend" || team.Team.CreatedBy != 5 || team.Role != domain.TeamRoleOwner {
		t.Fatalf("unexpected team: %#v", team)
	}
}

func TestTeamsInvite(t *testing.T) {
	for _, role := range []domain.TeamRole{domain.TeamRoleOwner, domain.TeamRoleAdmin} {
		t.Run(string(role)+" may invite", func(t *testing.T) {
			repository := &teamRepositoryMock{team: &domain.TeamWithRole{Team: domain.Team{ID: 3, Name: "Backend"}, Role: role}}
			notifier := &notifierMock{}
			service := NewTeams(repository, &teamUserFinderMock{user: &domain.User{ID: 8, Email: "bob@example.com"}}, notifier)

			result, err := service.Invite(context.Background(), InviteInput{TeamID: 3, InvitedBy: 1, Email: " BOB@example.com "})
			if err != nil {
				t.Fatalf("invite: %v", err)
			}
			if result.UserID != 8 || result.Role != domain.TeamRoleMember || repository.addedUser != 8 || !notifier.called {
				t.Fatalf("unexpected invitation result: %#v", result)
			}
		})
	}

	t.Run("member cannot invite", func(t *testing.T) {
		service := NewTeams(
			&teamRepositoryMock{team: &domain.TeamWithRole{Role: domain.TeamRoleMember}},
			&teamUserFinderMock{}, &notifierMock{},
		)
		_, err := service.Invite(context.Background(), InviteInput{TeamID: 3, InvitedBy: 1, Email: "bob@example.com"})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected forbidden, got %v", err)
		}
	})

	t.Run("rejects duplicate membership", func(t *testing.T) {
		service := NewTeams(
			&teamRepositoryMock{team: &domain.TeamWithRole{Role: domain.TeamRoleOwner}, addErr: domain.ErrTeamMemberAlreadyExists},
			&teamUserFinderMock{user: &domain.User{ID: 8}}, &notifierMock{},
		)
		_, err := service.Invite(context.Background(), InviteInput{TeamID: 3, InvitedBy: 1, Email: "bob@example.com"})
		if !errors.Is(err, domain.ErrTeamMemberAlreadyExists) {
			t.Fatalf("expected duplicate membership, got %v", err)
		}
	})

	t.Run("notification failure does not undo invitation", func(t *testing.T) {
		service := NewTeams(
			&teamRepositoryMock{team: &domain.TeamWithRole{Team: domain.Team{Name: "Backend"}, Role: domain.TeamRoleOwner}},
			&teamUserFinderMock{user: &domain.User{ID: 8, Email: "bob@example.com"}},
			&notifierMock{err: errors.New("email unavailable")},
		)
		if _, err := service.Invite(context.Background(), InviteInput{TeamID: 3, InvitedBy: 1, Email: "bob@example.com"}); err != nil {
			t.Fatalf("notification error must be non-fatal: %v", err)
		}
	})
}
