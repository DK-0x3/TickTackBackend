package tasks

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"
)

type TaskRepository interface {
	Create(ctx context.Context, params CreateParams) (Task, error)
	List(ctx context.Context, params ListParams) ([]Task, error)
	Count(ctx context.Context, userID uuid.UUID) (TaskCounts, error)
	Get(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (Task, error)
	SetCompleted(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, completed bool) (Task, error)
	SetSchedule(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, schedule TaskSchedule) (Task, error)
	Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error
	Restore(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (Task, error)
}

type Service struct {
	repository TaskRepository
}

func NewService(repository TaskRepository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Create(ctx context.Context, params CreateParams) (Task, error) {
	trimmedTitle := strings.TrimSpace(params.Title)
	if trimmedTitle == "" {
		return Task{}, ErrTitleRequired
	}

	if utf8.RuneCountInString(trimmedTitle) > MaxTitleLength {
		return Task{}, ErrTitleTooLong
	}

	var trimmedDescription *string
	if params.Description != nil {
		value := strings.TrimSpace(*params.Description)
		if utf8.RuneCountInString(value) > MaxDescriptionLength {
			return Task{}, ErrDescriptionTooLong
		}
		if value != "" {
			trimmedDescription = &value
		}
	}

	if !params.Priority.IsValid() {
		return Task{}, ErrInvalidPriority
	}
	if err := validateSchedule(TaskSchedule{DueDate: params.DueDate, DueTime: params.DueTime}); err != nil {
		return Task{}, err
	}

	return service.repository.Create(ctx, CreateParams{
		UserID:      params.UserID,
		Title:       trimmedTitle,
		Description: trimmedDescription,
		Priority:    params.Priority,
		DueDate:     params.DueDate,
		DueTime:     params.DueTime,
	})
}

func validateSchedule(schedule TaskSchedule) error {
	if schedule.DueDate == nil {
		if schedule.DueTime != nil {
			return ErrDueDateRequired
		}
		return nil
	}

	date, err := time.Parse("2006-01-02", *schedule.DueDate)
	if err != nil || date.Year() < 1 || date.Format("2006-01-02") != *schedule.DueDate {
		return ErrInvalidDueDate
	}

	if schedule.DueTime != nil {
		value, err := time.Parse("15:04", *schedule.DueTime)
		if err != nil || value.Format("15:04") != *schedule.DueTime {
			return ErrInvalidDueTime
		}
	}

	return nil
}

func (service *Service) List(ctx context.Context, params ListParams) (ListPage, error) {
	limit := params.Limit + 1
	tasks, err := service.repository.List(ctx, ListParams{
		UserID: params.UserID,
		Cursor: params.Cursor,
		Limit:  limit,
	})
	if err != nil {
		return ListPage{}, err
	}

	var nextCursor *ListCursor
	if len(tasks) == limit {
		tasks = tasks[:len(tasks)-1]

		lastTask := tasks[len(tasks)-1]
		value := ListCursor{
			ID:        lastTask.ID,
			CreatedAt: lastTask.CreatedAt,
		}
		nextCursor = &value
	}

	counts, err := service.repository.Count(ctx, params.UserID)
	if err != nil {
		return ListPage{}, err
	}

	return ListPage{
		Items:          tasks,
		NextCursor:     nextCursor,
		Total:          counts.Total,
		CompletedTotal: counts.CompletedTotal,
	}, nil
}

func (service *Service) Get(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
) (Task, error) {
	return service.repository.Get(ctx, userID, taskID)
}

func (service *Service) SetCompleted(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	completed bool,
) (Task, error) {
	return service.repository.SetCompleted(ctx, userID, taskID, completed)
}

func (service *Service) SetSchedule(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	schedule TaskSchedule,
) (Task, error) {
	if err := validateSchedule(schedule); err != nil {
		return Task{}, err
	}

	return service.repository.SetSchedule(ctx, userID, taskID, schedule)
}

func (service *Service) Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	return service.repository.Delete(ctx, userID, taskID)
}

func (service *Service) Restore(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (Task, error) {
	return service.repository.Restore(ctx, userID, taskID)
}
