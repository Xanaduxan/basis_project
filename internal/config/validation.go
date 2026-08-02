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

	if c.RateLimit.Requests <= 0 {
		return errors.New(
			"rate_limit.requests must be greater than zero",
		)
	}

	if c.RateLimit.Window <= 0 {
		return errors.New(
			"rate_limit.window must be greater than zero",
		)
	}

	return nil
}
