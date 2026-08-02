package task

import (
	"bytes"
	"encoding/json"
	"time"
)

type CreateRequest struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	AssigneeID  *int64 `json:"assignee_id"`
}

type UpdateRequest struct {
	Title       *string       `json:"title"`
	Description *string       `json:"description"`
	Status      *string       `json:"status"`
	AssigneeID  NullableInt64 `json:"assignee_id"`
}

type NullableInt64 struct {
	Value *int64
	Set   bool
}

func (value *NullableInt64) UnmarshalJSON(data []byte) error {
	value.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		value.Value = nil
		return nil
	}

	var parsed int64

	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	value.Value = &parsed

	return nil
}

type Response struct {
	ID          int64      `json:"id"`
	TeamID      int64      `json:"team_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	AssigneeID  *int64     `json:"assignee_id"`
	CreatedBy   int64      `json:"created_by"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type HistoryUserResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type HistoryResponse struct {
	ID        int64               `json:"id"`
	TaskID    int64               `json:"task_id"`
	ChangedBy HistoryUserResponse `json:"changed_by"`
	FieldName string              `json:"field_name"`
	OldValue  *string             `json:"old_value"`
	NewValue  *string             `json:"new_value"`
	ChangedAt time.Time           `json:"changed_at"`
}
