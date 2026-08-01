package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	mysqldriver "github.com/go-sql-driver/mysql"

	"basisProject/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(
	ctx context.Context,
	user *domain.User,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`
      INSERT INTO users (name, email, password_hash)
      VALUES (?, ?, ?)
    `,
		user.Name,
		user.Email,
		user.PasswordHash,
	)
	if err != nil {
		var mysqlError *mysqldriver.MySQLError

		if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
			return domain.ErrEmailAlreadyExists
		}

		return fmt.Errorf("insert user: %w", err)
	}

	user.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get inserted user id: %w", err)
	}

	return nil
}

func (r *UserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*domain.User, error) {
	var user domain.User

	err := r.db.QueryRowContext(
		ctx,
		`
      SELECT id, name, email, password_hash, created_at, updated_at
      FROM users
      WHERE email = ?
    `,
		email,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &user, nil
}
