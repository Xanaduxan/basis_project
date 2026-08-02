package task

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"basisProject/internal/domain"
)

type taskRepositoryMock struct {
	members    map[int64]bool
	task       *domain.Task
	listed     []domain.Task
	history    []domain.TaskHistory
	created    *domain.Task
	updated    *domain.Task
	changes    []domain.TaskChange
	lastFilter domain.TaskFilter
	listCalls  int
}

func (m *taskRepositoryMock) IsTeamMember(_ context.Context, _ int64, userID int64) (bool, error) {
	return m.members[userID], nil
}
func (m *taskRepositoryMock) Create(_ context.Context, task *domain.Task) error {
	m.created = task
	task.ID = 11
	return nil
}
func (m *taskRepositoryMock) List(_ context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	m.listCalls++
	m.lastFilter = filter
	return m.listed, nil
}
func (m *taskRepositoryMock) FindByID(context.Context, int64) (*domain.Task, error) {
	if m.task == nil {
		return nil, domain.ErrTaskNotFound
	}
	copy := *m.task
	return &copy, nil
}
func (m *taskRepositoryMock) Update(_ context.Context, task *domain.Task, _ int64, changes []domain.TaskChange) error {
	copy := *task
	m.updated = &copy
	m.changes = append([]domain.TaskChange(nil), changes...)
	return nil
}
func (m *taskRepositoryMock) History(context.Context, int64) ([]domain.TaskHistory, error) {
	return m.history, nil
}

type taskCacheMock struct {
	tasks           []domain.Task
	hit             bool
	getErr          error
	setCalls        int
	invalidatedTeam int64
}

func (m *taskCacheMock) Get(context.Context, domain.TaskFilter) ([]domain.Task, bool, int64, error) {
	return m.tasks, m.hit, 4, m.getErr
}
func (m *taskCacheMock) Set(context.Context, domain.TaskFilter, int64, []domain.Task) error {
	m.setCalls++
	return nil
}
func (m *taskCacheMock) InvalidateTeam(_ context.Context, teamID int64) error {
	m.invalidatedTeam = teamID
	return nil
}

func TestTasksCreate(t *testing.T) {
	t.Run("member creates task and invalidates cache", func(t *testing.T) {
		assigneeID := int64(2)
		repository := &taskRepositoryMock{members: map[int64]bool{1: true, 2: true}}
		cache := &taskCacheMock{}
		service := NewTasks(repository, cache)

		created, err := service.Create(context.Background(), CreateInput{
			TeamID: 3, CreatedBy: 1, Title: "  Implement API  ", Description: "work", AssigneeID: &assigneeID,
		})
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		if created.ID != 11 || created.Title != "Implement API" || created.Status != domain.TaskStatusTodo {
			t.Fatalf("unexpected task: %#v", created)
		}
		if cache.invalidatedTeam != 3 {
			t.Fatalf("cache was not invalidated")
		}
	})

	t.Run("rejects assignee outside team", func(t *testing.T) {
		assigneeID := int64(9)
		service := NewTasks(&taskRepositoryMock{members: map[int64]bool{1: true}}, &taskCacheMock{})
		_, err := service.Create(context.Background(), CreateInput{TeamID: 3, CreatedBy: 1, Title: "Task", AssigneeID: &assigneeID})
		if !errors.Is(err, domain.ErrAssigneeNotTeamMember) {
			t.Fatalf("expected invalid assignee, got %v", err)
		}
	})

	t.Run("rejects non-member creator", func(t *testing.T) {
		service := NewTasks(&taskRepositoryMock{members: map[int64]bool{}}, &taskCacheMock{})
		_, err := service.Create(context.Background(), CreateInput{TeamID: 3, CreatedBy: 1, Title: "Task"})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected forbidden, got %v", err)
		}
	})
}

func TestTasksList(t *testing.T) {
	status := domain.TaskStatusDone
	assigneeID := int64(2)
	databaseTasks := []domain.Task{{ID: 1, Status: status}}

	t.Run("passes filters and pagination to repository", func(t *testing.T) {
		repository := &taskRepositoryMock{members: map[int64]bool{1: true}, listed: databaseTasks}
		cache := &taskCacheMock{}
		service := NewTasks(repository, cache)

		got, err := service.List(context.Background(), ListInput{
			TeamID: 3, RequestedBy: 1, Status: "done", AssigneeID: &assigneeID, Limit: 20, Offset: 40,
		})
		if err != nil {
			t.Fatalf("list tasks: %v", err)
		}
		if !reflect.DeepEqual(got, databaseTasks) {
			t.Fatalf("unexpected tasks: %#v", got)
		}
		if repository.lastFilter.Status == nil || *repository.lastFilter.Status != status || repository.lastFilter.Limit != 20 || repository.lastFilter.Offset != 40 {
			t.Fatalf("unexpected filter: %#v", repository.lastFilter)
		}
		if cache.setCalls != 1 {
			t.Fatalf("expected cache set")
		}
	})

	t.Run("returns cached tasks without mysql", func(t *testing.T) {
		repository := &taskRepositoryMock{members: map[int64]bool{1: true}}
		cached := []domain.Task{{ID: 8}}
		service := NewTasks(repository, &taskCacheMock{tasks: cached, hit: true})
		got, err := service.List(context.Background(), ListInput{TeamID: 3, RequestedBy: 1, Limit: 20})
		if err != nil {
			t.Fatalf("list cached tasks: %v", err)
		}
		if !reflect.DeepEqual(got, cached) || repository.listCalls != 0 {
			t.Fatalf("cache hit did not bypass mysql")
		}
	})

	t.Run("falls back to mysql when redis fails", func(t *testing.T) {
		repository := &taskRepositoryMock{members: map[int64]bool{1: true}, listed: databaseTasks}
		cache := &taskCacheMock{getErr: errors.New("redis unavailable")}
		service := NewTasks(repository, cache)
		got, err := service.List(context.Background(), ListInput{TeamID: 3, RequestedBy: 1, Limit: 20})
		if err != nil {
			t.Fatalf("fallback list: %v", err)
		}
		if !reflect.DeepEqual(got, databaseTasks) || repository.listCalls != 1 || cache.setCalls != 0 {
			t.Fatalf("unexpected fallback behavior")
		}
	})
}

func TestTasksUpdate(t *testing.T) {
	now := time.Now().UTC()
	current := &domain.Task{ID: 5, TeamID: 3, Title: "Old", Description: "Before", Status: domain.TaskStatusInProgress, CreatedBy: 1, CreatedAt: now, UpdatedAt: now}

	t.Run("updates changed fields, history input and cache", func(t *testing.T) {
		repository := &taskRepositoryMock{members: map[int64]bool{1: true, 2: true}, task: current}
		cache := &taskCacheMock{}
		service := NewTasks(repository, cache)
		title, description, status := "New", "After", "done"
		assigneeID := int64(2)

		updated, err := service.Update(context.Background(), UpdateInput{
			TaskID: 5, RequestedBy: 1, Title: &title, Description: &description,
			Status: &status, AssigneeID: &assigneeID, AssigneeIDSet: true,
		})
		if err != nil {
			t.Fatalf("update task: %v", err)
		}
		if updated.Status != domain.TaskStatusDone || updated.CompletedAt == nil || len(repository.changes) != 4 {
			t.Fatalf("unexpected update: task=%#v changes=%#v", updated, repository.changes)
		}
		if cache.invalidatedTeam != 3 {
			t.Fatalf("cache was not invalidated")
		}
	})

	t.Run("does not persist unchanged fields", func(t *testing.T) {
		repository := &taskRepositoryMock{members: map[int64]bool{1: true}, task: current}
		cache := &taskCacheMock{}
		service := NewTasks(repository, cache)
		title := "Old"
		updated, err := service.Update(context.Background(), UpdateInput{TaskID: 5, RequestedBy: 1, Title: &title})
		if err != nil {
			t.Fatalf("update unchanged task: %v", err)
		}
		if updated.Title != "Old" || repository.updated != nil || cache.invalidatedTeam != 0 {
			t.Fatalf("unchanged task was persisted")
		}
	})

	t.Run("clears completed at when leaving done", func(t *testing.T) {
		done := *current
		done.Status = domain.TaskStatusDone
		done.CompletedAt = &now
		repository := &taskRepositoryMock{members: map[int64]bool{1: true}, task: &done}
		service := NewTasks(repository, &taskCacheMock{})
		status := "todo"
		updated, err := service.Update(context.Background(), UpdateInput{TaskID: 5, RequestedBy: 1, Status: &status})
		if err != nil {
			t.Fatalf("leave done: %v", err)
		}
		if updated.CompletedAt != nil {
			t.Fatalf("completed_at was not cleared")
		}
	})

	t.Run("rejects new assignee outside team", func(t *testing.T) {
		repository := &taskRepositoryMock{members: map[int64]bool{1: true}, task: current}
		service := NewTasks(repository, &taskCacheMock{})
		assigneeID := int64(9)
		_, err := service.Update(context.Background(), UpdateInput{TaskID: 5, RequestedBy: 1, AssigneeID: &assigneeID, AssigneeIDSet: true})
		if !errors.Is(err, domain.ErrAssigneeNotTeamMember) {
			t.Fatalf("expected invalid assignee, got %v", err)
		}
	})
}

func TestTasksHistory(t *testing.T) {
	expected := []domain.TaskHistory{{ID: 1, TaskID: 5, FieldName: domain.TaskFieldTitle}}
	repository := &taskRepositoryMock{members: map[int64]bool{1: true}, task: &domain.Task{ID: 5, TeamID: 3}, history: expected}
	service := NewTasks(repository, &taskCacheMock{})
	got, err := service.History(context.Background(), HistoryInput{TaskID: 5, RequestedBy: 1})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected history: %#v", got)
	}
}
