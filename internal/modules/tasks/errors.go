package tasks

import "errors"

var (
	ErrNotFound           = errors.New("task not found")
	ErrTitleRequired      = errors.New("task title is required")
	ErrTitleTooLong       = errors.New("task title is too long")
	ErrDescriptionTooLong = errors.New("task description is too long")
	ErrInvalidPriority    = errors.New("task priority is invalid")
	ErrInvalidDueDate     = errors.New("task due date is invalid")
	ErrInvalidDueTime     = errors.New("task due time is invalid")
	ErrDueDateRequired    = errors.New("task due date is required when due time is set")
	ErrInvalidExpectedVersion = errors.New("invalid expected version")
	ErrVersionConflict = errors.New("task version conflict")
)

type VersionConflictError struct{ Current Task }

func (err *VersionConflictError) Error() string { return ErrVersionConflict.Error() }
func (err *VersionConflictError) Unwrap() error { return ErrVersionConflict }
