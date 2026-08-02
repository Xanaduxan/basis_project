package domain

import (
	"errors"
	"time"
)

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

var (
	ErrAssigneeNotTeamMember = errors.New(
		"assignee is not a team member",
	)
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusTodo,
		TaskStatusInProgress,
		TaskStatusDone:
		return true

	default:
		return false
	}
}

type Task struct {
	ID          int64
	TeamID      int64
	Title       string
	Description string
	Status      TaskStatus
	AssigneeID  *int64
	CreatedBy   int64
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TaskFilter struct {
	TeamID     int64
	Status     *TaskStatus
	AssigneeID *int64
	Limit      int
	Offset     int
}
