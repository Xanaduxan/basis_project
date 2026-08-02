package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"basisProject/internal/domain"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{
		db: db,
	}
}

func (r *ReportRepository) GetTeamStatistics(
	ctx context.Context,
) ([]domain.TeamStatistics, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
      SELECT
        t.id,
        t.name,
        COUNT(DISTINCT tm.user_id) AS member_count,
        COUNT(
          DISTINCT CASE
            WHEN task.status = 'done'
              AND task.completed_at >= CURRENT_TIMESTAMP - INTERVAL 7 DAY
            THEN task.id
          END
        ) AS done_last_seven_days
      FROM teams AS t
      LEFT JOIN team_members AS tm
        ON tm.team_id = t.id
      LEFT JOIN tasks AS task
        ON task.team_id = t.id
      GROUP BY t.id, t.name
      ORDER BY t.id ASC
    `,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query team statistics: %w",
			err,
		)
	}
	defer rows.Close()

	statistics := make([]domain.TeamStatistics, 0)

	for rows.Next() {
		var item domain.TeamStatistics

		if err := rows.Scan(
			&item.TeamID,
			&item.TeamName,
			&item.MemberCount,
			&item.DoneLastSevenDays,
		); err != nil {
			return nil, fmt.Errorf(
				"scan team statistics: %w",
				err,
			)
		}

		statistics = append(statistics, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate team statistics: %w",
			err,
		)
	}

	return statistics, nil
}

func (r *ReportRepository) GetTopTaskCreators(
	ctx context.Context,
) ([]domain.TopTaskCreator, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
      WITH creator_counts AS (
        SELECT
          task.team_id,
          team.name AS team_name,
          task.created_by AS user_id,
          user.name AS user_name,
          COUNT(*) AS created_task_count
        FROM tasks AS task
        INNER JOIN teams AS team
          ON team.id = task.team_id
        INNER JOIN users AS user
          ON user.id = task.created_by
        WHERE task.created_at >=
          CURRENT_DATE - INTERVAL (DAYOFMONTH(CURRENT_DATE) - 1) DAY
          AND task.created_at <
            LAST_DAY(CURRENT_DATE) + INTERVAL 1 DAY
        GROUP BY
          task.team_id,
          team.name,
          task.created_by,
          user.name
      ),
      ranked_creators AS (
        SELECT
          team_id,
          team_name,
          user_id,
          user_name,
          created_task_count,
          ROW_NUMBER() OVER (
            PARTITION BY team_id
            ORDER BY created_task_count DESC, user_id ASC
          ) AS position
        FROM creator_counts
      )
      SELECT
        team_id,
        team_name,
        user_id,
        user_name,
        created_task_count,
        position
      FROM ranked_creators
      WHERE position <= 3
      ORDER BY team_id ASC, position ASC
    `,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query top task creators: %w",
			err,
		)
	}
	defer rows.Close()

	creators := make([]domain.TopTaskCreator, 0)

	for rows.Next() {
		var item domain.TopTaskCreator

		if err := rows.Scan(
			&item.TeamID,
			&item.TeamName,
			&item.UserID,
			&item.UserName,
			&item.CreatedTaskCount,
			&item.Position,
		); err != nil {
			return nil, fmt.Errorf(
				"scan top task creator: %w",
				err,
			)
		}

		creators = append(creators, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate top task creators: %w",
			err,
		)
	}

	return creators, nil
}

func (r *ReportRepository) FindTasksWithInvalidAssignees(
	ctx context.Context,
) ([]domain.InvalidTaskAssignee, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
      SELECT
        task.id,
        task.team_id,
        task.title,
        task.assignee_id
      FROM tasks AS task
      WHERE task.assignee_id IS NOT NULL
        AND NOT EXISTS (
          SELECT 1
          FROM team_members AS tm
          WHERE tm.team_id = task.team_id
            AND tm.user_id = task.assignee_id
        )
      ORDER BY task.id ASC
    `,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query tasks with invalid assignees: %w",
			err,
		)
	}
	defer rows.Close()

	tasks := make([]domain.InvalidTaskAssignee, 0)

	for rows.Next() {
		var item domain.InvalidTaskAssignee

		if err := rows.Scan(
			&item.TaskID,
			&item.TeamID,
			&item.Title,
			&item.AssigneeID,
		); err != nil {
			return nil, fmt.Errorf(
				"scan task with invalid assignee: %w",
				err,
			)
		}

		tasks = append(tasks, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate tasks with invalid assignees: %w",
			err,
		)
	}

	return tasks, nil
}
