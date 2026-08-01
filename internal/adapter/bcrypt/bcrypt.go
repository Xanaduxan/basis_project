package bcrypt

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Hasher struct{}

func New() *Hasher {
	return &Hasher{}
}

func (*Hasher) Hash(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", fmt.Errorf("generate password hash: %w", err)
	}

	return string(passwordHash), nil
}

func (*Hasher) Compare(passwordHash, password string) error {
	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(password),
	); err != nil {
		return fmt.Errorf("compare password hash: %w", err)
	}

	return nil
}
