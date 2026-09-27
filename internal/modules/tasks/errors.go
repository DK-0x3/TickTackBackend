package tasks

import "errors"

var (
	ErrNotFound           = errors.New("task not found")
	ErrTitleRequired      = errors.New("task title is required")
	ErrTitleTooLong       = errors.New("task title is too long")
	ErrDescriptionTooLong = errors.New("task description is too long")
	ErrInvalidPriority    = errors.New("task priority is invalid")
)
