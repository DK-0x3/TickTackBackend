package tasks

import (
	"time"
	"uuid"
)

type createTaskRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Priority    string  `json:"priority"`
}

type setTaskCompletedRequest struct {
	Completed *bool `json:"completed"`
}

type taskResponse struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Priority    Priority   `json:"priority"`
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
	Items          []taskResponse `json:"items"`
	NextCursor     *string        `json:"next_cursor"`
	Total          int64          `json:"total"`
	CompletedTotal int64          `json:"completed_total"`
}

func newTaskResponse(task Task) taskResponse {
	return taskResponse{
		ID:          task.ID,
		UserID:      task.UserID,
		Title:       task.Title,
		Description: task.Description,
		Priority:    task.Priority,
		CompletedAt: task.CompletedAt,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		Version:     task.Version,
	}
}

func newTaskListResponse(page ListPage) taskListResponse {
	response := taskListResponse{
		Items:          make([]taskResponse, 0, len(page.Items)),
		Total:          page.Total,
		CompletedTotal: page.CompletedTotal,
	}
	for _, task := range page.Items {
		response.Items = append(response.Items, newTaskResponse(task))
	}

	if page.NextCursor != nil {
		encodedCursor := encodeTaskCursor(*page.NextCursor)
		response.NextCursor = &encodedCursor
	}

	return response
}
