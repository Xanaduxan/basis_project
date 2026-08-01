package config

import (
	"errors"
	"strings"
)

func (c Config) Validate() error {
	if strings.TrimSpace(c.MySQL.DSN) == "" {
		return errors.New("mysql.dsn is required")
	}

	if strings.TrimSpace(c.JWT.Secret) == "" {
		return errors.New("jwt.secret is required")
	}

	return nil
}
