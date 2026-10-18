package taskviews

import (
	"errors"
	"time"
	"uuid"

	"github.com/DK-0x3/TickTackBackend/internal/modules/tasks"
)

var (
	ErrNotFound               = errors.New("task view not found")
	ErrInvalidSortField       = errors.New("invalid sort field")
	ErrInvalidSortDirection   = errors.New("invalid sort direction")
	ErrInvalidExpectedVersion = errors.New("invalid expected version")
	ErrInvalidRequest         = errors.New("invalid request")
	ErrInvalidShowCompleted   = errors.New("invalid show competed")
	ErrVersionConflict        = errors.New("task view version conflict")
	ErrInvalidCursor          = errors.New("invalid task view cursor")
	ErrStaleCursor            = errors.New("task view cursor is stale")
	ErrInvalidLimit           = errors.New("invalid page limit")
)

type VersionConflictError struct {
	Current View
}

func (err *VersionConflictError) Error() string {
	return ErrVersionConflict.Error()
}

func (err *VersionConflictError) Unwrap() error {
	return ErrVersionConflict
}

type SortField string

const (
	SortTitle     SortField = "title"
	SortDueAt     SortField = "due_at"
	SortCreatedAt SortField = "created_at"
	SortUpdatedAt SortField = "updated_at"
	SortPriority  SortField = "priority"
)

func (field SortField) IsValid() bool {
	switch field {
	case SortTitle, SortDueAt, SortCreatedAt, SortUpdatedAt, SortPriority:
		return true
	default:
		return false
	}
}

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

func (direction SortDirection) IsValid() bool {
	return direction == SortAscending || direction == SortDescending
}

type View struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	ListID        uuid.UUID
	Section       string
	Position      int32
	SortField     SortField
	SortDirection SortDirection
	ShowCompleted bool
	Version       int64
}

type UpdateSettingsParams struct {
	UserID          uuid.UUID
	ViewID          uuid.UUID
	SortField       SortField
	SortDirection   SortDirection
	ShowCompleted   bool
	ExpectedVersion int64
}

type PageCursor struct {
	ViewID  uuid.UUID `json:"view_id"`
	Version int64     `json:"version"`
	Today   string    `json:"today"`
	SortKey string    `json:"sort_key"`
	ID      uuid.UUID `json:"id"`
}

type ListTasksParams struct {
	UserID uuid.UUID
	ViewID uuid.UUID
	Limit  int
	Cursor string
}

type TaskRow struct {
	Task    tasks.Task
	SortKey string
}

type TaskPage struct {
	Items          []tasks.Task
	NextCursor     *string
	Total          int64
	CompletedTotal int64
	VisibleTotal   int64
	ViewVersion    int64
	Today          string
	DayEnd         string
}

type Counts struct {
	Total          int64
	CompletedTotal int64
	VisibleTotal   int64
}

type TaskQuery struct {
	UserID   uuid.UUID
	ViewID   uuid.UUID
	Today    time.Time
	Cursor   *PageCursor
	Limit    int
	DayStart time.Time
	DayEnd   time.Time
}
