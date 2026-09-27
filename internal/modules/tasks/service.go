package tasks

import (
	"context"
	"strings"
	"unicode/utf8"
	"uuid"
)

type TaskRepository interface {
	Create(ctx context.Context, params CreateParams) (Task, error)
	List(ctx context.Context, params ListParams) ([]Task, error)
	Count(ctx context.Context, userID uuid.UUID) (TaskCounts, error)
	Get(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (Task, error)
	SetCompleted(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, completed bool) (Task, error)
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

	return service.repository.Create(ctx, CreateParams{
		UserID:      params.UserID,
		Title:       trimmedTitle,
		Description: trimmedDescription,
		Priority:    params.Priority,
	})
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

func (service *Service) Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	return service.repository.Delete(ctx, userID, taskID)
}

func (service *Service) Restore(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (Task, error) {
	return service.repository.Restore(ctx, userID, taskID)
}
