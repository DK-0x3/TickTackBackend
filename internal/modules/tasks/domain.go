package tasks

import (
	"time"
	"uuid"
)

const (
	MaxTitleLength       = 500
	MaxDescriptionLength = 10000
)

type Priority string

const (
	PriorityNone   Priority = "none"
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

func (priority Priority) IsValid() bool {
	switch priority {
	case PriorityNone, PriorityLow, PriorityMedium, PriorityHigh:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ListID      uuid.UUID
	Title       string
	Description *string
	Priority    Priority
	DueDate     *string
	DueTime     *string
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
	Version     int64
}

type CreateParams struct {
	UserID      uuid.UUID
	Title       string
	Description *string
	Priority    Priority
	DueDate     *string
	DueTime     *string
}

type TaskSchedule struct {
	DueDate *string
	DueTime *string
}

type ListParams struct {
	UserID uuid.UUID
	Cursor *ListCursor
	Limit  int
}

type ListCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type ListPage struct {
	Items          []Task
	NextCursor     *ListCursor
	Total          int64
	CompletedTotal int64
}

type TaskCounts struct {
	Total          int64
	CompletedTotal int64
}
