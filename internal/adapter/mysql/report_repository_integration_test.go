package mysql

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"basisProject/internal/domain"
)

func TestReportRepository(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Minute,
	)
	defer cancel()

	db := openReportTestDatabase(t, ctx)
	seedReportTestData(t, ctx, db)

	repository := NewReportRepository(db)

	t.Run("team statistics", func(t *testing.T) {
		statistics, err := repository.GetTeamStatistics(ctx)
		if err != nil {
			t.Fatalf("get team statistics: %v", err)
		}

		expected := []domain.TeamStatistics{
			{
				TeamID:            1,
				TeamName:          "Backend",
				MemberCount:       4,
				DoneLastSevenDays: 1,
			},
			{
				TeamID:            2,
				TeamName:          "Frontend",
				MemberCount:       2,
				DoneLastSevenDays: 1,
			},
			{
				TeamID:            3,
				TeamName:          "Empty team",
				MemberCount:       1,
				DoneLastSevenDays: 0,
			},
		}

		if !reflect.DeepEqual(statistics, expected) {
			t.Fatalf(
				"unexpected team statistics:\nwant: %#v\ngot:  %#v",
				expected,
				statistics,
			)
		}
	})

	t.Run("top three creators per team", func(t *testing.T) {
		creators, err := repository.GetTopTaskCreators(ctx)
		if err != nil {
			t.Fatalf("get top task creators: %v", err)
		}

		expected := []domain.TopTaskCreator{
			{
				TeamID:           1,
				TeamName:         "Backend",
				UserID:           1,
				UserName:         "Alice",
				CreatedTaskCount: 4,
				Position:         1,
			},
			{
				TeamID:           1,
				TeamName:         "Backend",
				UserID:           2,
				UserName:         "Bob",
				CreatedTaskCount: 3,
				Position:         2,
			},
			{
				TeamID:           1,
				TeamName:         "Backend",
				UserID:           3,
				UserName:         "Carol",
				CreatedTaskCount: 2,
				Position:         3,
			},
			{
				TeamID:           2,
				TeamName:         "Frontend",
				UserID:           4,
				UserName:         "Dave",
				CreatedTaskCount: 2,
				Position:         1,
			},
			{
				TeamID:           2,
				TeamName:         "Frontend",
				UserID:           5,
				UserName:         "Eve",
				CreatedTaskCount: 1,
				Position:         2,
			},
		}

		if !reflect.DeepEqual(creators, expected) {
			t.Fatalf(
				"unexpected top task creators:\nwant: %#v\ngot:  %#v",
				expected,
				creators,
			)
		}
	})

	t.Run("tasks with invalid assignees", func(t *testing.T) {
		tasks, err := repository.FindTasksWithInvalidAssignees(ctx)
		if err != nil {
			t.Fatalf(
				"find tasks with invalid assignees: %v",
				err,
			)
		}

		expected := []domain.InvalidTaskAssignee{
			{
				TaskID:     16,
				TeamID:     1,
				Title:      "Invalid backend assignee",
				AssigneeID: 5,
			},
			{
				TaskID:     17,
				TeamID:     2,
				Title:      "Invalid frontend assignee",
				AssigneeID: 1,
			},
		}

		if !reflect.DeepEqual(tasks, expected) {
			t.Fatalf(
				"unexpected tasks with invalid assignees:\nwant: %#v\ngot:  %#v",
				expected,
				tasks,
			)
		}
	})
}

func openReportTestDatabase(
	t *testing.T,
	ctx context.Context,
) *sql.DB {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("get integration test file path")
	}

	migrationPath := filepath.Join(
		filepath.Dir(currentFile),
		"..",
		"..",
		"migrations",
		"000001_init_schema.up.sql",
	)

	container, err := testmysql.Run(
		ctx,
		"mysql:8.4",
		testmysql.WithDatabase("task_manager"),
		testmysql.WithUsername("task_user"),
		testmysql.WithPassword("task_password"),
		testmysql.WithScripts(migrationPath),
	)
	if err != nil {
		t.Fatalf("start MySQL testcontainer: %v", err)
	}

	t.Cleanup(func() {
		cleanupContext, cancel := context.WithTimeout(
			context.Background(),
			30*time.Second,
		)
		defer cancel()

		if err := container.Terminate(cleanupContext); err != nil {
			t.Errorf("terminate MySQL testcontainer: %v", err)
		}
	})

	connectionString, err := container.ConnectionString(
		ctx,
		"parseTime=true",
		"multiStatements=true",
	)
	if err != nil {
		t.Fatalf("get MySQL connection string: %v", err)
	}

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		t.Fatalf("open test MySQL: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test MySQL: %v", err)
		}
	})

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping test MySQL: %v", err)
	}

	return db
}

func seedReportTestData(
	t *testing.T,
	ctx context.Context,
	db *sql.DB,
) {
	t.Helper()

	_, err := db.ExecContext(
		ctx,
		`
      INSERT INTO users (id, name, email, password_hash)
      VALUES
        (1, 'Alice', 'alice@example.com', 'hash'),
        (2, 'Bob', 'bob@example.com', 'hash'),
        (3, 'Carol', 'carol@example.com', 'hash'),
        (4, 'Dave', 'dave@example.com', 'hash'),
        (5, 'Eve', 'eve@example.com', 'hash');

      INSERT INTO teams (id, name, created_by)
      VALUES
        (1, 'Backend', 1),
        (2, 'Frontend', 4),
        (3, 'Empty team', 1);

      INSERT INTO team_members (team_id, user_id, role)
      VALUES
        (1, 1, 'owner'),
        (1, 2, 'member'),
        (1, 3, 'member'),
        (1, 4, 'member'),
        (2, 4, 'owner'),
        (2, 5, 'member'),
        (3, 1, 'owner');

      INSERT INTO tasks (
        id,
        team_id,
        title,
        status,
        assignee_id,
        created_by,
        completed_at,
        created_at
      )
      VALUES
        (1, 1, 'A1', 'done', NULL, 1,
          CURRENT_TIMESTAMP - INTERVAL 1 DAY, CURRENT_TIMESTAMP),
        (2, 1, 'A2', 'todo', NULL, 1, NULL, CURRENT_TIMESTAMP),
        (3, 1, 'A3', 'in_progress', NULL, 1, NULL, CURRENT_TIMESTAMP),
        (4, 1, 'A4', 'todo', NULL, 1, NULL, CURRENT_TIMESTAMP),
        (5, 1, 'B1', 'todo', NULL, 2, NULL, CURRENT_TIMESTAMP),
        (6, 1, 'B2', 'todo', NULL, 2, NULL, CURRENT_TIMESTAMP),
        (7, 1, 'B3', 'todo', NULL, 2, NULL, CURRENT_TIMESTAMP),
        (8, 1, 'C1', 'todo', NULL, 3, NULL, CURRENT_TIMESTAMP),
        (9, 1, 'C2', 'todo', NULL, 3, NULL, CURRENT_TIMESTAMP),
        (10, 1, 'D1', 'todo', NULL, 4, NULL, CURRENT_TIMESTAMP),
        (11, 1, 'Old done', 'done', NULL, 4,
          CURRENT_TIMESTAMP - INTERVAL 8 DAY,
          CURRENT_DATE - INTERVAL DAYOFMONTH(CURRENT_DATE) DAY),
        (12, 2, 'D2', 'done', NULL, 4,
          CURRENT_TIMESTAMP - INTERVAL 1 DAY, CURRENT_TIMESTAMP),
        (13, 2, 'D3', 'todo', NULL, 4, NULL, CURRENT_TIMESTAMP),
        (14, 2, 'E1', 'todo', NULL, 5, NULL, CURRENT_TIMESTAMP),
        (15, 1, 'Valid backend assignee', 'todo', 2, 1, NULL,
          CURRENT_DATE - INTERVAL DAYOFMONTH(CURRENT_DATE) DAY),
        (16, 1, 'Invalid backend assignee', 'todo', 5, 1, NULL,
          CURRENT_DATE - INTERVAL DAYOFMONTH(CURRENT_DATE) DAY),
        (17, 2, 'Invalid frontend assignee', 'todo', 1, 4, NULL,
          CURRENT_DATE - INTERVAL DAYOFMONTH(CURRENT_DATE) DAY);
    `,
	)
	if err != nil {
		t.Fatalf("seed report test data: %v", err)
	}
}
