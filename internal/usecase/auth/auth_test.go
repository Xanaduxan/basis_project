package usecase

import (
	"context"
	"errors"
	"testing"

	"basisProject/internal/domain"
)

type userRepositoryMock struct {
	created   *domain.User
	user      *domain.User
	createErr error
	findErr   error
}

func (m *userRepositoryMock) Create(_ context.Context, user *domain.User) error {
	m.created = user
	if m.createErr == nil {
		user.ID = 42
	}
	return m.createErr
}

func (m *userRepositoryMock) FindByEmail(_ context.Context, _ string) (*domain.User, error) {
	return m.user, m.findErr
}

type passwordHasherMock struct {
	hash       string
	hashErr    error
	compareErr error
}

func (m *passwordHasherMock) Hash(string) (string, error)  { return m.hash, m.hashErr }
func (m *passwordHasherMock) Compare(string, string) error { return m.compareErr }

type tokenGeneratorMock struct {
	token string
	err   error
}

func (m *tokenGeneratorMock) Generate(int64) (string, error) { return m.token, m.err }

func TestAuthRegister(t *testing.T) {
	t.Run("registers normalized user", func(t *testing.T) {
		repository := &userRepositoryMock{}
		service := NewAuth(repository, &passwordHasherMock{hash: "hash"}, &tokenGeneratorMock{})

		user, err := service.Register(context.Background(), RegisterInput{
			Name: "  Alice  ", Email: "  ALICE@example.com ", Password: "secret123",
		})
		if err != nil {
			t.Fatalf("register: %v", err)
		}
		if user.ID != 42 || user.Name != "Alice" || user.Email != "alice@example.com" || user.PasswordHash != "hash" {
			t.Fatalf("unexpected user: %#v", user)
		}
	})

	t.Run("rejects invalid input", func(t *testing.T) {
		service := NewAuth(&userRepositoryMock{}, &passwordHasherMock{}, &tokenGeneratorMock{})
		_, err := service.Register(context.Background(), RegisterInput{Name: "Alice", Email: "bad email", Password: "secret123"})
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected invalid input, got %v", err)
		}
	})

	t.Run("preserves duplicate email error", func(t *testing.T) {
		service := NewAuth(
			&userRepositoryMock{createErr: domain.ErrEmailAlreadyExists},
			&passwordHasherMock{hash: "hash"},
			&tokenGeneratorMock{},
		)
		_, err := service.Register(context.Background(), RegisterInput{Name: "Alice", Email: "alice@example.com", Password: "secret123"})
		if !errors.Is(err, domain.ErrEmailAlreadyExists) {
			t.Fatalf("expected duplicate email, got %v", err)
		}
	})
}

func TestAuthLogin(t *testing.T) {
	t.Run("returns token", func(t *testing.T) {
		service := NewAuth(
			&userRepositoryMock{user: &domain.User{ID: 7, PasswordHash: "hash"}},
			&passwordHasherMock{},
			&tokenGeneratorMock{token: "jwt"},
		)
		token, err := service.Login(context.Background(), LoginInput{Email: "alice@example.com", Password: "secret123"})
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		if token != "jwt" {
			t.Fatalf("unexpected token %q", token)
		}
	})

	t.Run("hides missing user", func(t *testing.T) {
		service := NewAuth(
			&userRepositoryMock{findErr: domain.ErrUserNotFound},
			&passwordHasherMock{},
			&tokenGeneratorMock{},
		)
		_, err := service.Login(context.Background(), LoginInput{Email: "missing@example.com", Password: "secret123"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials, got %v", err)
		}
	})

	t.Run("rejects wrong password", func(t *testing.T) {
		service := NewAuth(
			&userRepositoryMock{user: &domain.User{ID: 7, PasswordHash: "hash"}},
			&passwordHasherMock{compareErr: errors.New("mismatch")},
			&tokenGeneratorMock{},
		)
		_, err := service.Login(context.Background(), LoginInput{Email: "alice@example.com", Password: "wrong"})
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("expected invalid credentials, got %v", err)
		}
	})
}
