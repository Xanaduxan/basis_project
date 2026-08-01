package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"basisProject/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
}

type TokenGenerator interface {
	Generate(userID int64) (string, error)
}

type Auth struct {
	users    UserRepository
	password PasswordHasher
	tokens   TokenGenerator
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

func NewAuth(
	users UserRepository,
	password PasswordHasher,
	tokens TokenGenerator,
) *Auth {
	return &Auth{
		users:    users,
		password: password,
		tokens:   tokens,
	}
}

func (a *Auth) Register(
	ctx context.Context,
	input RegisterInput,
) (*domain.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))

	if name == "" || email == "" || input.Password == "" {
		return nil, domain.ErrInvalidInput
	}

	if len([]byte(input.Password)) > 72 {
		return nil, domain.ErrInvalidInput
	}

	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return nil, domain.ErrInvalidInput
	}

	passwordHash, err := a.password.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash user password: %w", err)
	}

	user := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
	}

	if err := a.users.Create(ctx, user); err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			return nil, err
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (a *Auth) Login(
	ctx context.Context,
	input LoginInput,
) (string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	if email == "" || input.Password == "" {
		return "", domain.ErrInvalidInput
	}

	user, err := a.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}

		return "", fmt.Errorf("find user: %w", err)
	}

	if err := a.password.Compare(
		user.PasswordHash,
		input.Password,
	); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	token, err := a.tokens.Generate(user.ID)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	return token, nil
}
