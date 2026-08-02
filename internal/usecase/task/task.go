package task

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
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

	FindByID(
		ctx context.Context,
		taskID int64,
	) (*domain.Task, error)

	Update(
		ctx context.Context,
		task *domain.Task,
		changedBy int64,
		changes []domain.TaskChange,
	) error

	History(
		ctx context.Context,
		taskID int64,
	) ([]domain.TaskHistory, error)
}

type Cache interface {
	Get(
		ctx context.Context,
		filter domain.TaskFilter,
	) (
		tasks []domain.Task,
		hit bool,
		version int64,
		err error,
	)

	Set(
		ctx context.Context,
		filter domain.TaskFilter,
		version int64,
		tasks []domain.Task,
	) error

	InvalidateTeam(
		ctx context.Context,
		teamID int64,
	) error
}

type Tasks struct {
	tasks Repository
	cache Cache
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

type UpdateInput struct {
	TaskID        int64
	RequestedBy   int64
	Title         *string
	Description   *string
	Status        *string
	AssigneeID    *int64
	AssigneeIDSet bool
}

type HistoryInput struct {
	TaskID      int64
	RequestedBy int64
}

func NewTasks(
	tasks Repository,
	cache Cache,
) *Tasks {
	return &Tasks{
		tasks: tasks,
		cache: cache,
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

	u.invalidateCache(ctx, input.TeamID)

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

	filter := domain.TaskFilter{
		TeamID:     input.TeamID,
		Status:     status,
		AssigneeID: input.AssigneeID,
		Limit:      input.Limit,
		Offset:     input.Offset,
	}

	cachedTasks, hit, version, cacheErr := u.cache.Get(
		ctx,
		filter,
	)

	cacheAvailable := cacheErr == nil

	if cacheErr != nil {
		slog.Warn(
			"task cache unavailable, use mysql",
			"team_id", input.TeamID,
			"error", cacheErr,
		)
	} else if hit {
		slog.Info(
			"task cache hit",
			"team_id", input.TeamID,
			"status", input.Status,
			"assignee_id", input.AssigneeID,
			"limit", input.Limit,
			"offset", input.Offset,
		)

		return cachedTasks, nil
	} else {
		slog.Info(
			"task cache miss",
			"team_id", input.TeamID,
			"status", input.Status,
			"assignee_id", input.AssigneeID,
			"limit", input.Limit,
			"offset", input.Offset,
		)
	}

	tasks, err := u.tasks.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf(
			"list tasks: %w",
			err,
		)
	}

	if cacheAvailable {
		if err := u.cache.Set(
			ctx,
			filter,
			version,
			tasks,
		); err != nil {
			slog.Warn(
				"store task list in cache",
				"team_id", input.TeamID,
				"error", err,
			)
		}
	}

	return tasks, nil
}

func (u *Tasks) Update(
	ctx context.Context,
	input UpdateInput,
) (*domain.Task, error) {
	if input.TaskID <= 0 || input.RequestedBy <= 0 {
		return nil, domain.ErrInvalidInput
	}

	if input.Title == nil &&
		input.Description == nil &&
		input.Status == nil &&
		!input.AssigneeIDSet {
		return nil, domain.ErrInvalidInput
	}

	currentTask, err := u.tasks.FindByID(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}

	isMember, err := u.tasks.IsTeamMember(
		ctx,
		currentTask.TeamID,
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

	updatedTask := *currentTask
	changes := make([]domain.TaskChange, 0, 4)

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" || utf8.RuneCountInString(title) > 255 {
			return nil, domain.ErrInvalidInput
		}

		if title != currentTask.Title {
			changes = append(
				changes,
				newTaskChange(
					domain.TaskFieldTitle,
					currentTask.Title,
					title,
				),
			)
			updatedTask.Title = title
		}
	}

	if input.Description != nil &&
		*input.Description != currentTask.Description {
		changes = append(
			changes,
			newTaskChange(
				domain.TaskFieldDescription,
				currentTask.Description,
				*input.Description,
			),
		)
		updatedTask.Description = *input.Description
	}

	if input.Status != nil {
		status := domain.TaskStatus(
			strings.TrimSpace(*input.Status),
		)
		if !status.Valid() {
			return nil, domain.ErrInvalidInput
		}

		if status != currentTask.Status {
			changes = append(
				changes,
				newTaskChange(
					domain.TaskFieldStatus,
					string(currentTask.Status),
					string(status),
				),
			)
			updatedTask.Status = status

			if status == domain.TaskStatusDone {
				completedAt := time.Now().UTC()
				updatedTask.CompletedAt = &completedAt
			} else {
				updatedTask.CompletedAt = nil
			}
		}
	}

	if input.AssigneeIDSet {
		if input.AssigneeID != nil {
			if *input.AssigneeID <= 0 {
				return nil, domain.ErrInvalidInput
			}

			isAssigneeMember, err := u.tasks.IsTeamMember(
				ctx,
				currentTask.TeamID,
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

		if !sameOptionalInt64(
			input.AssigneeID,
			currentTask.AssigneeID,
		) {
			changes = append(
				changes,
				domain.TaskChange{
					FieldName: domain.TaskFieldAssigneeID,
					OldValue: optionalInt64String(
						currentTask.AssigneeID,
					),
					NewValue: optionalInt64String(
						input.AssigneeID,
					),
				},
			)
			updatedTask.AssigneeID = input.AssigneeID
		}
	}

	if len(changes) == 0 {
		return currentTask, nil
	}

	if err := u.tasks.Update(
		ctx,
		&updatedTask,
		input.RequestedBy,
		changes,
	); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	u.invalidateCache(ctx, updatedTask.TeamID)

	return &updatedTask, nil
}

func (u *Tasks) History(
	ctx context.Context,
	input HistoryInput,
) ([]domain.TaskHistory, error) {
	if input.TaskID <= 0 || input.RequestedBy <= 0 {
		return nil, domain.ErrInvalidInput
	}

	task, err := u.tasks.FindByID(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}

	isMember, err := u.tasks.IsTeamMember(
		ctx,
		task.TeamID,
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

	history, err := u.tasks.History(ctx, input.TaskID)
	if err != nil {
		return nil, fmt.Errorf("get task history: %w", err)
	}

	return history, nil
}

func newTaskChange(
	fieldName string,
	oldValue string,
	newValue string,
) domain.TaskChange {
	return domain.TaskChange{
		FieldName: fieldName,
		OldValue:  stringPointer(oldValue),
		NewValue:  stringPointer(newValue),
	}
}

func stringPointer(value string) *string {
	return &value
}

func optionalInt64String(value *int64) *string {
	if value == nil {
		return nil
	}

	result := strconv.FormatInt(*value, 10)
	return &result
}

func sameOptionalInt64(first *int64, second *int64) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}

	return *first == *second
}

func (u *Tasks) invalidateCache(
	ctx context.Context,
	teamID int64,
) {
	if err := u.cache.InvalidateTeam(
		ctx,
		teamID,
	); err != nil {
		slog.Warn(
			"invalidate task cache",
			"team_id", teamID,
			"error", err,
		)
		return
	}

	slog.Info(
		"task cache invalidated",
		"team_id", teamID,
	)
}
