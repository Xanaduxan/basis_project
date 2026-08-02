package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	mysqldriver "github.com/go-sql-driver/mysql"

	"basisProject/internal/domain"
)

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{
		db: db,
	}
}

func (r *TeamRepository) CreateWithOwner(
	ctx context.Context,
	name string,
	createdBy int64,
) (*domain.Team, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf(
			"begin create team transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(
		ctx,
		`
      INSERT INTO teams (name, created_by)
      VALUES (?, ?)
    `,
		name,
		createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("insert team: %w", err)
	}

	teamID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf(
			"get inserted team id: %w",
			err,
		)
	}

	_, err = tx.ExecContext(
		ctx,
		`
      INSERT INTO team_members (team_id, user_id, role)
      VALUES (?, ?, ?)
    `,
		teamID,
		createdBy,
		domain.TeamRoleOwner,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"insert team owner: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"commit create team transaction: %w",
			err,
		)
	}

	return &domain.Team{
		ID:        teamID,
		Name:      name,
		CreatedBy: createdBy,
	}, nil
}

func (r *TeamRepository) ListByUser(
	ctx context.Context,
	userID int64,
) ([]domain.TeamWithRole, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
      SELECT
        t.id,
        t.name,
        t.created_by,
        tm.role
      FROM teams AS t
      INNER JOIN team_members AS tm
        ON tm.team_id = t.id
      WHERE tm.user_id = ?
      ORDER BY t.id DESC
    `,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query user teams: %w",
			err,
		)
	}
	defer rows.Close()

	teams := make([]domain.TeamWithRole, 0)

	for rows.Next() {
		var team domain.TeamWithRole
		var role string

		if err := rows.Scan(
			&team.Team.ID,
			&team.Team.Name,
			&team.Team.CreatedBy,
			&role,
		); err != nil {
			return nil, fmt.Errorf(
				"scan user team: %w",
				err,
			)
		}

		team.Role = domain.TeamRole(role)
		teams = append(teams, team)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate user teams: %w",
			err,
		)
	}

	return teams, nil
}

func (r *TeamRepository) FindForMember(
	ctx context.Context,
	teamID int64,
	userID int64,
) (*domain.TeamWithRole, error) {
	var team domain.TeamWithRole
	var role string

	err := r.db.QueryRowContext(
		ctx,
		`
      SELECT
        t.id,
        t.name,
        t.created_by,
        tm.role
      FROM teams AS t
      INNER JOIN team_members AS tm
        ON tm.team_id = t.id
      WHERE t.id = ? AND tm.user_id = ?
    `,
		teamID,
		userID,
	).Scan(
		&team.Team.ID,
		&team.Team.Name,
		&team.Team.CreatedBy,
		&role,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrForbidden
		}

		return nil, fmt.Errorf(
			"find team for member: %w",
			err,
		)
	}

	team.Role = domain.TeamRole(role)

	return &team, nil
}

func (r *TeamRepository) AddMember(
	ctx context.Context,
	teamID int64,
	userID int64,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`
      INSERT INTO team_members (team_id, user_id, role)
      VALUES (?, ?, ?)
    `,
		teamID,
		userID,
		domain.TeamRoleMember,
	)
	if err != nil {
		var mysqlError *mysqldriver.MySQLError

		if errors.As(err, &mysqlError) &&
			mysqlError.Number == 1062 {
			return domain.ErrTeamMemberAlreadyExists
		}

		return fmt.Errorf(
			"insert team member: %w",
			err,
		)
	}

	return nil
}
