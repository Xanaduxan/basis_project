package jwt

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret   []byte
	lifetime time.Duration
}

func New(secret string, lifetime time.Duration) *Manager {
	return &Manager{
		secret:   []byte(secret),
		lifetime: lifetime,
	}
}

func (m *Manager) Generate(userID int64) (string, error) {
	now := time.Now()

	claims := jwtlib.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwtlib.NewNumericDate(now),
		ExpiresAt: jwtlib.NewNumericDate(now.Add(m.lifetime)),
	}

	token := jwtlib.NewWithClaims(
		jwtlib.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signedToken, nil
}

func (m *Manager) Parse(tokenString string) (int64, error) {
	claims := new(jwtlib.RegisteredClaims)

	token, err := jwtlib.ParseWithClaims(
		tokenString,
		claims,
		func(_ *jwtlib.Token) (any, error) {
			return m.secret, nil
		},
		jwtlib.WithValidMethods(
			[]string{jwtlib.SigningMethodHS256.Alg()},
		),
		jwtlib.WithExpirationRequired(),
	)
	if err != nil {
		return 0, fmt.Errorf("parse JWT: %w", err)
	}

	if !token.Valid {
		return 0, errors.New("invalid JWT")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return 0, errors.New("invalid JWT subject")
	}

	return userID, nil
}
