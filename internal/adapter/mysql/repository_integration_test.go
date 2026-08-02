package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	"basisProject/internal/domain"
)

func TestMigrationAndRepositories(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	db := openReportTestDatabase(t, ctx)

	t.Run("migration creates all tables", func(t *testing.T) {
		rows, err := db.QueryContext(ctx, `
			SELECT table_name
			FROM information_schema.tables
			WHERE table_schema = DATABASE()
		`)
		if err != nil {
			t.Fatalf("query migrated tables: %v", err)
		}
		defer rows.Close()

		found := make(map[string]bool)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatalf("scan table: %v", err)
			}
			found[name] = true
		}
		for _, name := range []string{"users", "teams", "team_members", "tasks", "task_history", "task_comments"} {
			if !found[name] {
				t.Fatalf("migration did not create table %q", name)
			}
		}
	})

	users := NewUserRepository(db)
	alice := &domain.User{Name: "Alice", Email: "alice@example.com", PasswordHash: "hash"}
	bob := &domain.User{Name: "Bob", Email: "bob@example.com", PasswordHash: "hash"}

	t.Run("user repository creates and finds user", func(t *testing.T) {
		if err := users.Create(ctx, alice); err != nil {
			t.Fatalf("create Alice: %v", err)
		}
		if err := users.Create(ctx, bob); err != nil {
			t.Fatalf("create Bob: %v", err)
		}
		found, err := users.FindByEmail(ctx, alice.Email)
		if err != nil {
			t.Fatalf("find Alice: %v", err)
		}
		if found.ID != alice.ID || found.Name != alice.Name || found.PasswordHash != alice.PasswordHash {
			t.Fatalf("unexpected user: %#v", found)
		}
		duplicate := &domain.User{Name: "Another Alice", Email: alice.Email, PasswordHash: "hash"}
		if err := users.Create(ctx, duplicate); !errors.Is(err, domain.ErrEmailAlreadyExists) {
			t.Fatalf("expected duplicate email error, got %v", err)
		}
	})

	teams := NewTeamRepository(db)
	var team *domain.Team

	t.Run("team and owner membership are created atomically", func(t *testing.T) {
		var err error
		team, err = teams.CreateWithOwner(ctx, "Backend", alice.ID)
		if err != nil {
			t.Fatalf("create team: %v", err)
		}

		var role string
		if err := db.QueryRowContext(ctx,
			"SELECT role FROM team_members WHERE team_id = ? AND user_id = ?", team.ID, alice.ID,
		).Scan(&role); err != nil {
			t.Fatalf("find owner membership: %v", err)
		}
		if role != string(domain.TeamRoleOwner) {
			t.Fatalf("unexpected owner role %q", role)
		}
	})

	tasks := NewTaskRepository(db)
	var createdTask domain.Task

	t.Run("task repository creates filters and updates with history", func(t *testing.T) {
		if err := teams.AddMember(ctx, team.ID, bob.ID); err != nil {
			t.Fatalf("add Bob: %v", err)
		}
		createdTask = domain.Task{
			TeamID: team.ID, Title: "Implement API", Description: "Initial",
			Status: domain.TaskStatusTodo, AssigneeID: &bob.ID, CreatedBy: alice.ID,
		}
		if err := tasks.Create(ctx, &createdTask); err != nil {
			t.Fatalf("create task: %v", err)
		}

		status := domain.TaskStatusTodo
		listed, err := tasks.List(ctx, domain.TaskFilter{TeamID: team.ID, Status: &status, AssigneeID: &bob.ID, Limit: 10})
		if err != nil {
			t.Fatalf("list tasks: %v", err)
		}
		if len(listed) != 1 || listed[0].ID != createdTask.ID {
			t.Fatalf("unexpected filtered tasks: %#v", listed)
		}

		oldTitle, newTitle := createdTask.Title, "Implemented API"
		oldStatus, newStatus := string(createdTask.Status), string(domain.TaskStatusDone)
		completedAt := time.Now().UTC()
		createdTask.Title = newTitle
		createdTask.Status = domain.TaskStatusDone
		createdTask.CompletedAt = &completedAt
		changes := []domain.TaskChange{
			{FieldName: domain.TaskFieldTitle, OldValue: &oldTitle, NewValue: &newTitle},
			{FieldName: domain.TaskFieldStatus, OldValue: &oldStatus, NewValue: &newStatus},
		}
		if err := tasks.Update(ctx, &createdTask, alice.ID, changes); err != nil {
			t.Fatalf("update task: %v", err)
		}

		history, err := tasks.History(ctx, createdTask.ID)
		if err != nil {
			t.Fatalf("task history: %v", err)
		}
		if len(history) != 2 || history[0].ChangedByName != alice.Name || history[0].FieldName != domain.TaskFieldTitle || history[1].FieldName != domain.TaskFieldStatus {
			t.Fatalf("unexpected history: %#v", history)
		}
	})

	t.Run("foreign keys and unique constraints are enforced", func(t *testing.T) {
		if _, err := db.ExecContext(ctx,
			"INSERT INTO teams (name, created_by) VALUES ('Invalid', ?)", int64(999999),
		); err == nil {
			t.Fatal("expected teams.created_by foreign key violation")
		}

		if _, err := db.ExecContext(ctx,
			"INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, 'member')", team.ID, bob.ID,
		); err == nil {
			t.Fatal("expected duplicate team membership violation")
		}

		if _, err := db.ExecContext(ctx,
			"INSERT INTO task_history (task_id, changed_by, field_name) VALUES (?, ?, 'title')", int64(999999), alice.ID,
		); err == nil {
			t.Fatal("expected task_history.task_id foreign key violation")
		}

		if _, err := db.ExecContext(ctx,
			"INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, 'text')", createdTask.ID, int64(999999),
		); err == nil {
			t.Fatal("expected task_comments.user_id foreign key violation")
		}
	})
}
