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

const (
	TaskFieldTitle       = "title"
	TaskFieldDescription = "description"
	TaskFieldStatus      = "status"
	TaskFieldAssigneeID  = "assignee_id"
)

var (
	ErrAssigneeNotTeamMember = errors.New(
		"assignee is not a team member",
	)
	ErrTaskNotFound = errors.New("task not found")
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

type TaskChange struct {
	FieldName string
	OldValue  *string
	NewValue  *string
}

type TaskHistory struct {
	ID             int64
	TaskID         int64
	ChangedBy      int64
	ChangedByName  string
	ChangedByEmail string
	FieldName      string
	OldValue       *string
	NewValue       *string
	ChangedAt      time.Time
}
