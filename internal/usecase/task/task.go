package task

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"basisProject/internal/domain"
)

type Repository interface {
	IsTeamMember(
		ctx context.Context,
		teamID int64,
		userID int64,
	) (bool, error)

	Create(
		ctx context.Context,
		task *domain.Task,
	) error

	List(
		ctx context.Context,
		filter domain.TaskFilter,
	) ([]domain.Task, error)
}

type Tasks struct {
	tasks Repository
}

type CreateInput struct {
	TeamID      int64
	CreatedBy   int64
	Title       string
	Description string
	AssigneeID  *int64
}

type ListInput struct {
	TeamID      int64
	RequestedBy int64
	Status      string
	AssigneeID  *int64
	Limit       int
	Offset      int
}

func NewTasks(tasks Repository) *Tasks {
	return &Tasks{
		tasks: tasks,
	}
}

func (u *Tasks) Create(
	ctx context.Context,
	input CreateInput,
) (*domain.Task, error) {
	title := strings.TrimSpace(input.Title)

	if input.TeamID <= 0 ||
		input.CreatedBy <= 0 ||
		title == "" ||
		utf8.RuneCountInString(title) > 255 {
		return nil, domain.ErrInvalidInput
	}

	isMember, err := u.tasks.IsTeamMember(
		ctx,
		input.TeamID,
		input.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"check task creator membership: %w",
			err,
		)
	}

	if !isMember {
		return nil, domain.ErrForbidden
	}

	if input.AssigneeID != nil {
		if *input.AssigneeID <= 0 {
			return nil, domain.ErrInvalidInput
		}

		isAssigneeMember, err := u.tasks.IsTeamMember(
			ctx,
			input.TeamID,
			*input.AssigneeID,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"check assignee membership: %w",
				err,
			)
		}

		if !isAssigneeMember {
			return nil, domain.ErrAssigneeNotTeamMember
		}
	}

	task := &domain.Task{
		TeamID:      input.TeamID,
		Title:       title,
		Description: input.Description,
		Status:      domain.TaskStatusTodo,
		AssigneeID:  input.AssigneeID,
		CreatedBy:   input.CreatedBy,
	}

	if err := u.tasks.Create(ctx, task); err != nil {
		return nil, fmt.Errorf(
			"create task: %w",
			err,
		)
	}

	return task, nil
}

func (u *Tasks) List(
	ctx context.Context,
	input ListInput,
) ([]domain.Task, error) {
	if input.TeamID <= 0 ||
		input.RequestedBy <= 0 ||
		input.Limit <= 0 ||
		input.Limit > 100 ||
		input.Offset < 0 {
		return nil, domain.ErrInvalidInput
	}

	isMember, err := u.tasks.IsTeamMember(
		ctx,
		input.TeamID,
		input.RequestedBy,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"check requesting user membership: %w",
			err,
		)
	}

	if !isMember {
		return nil, domain.ErrForbidden
	}

	var status *domain.TaskStatus

	statusValue := domain.TaskStatus(
		strings.TrimSpace(input.Status),
	)

	if statusValue != "" {
		if !statusValue.Valid() {
			return nil, domain.ErrInvalidInput
		}

		status = &statusValue
	}

	if input.AssigneeID != nil &&
		*input.AssigneeID <= 0 {
		return nil, domain.ErrInvalidInput
	}

	tasks, err := u.tasks.List(
		ctx,
		domain.TaskFilter{
			TeamID:     input.TeamID,
			Status:     status,
			AssigneeID: input.AssigneeID,
			Limit:      input.Limit,
			Offset:     input.Offset,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list tasks: %w",
			err,
		)
	}

	return tasks, nil
}
