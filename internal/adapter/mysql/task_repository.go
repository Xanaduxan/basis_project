package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"basisProject/internal/domain"
)

type TaskRepository struct {
	db *sql.DB
}

type taskScanner interface {
	Scan(destination ...any) error
}

type taskQueryer interface {
	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
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

	createdTask, err := r.FindByID(ctx, taskID)
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

func (r *TaskRepository) FindByID(
	ctx context.Context,
	taskID int64,
) (*domain.Task, error) {
	task, err := findTaskByID(ctx, r.db, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}

		return nil, fmt.Errorf(
			"find task by id: %w",
			err,
		)
	}

	return task, nil
}

func (r *TaskRepository) Update(
	ctx context.Context,
	task *domain.Task,
	changedBy int64,
	changes []domain.TaskChange,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin update task transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	assignments := make([]string, 0, len(changes)+1)
	arguments := make([]any, 0, len(changes)+2)

	for _, change := range changes {
		switch change.FieldName {
		case domain.TaskFieldTitle:
			assignments = append(assignments, "title = ?")
			arguments = append(arguments, task.Title)

		case domain.TaskFieldDescription:
			assignments = append(assignments, "description = ?")
			arguments = append(arguments, task.Description)

		case domain.TaskFieldStatus:
			assignments = append(
				assignments,
				"status = ?",
				"completed_at = ?",
			)
			arguments = append(
				arguments,
				string(task.Status),
				nullableTime(task.CompletedAt),
			)

		case domain.TaskFieldAssigneeID:
			assignments = append(assignments, "assignee_id = ?")
			arguments = append(
				arguments,
				nullableInt64(task.AssigneeID),
			)

		default:
			return fmt.Errorf(
				"unsupported task field %q",
				change.FieldName,
			)
		}
	}

	if len(assignments) == 0 {
		return nil
	}

	arguments = append(arguments, task.ID)

	result, err := tx.ExecContext(
		ctx,
		"UPDATE tasks SET "+
			strings.Join(assignments, ", ")+
			" WHERE id = ?",
		arguments...,
	)
	if err != nil {
		return fmt.Errorf("update task row: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get updated task row count: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return domain.ErrTaskNotFound
	}

	for _, change := range changes {
		_, err := tx.ExecContext(
			ctx,
			`
        INSERT INTO task_history (
          task_id,
          changed_by,
          field_name,
          old_value,
          new_value
        )
        VALUES (?, ?, ?, ?, ?)
      `,
			task.ID,
			changedBy,
			change.FieldName,
			nullableString(change.OldValue),
			nullableString(change.NewValue),
		)
		if err != nil {
			return fmt.Errorf(
				"insert task history: %w",
				err,
			)
		}
	}

	updatedTask, err := findTaskByID(ctx, tx, task.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrTaskNotFound
		}

		return fmt.Errorf(
			"load updated task: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit update task transaction: %w",
			err,
		)
	}

	*task = *updatedTask

	return nil
}

func (r *TaskRepository) History(
	ctx context.Context,
	taskID int64,
) ([]domain.TaskHistory, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
      SELECT
        th.id,
        th.task_id,
        th.changed_by,
        u.name,
        u.email,
        th.field_name,
        th.old_value,
        th.new_value,
        th.changed_at
      FROM task_history AS th
      INNER JOIN users AS u
        ON u.id = th.changed_by
      WHERE th.task_id = ?
      ORDER BY th.changed_at ASC, th.id ASC
    `,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query task history: %w",
			err,
		)
	}
	defer rows.Close()

	history := make([]domain.TaskHistory, 0)

	for rows.Next() {
		var entry domain.TaskHistory
		var oldValue sql.NullString
		var newValue sql.NullString

		if err := rows.Scan(
			&entry.ID,
			&entry.TaskID,
			&entry.ChangedBy,
			&entry.ChangedByName,
			&entry.ChangedByEmail,
			&entry.FieldName,
			&oldValue,
			&newValue,
			&entry.ChangedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"scan task history: %w",
				err,
			)
		}

		if oldValue.Valid {
			value := oldValue.String
			entry.OldValue = &value
		}

		if newValue.Valid {
			value := newValue.String
			entry.NewValue = &value
		}

		history = append(history, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate task history: %w",
			err,
		)
	}

	return history, nil
}

func findTaskByID(
	ctx context.Context,
	queryer taskQueryer,
	taskID int64,
) (*domain.Task, error) {
	row := queryer.QueryRowContext(
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

	return scanTask(row)
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

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}

	return *value
}
