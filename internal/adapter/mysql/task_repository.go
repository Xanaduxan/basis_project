package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"basisProject/internal/domain"
)

type TaskRepository struct {
	db *sql.DB
}

type taskScanner interface {
	Scan(destination ...any) error
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{
		db: db,
	}
}

func (r *TaskRepository) IsTeamMember(
	ctx context.Context,
	teamID int64,
	userID int64,
) (bool, error) {
	var exists int

	err := r.db.QueryRowContext(
		ctx,
		`
      SELECT EXISTS(
        SELECT 1
        FROM team_members
        WHERE team_id = ? AND user_id = ?
      )
    `,
		teamID,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf(
			"check team membership: %w",
			err,
		)
	}

	return exists == 1, nil
}

func (r *TaskRepository) Create(
	ctx context.Context,
	task *domain.Task,
) error {
	var assigneeID any

	if task.AssigneeID != nil {
		assigneeID = *task.AssigneeID
	}

	result, err := r.db.ExecContext(
		ctx,
		`
      INSERT INTO tasks (
        team_id,
        title,
        description,
        status,
        assignee_id,
        created_by
      )
      VALUES (?, ?, ?, ?, ?, ?)
    `,
		task.TeamID,
		task.Title,
		task.Description,
		string(task.Status),
		assigneeID,
		task.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf(
			"get inserted task id: %w",
			err,
		)
	}

	createdTask, err := r.findByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf(
			"load created task: %w",
			err,
		)
	}

	*task = *createdTask

	return nil
}

func (r *TaskRepository) List(
	ctx context.Context,
	filter domain.TaskFilter,
) ([]domain.Task, error) {
	var query strings.Builder

	query.WriteString(`
    SELECT
      t.id,
      t.team_id,
      t.title,
      COALESCE(t.description, ''),
      t.status,
      t.assignee_id,
      t.created_by,
      t.completed_at,
      t.created_at,
      t.updated_at
    FROM tasks AS t
    WHERE t.team_id = ?
  `)

	arguments := []any{
		filter.TeamID,
	}

	if filter.Status != nil {
		query.WriteString(" AND t.status = ?")
		arguments = append(
			arguments,
			string(*filter.Status),
		)
	}

	if filter.AssigneeID != nil {
		query.WriteString(" AND t.assignee_id = ?")
		arguments = append(
			arguments,
			*filter.AssigneeID,
		)
	}

	query.WriteString(`
    ORDER BY t.created_at DESC, t.id DESC
    LIMIT ? OFFSET ?
  `)

	arguments = append(
		arguments,
		filter.Limit,
		filter.Offset,
	)

	rows, err := r.db.QueryContext(
		ctx,
		query.String(),
		arguments...,
	)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)

	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate tasks: %w",
			err,
		)
	}

	return tasks, nil
}

func (r *TaskRepository) findByID(
	ctx context.Context,
	taskID int64,
) (*domain.Task, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
      SELECT
        t.id,
        t.team_id,
        t.title,
        COALESCE(t.description, ''),
        t.status,
        t.assignee_id,
        t.created_by,
        t.completed_at,
        t.created_at,
        t.updated_at
      FROM tasks AS t
      WHERE t.id = ?
    `,
		taskID,
	)

	task, err := scanTask(row)
	if err != nil {
		return nil, fmt.Errorf(
			"find task by id: %w",
			err,
		)
	}

	return task, nil
}

func scanTask(scanner taskScanner) (*domain.Task, error) {
	var task domain.Task
	var status string
	var assigneeID sql.NullInt64
	var completedAt sql.NullTime

	err := scanner.Scan(
		&task.ID,
		&task.TeamID,
		&task.Title,
		&task.Description,
		&status,
		&assigneeID,
		&task.CreatedBy,
		&completedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	task.Status = domain.TaskStatus(status)

	if assigneeID.Valid {
		value := assigneeID.Int64
		task.AssigneeID = &value
	}

	if completedAt.Valid {
		value := completedAt.Time
		task.CompletedAt = &value
	}

	return &task, nil
}
