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

type taskListResponse struct {
	Items          []TaskResponse `json:"items"`
	NextCursor     *string        `json:"next_cursor"`
	Total          int64          `json:"total"`
	CompletedTotal int64          `json:"completed_total"`
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

func newTaskListResponse(page ListPage) taskListResponse {
	response := taskListResponse{
		Items:          make([]TaskResponse, 0, len(page.Items)),
		Total:          page.Total,
		CompletedTotal: page.CompletedTotal,
	}
	for _, task := range page.Items {
		response.Items = append(response.Items, NewTaskResponse(task))
	}

	if page.NextCursor != nil {
		encodedCursor := encodeTaskCursor(*page.NextCursor)
		response.NextCursor = &encodedCursor
	}

	return response
}
