package tasks

import (
	"encoding/json"
	"time"
	"uuid"
)

type createTaskRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
	DueTime     *string `json:"due_time"`
}

type setTaskScheduleRequest struct {
	DueDate json.RawMessage `json:"due_date"`
	DueTime json.RawMessage `json:"due_time"`
}

type setTaskCompletedRequest struct {
	Completed *bool `json:"completed"`
}

type TaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Priority    Priority   `json:"priority"`
	DueDate     *string    `json:"due_date"`
	DueTime     *string    `json:"due_time"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Version     int64      `json:"version"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func NewTaskResponse(task Task) TaskResponse {
	return TaskResponse{
		ID:          task.ID,
		UserID:      task.UserID,
		Title:       task.Title,
		Description: task.Description,
		Priority:    task.Priority,
		DueDate:     task.DueDate,
		DueTime:     task.DueTime,
		CompletedAt: task.CompletedAt,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		Version:     task.Version,
	}
}
