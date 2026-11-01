package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbsqlc "github.com/DK-0x3/TickTackBackend/internal/platform/postgres/sqlc"
)

type Repository struct {
	queries dbsqlc.Querier
}

func NewRepository(queries dbsqlc.Querier) *Repository {
	return &Repository{queries: queries}
}

func (repository *Repository) Create(ctx context.Context, params CreateParams) (Task, error) {
	dueDate, dueTime, err := scheduleToDatabase(TaskSchedule{DueDate: params.DueDate, DueTime: params.DueTime})
	if err != nil {
		return Task{}, err
	}

	task, err := repository.queries.CreateTask(ctx, dbsqlc.CreateTaskParams{
		UserID:      params.UserID,
		Title:       params.Title,
		Description: params.Description,
		Priority:    string(params.Priority),
		DueDate:     dueDate,
		DueTime:     dueTime,
	})
	if err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}

	return taskFromDatabase(task), nil
}

func (repository *Repository) Get(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
) (Task, error) {
	task, err := repository.queries.GetTask(ctx, dbsqlc.GetTaskParams{
		UserID: userID,
		ID:     taskID,
	})
	if err != nil {
		return Task{}, repositoryError("get task", err)
	}

	return taskFromDatabase(task), nil
}

func (repository *Repository) SetCompleted(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	completed bool,
) (Task, error) {
	task, err := repository.queries.SetTaskCompleted(ctx, dbsqlc.SetTaskCompletedParams{
		Completed: completed,
		UserID:    userID,
		ID:        taskID,
	})
	if err != nil {
		return Task{}, repositoryError("set task completion", err)
	}

	return taskFromDatabase(task), nil
}

func (repository *Repository) SetSchedule(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	schedule TaskSchedule,
) (Task, error) {
	dueDate, dueTime, err := scheduleToDatabase(schedule)
	if err != nil {
		return Task{}, err
	}

	task, err := repository.queries.SetTaskSchedule(ctx, dbsqlc.SetTaskScheduleParams{
		DueDate: dueDate,
		DueTime: dueTime,
		UserID:  userID,
		ID:      taskID,
	})
	if err != nil {
		return Task{}, repositoryError("set task schedule", err)
	}

	return taskFromDatabase(task), nil
}

func (repository *Repository) Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	rows, err := repository.queries.DeleteTask(ctx, dbsqlc.DeleteTaskParams{
		UserID: userID,
		ID:     taskID,
	})
	if err != nil {
		return repositoryError("delete task", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (repository *Repository) Restore(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (Task, error) {
	task, err := repository.queries.RestoreTask(ctx, dbsqlc.RestoreTaskParams{
		UserID: userID,
		ID:     taskID,
	})
	if err != nil {
		return Task{}, repositoryError("restore task", err)
	}

	return taskFromDatabase(task), nil
}

func repositoryError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return fmt.Errorf("%s: %w", operation, err)
}

func taskFromDatabase(task dbsqlc.Task) Task {
	return Task{
		ID:          task.ID,
		UserID:      task.UserID,
		ListID:      task.ListID,
		Title:       task.Title,
		Description: task.Description,
		Priority:    Priority(task.Priority),
		DueDate:     nullableDate(task.DueDate),
		DueTime:     nullableClock(task.DueTime),
		CompletedAt: nullableTime(task.CompletedAt.Time, task.CompletedAt.Valid),
		CreatedAt:   task.CreatedAt.Time,
		UpdatedAt:   task.UpdatedAt.Time,
		DeletedAt:   nullableTime(task.DeletedAt.Time, task.DeletedAt.Valid),
		Version:     task.Version,
	}
}

func TaskFromViewRow(row dbsqlc.ListTasksForViewRow) Task {
	return taskFromDatabase(dbsqlc.Task{
		ID:          row.ID,
		UserID:      row.UserID,
		ListID:      row.ListID,
		Title:       row.Title,
		Description: row.Description,
		Priority:    row.Priority,
		DueDate:     row.DueDate,
		DueTime:     row.DueTime,
		CompletedAt: row.CompletedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		DeletedAt:   row.DeletedAt,
		Version:     row.Version,
	})
}

func scheduleToDatabase(schedule TaskSchedule) (pgtype.Date, pgtype.Time, error) {
	if err := validateSchedule(schedule); err != nil {
		return pgtype.Date{}, pgtype.Time{}, err
	}

	var dueDate pgtype.Date
	if schedule.DueDate != nil {
		value, err := time.Parse("2006-01-02", *schedule.DueDate)
		if err != nil {
			return pgtype.Date{}, pgtype.Time{}, ErrInvalidDueDate
		}
		dueDate = pgtype.Date{Time: value, Valid: true}
	}

	var dueTime pgtype.Time
	if schedule.DueTime != nil {
		value, err := time.Parse("15:04", *schedule.DueTime)
		if err != nil {
			return pgtype.Date{}, pgtype.Time{}, ErrInvalidDueTime
		}
		dueTime = pgtype.Time{Microseconds: int64(value.Hour()*60+value.Minute()) * 60_000_000, Valid: true}
	}

	return dueDate, dueTime, nil
}

func nullableDate(value pgtype.Date) *string {
	if !value.Valid {
		return nil
	}

	date := value.Time.Format("2006-01-02")
	return &date
}

func nullableClock(value pgtype.Time) *string {
	if !value.Valid {
		return nil
	}

	clock := fmt.Sprintf("%02d:%02d", value.Microseconds/3_600_000_000, (value.Microseconds/60_000_000)%60)
	return &clock
}

func nullableTime(value time.Time, valid bool) *time.Time {
	if !valid {
		return nil
	}

	return &value
}

func (repository *Repository) SetTitle(ctx context.Context, params RenameParams) (Task, error) {
	row, err := repository.queries.SetTaskTitle(ctx, dbsqlc.SetTaskTitleParams{
		UserID: params.UserID, ID: params.TaskID, Title: params.Title, ExpectedVersion: params.ExpectedVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		current, readErr := repository.Get(ctx, params.UserID, params.TaskID)
		if readErr != nil {
			return Task{}, readErr
		}
		return Task{}, &VersionConflictError{Current: current}
	}
	if err != nil {
		return Task{}, repositoryError("set task title", err)
	}
	return taskFromDatabase(row), nil
}
