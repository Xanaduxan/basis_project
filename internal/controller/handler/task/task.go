package task

import (
	"errors"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"

	taskdto "basisProject/internal/controller/dto/task"
	httpcontroller "basisProject/internal/controller/http"
	"basisProject/internal/controller/middleware"
	"basisProject/internal/domain"
	taskusecase "basisProject/internal/usecase/task"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Task struct {
	tasks *taskusecase.Tasks
}

func NewTask(tasks *taskusecase.Tasks) *Task {
	return &Task{
		tasks: tasks,
	}
}

func (h *Task) Create(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	var request taskdto.CreateRequest

	if err := httpcontroller.ReadJSON(r, &request); err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid JSON",
		)
		return
	}

	createdTask, err := h.tasks.Create(
		r.Context(),
		taskusecase.CreateInput{
			TeamID:      request.TeamID,
			CreatedBy:   userID,
			Title:       request.Title,
			Description: request.Description,
			AssigneeID:  request.AssigneeID,
		},
	)
	if err != nil {
		writeTaskError(w, err, "create task")
		return
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusCreated,
		taskResponse(*createdTask),
	); err != nil {
		slog.Error(
			"write create task response",
			"error", err,
		)
	}
}

func (h *Task) List(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	query := r.URL.Query()

	teamID, err := parseRequiredPositiveInt64(
		query.Get("team_id"),
	)
	if err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid team_id",
		)
		return
	}

	assigneeID, err := parseOptionalPositiveInt64(
		query.Get("assignee_id"),
	)
	if err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid assignee_id",
		)
		return
	}

	limit, err := parseInteger(
		query.Get("limit"),
		defaultLimit,
	)
	if err != nil || limit <= 0 || limit > maxLimit {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"limit must be between 1 and 100",
		)
		return
	}

	offset, err := parseInteger(
		query.Get("offset"),
		0,
	)
	if err != nil || offset < 0 {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"offset must be zero or greater",
		)
		return
	}

	tasks, err := h.tasks.List(
		r.Context(),
		taskusecase.ListInput{
			TeamID:      teamID,
			RequestedBy: userID,
			Status:      strings.TrimSpace(query.Get("status")),
			AssigneeID:  assigneeID,
			Limit:       limit,
			Offset:      offset,
		},
	)
	if err != nil {
		writeTaskError(w, err, "list tasks")
		return
	}

	response := make(
		[]taskdto.Response,
		0,
		len(tasks),
	)

	for _, currentTask := range tasks {
		response = append(
			response,
			taskResponse(currentTask),
		)
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusOK,
		response,
	); err != nil {
		slog.Error(
			"write task list response",
			"error", err,
		)
	}
}

func (h *Task) Update(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	taskID, err := parseRequiredPositiveInt64(
		r.PathValue("id"),
	)
	if err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid task id",
		)
		return
	}

	var request taskdto.UpdateRequest

	if err := httpcontroller.ReadJSON(r, &request); err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid JSON",
		)
		return
	}

	updatedTask, err := h.tasks.Update(
		r.Context(),
		taskusecase.UpdateInput{
			TaskID:        taskID,
			RequestedBy:   userID,
			Title:         request.Title,
			Description:   request.Description,
			Status:        request.Status,
			AssigneeID:    request.AssigneeID.Value,
			AssigneeIDSet: request.AssigneeID.Set,
		},
	)
	if err != nil {
		writeTaskError(w, err, "update task")
		return
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusOK,
		taskResponse(*updatedTask),
	); err != nil {
		slog.Error(
			"write update task response",
			"error", err,
		)
	}
}

func (h *Task) History(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) {
	userID, ok := authenticatedUserID(w, r)
	if !ok {
		return
	}

	taskID, err := parseRequiredPositiveInt64(
		r.PathValue("id"),
	)
	if err != nil {
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid task id",
		)
		return
	}

	history, err := h.tasks.History(
		r.Context(),
		taskusecase.HistoryInput{
			TaskID:      taskID,
			RequestedBy: userID,
		},
	)
	if err != nil {
		writeTaskError(w, err, "get task history")
		return
	}

	response := make(
		[]taskdto.HistoryResponse,
		0,
		len(history),
	)

	for _, entry := range history {
		response = append(
			response,
			historyResponse(entry),
		)
	}

	if err := httpcontroller.WriteJSON(
		w,
		nethttp.StatusOK,
		response,
	); err != nil {
		slog.Error(
			"write task history response",
			"error", err,
		)
	}
}

func authenticatedUserID(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
) (int64, bool) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		httpcontroller.WriteError(
			w,
			nethttp.StatusUnauthorized,
			"unauthorized",
		)
		return 0, false
	}

	return userID, true
}

func parseRequiredPositiveInt64(
	value string,
) (int64, error) {
	parsed, err := strconv.ParseInt(
		strings.TrimSpace(value),
		10,
		64,
	)
	if err != nil || parsed <= 0 {
		return 0, errors.New("invalid positive integer")
	}

	return parsed, nil
}

func parseOptionalPositiveInt64(
	value string,
) (*int64, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil, nil
	}

	parsed, err := parseRequiredPositiveInt64(value)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func parseInteger(
	value string,
	defaultValue int,
) (int, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return defaultValue, nil
	}

	return strconv.Atoi(value)
}

func writeTaskError(
	w nethttp.ResponseWriter,
	err error,
	operation string,
) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"invalid task data",
		)

	case errors.Is(err, domain.ErrForbidden):
		httpcontroller.WriteError(
			w,
			nethttp.StatusForbidden,
			"team membership required",
		)

	case errors.Is(
		err,
		domain.ErrAssigneeNotTeamMember,
	):
		httpcontroller.WriteError(
			w,
			nethttp.StatusBadRequest,
			"assignee is not a team member",
		)

	case errors.Is(err, domain.ErrTaskNotFound):
		httpcontroller.WriteError(
			w,
			nethttp.StatusNotFound,
			"task not found",
		)

	default:
		slog.Error(
			operation,
			"error", err,
		)

		httpcontroller.WriteError(
			w,
			nethttp.StatusInternalServerError,
			"internal server error",
		)
	}
}

func taskResponse(task domain.Task) taskdto.Response {
	return taskdto.Response{
		ID:          task.ID,
		TeamID:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		AssigneeID:  task.AssigneeID,
		CreatedBy:   task.CreatedBy,
		CompletedAt: task.CompletedAt,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

func historyResponse(
	history domain.TaskHistory,
) taskdto.HistoryResponse {
	return taskdto.HistoryResponse{
		ID:     history.ID,
		TaskID: history.TaskID,
		ChangedBy: taskdto.HistoryUserResponse{
			ID:    history.ChangedBy,
			Name:  history.ChangedByName,
			Email: history.ChangedByEmail,
		},
		FieldName: history.FieldName,
		OldValue:  history.OldValue,
		NewValue:  history.NewValue,
		ChangedAt: history.ChangedAt,
	}
}
