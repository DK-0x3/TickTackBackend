package taskviews

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/DK-0x3/TickTackBackend/internal/modules/tasks"
)

type ViewRepository interface {
	InboxViews(context.Context, uuid.UUID) (uuid.UUID, []View, error)
	Get(ctx context.Context, userID uuid.UUID, viewID uuid.UUID) (View, error)
	ListTasks(context.Context, TaskQuery) ([]TaskRow, error)
	Count(
		ctx context.Context,
		view View,
		today time.Time,
		dayStart time.Time,
		dayEnd time.Time,
	) (Counts, error)
	SaveSettings(context.Context, UpdateSettingsParams) (View, error)
}

type Service struct {
	repository ViewRepository
	location   *time.Location
	clock      func() time.Time
}

func NewService(repository ViewRepository, location *time.Location, clock func() time.Time) *Service {
	return &Service{repository: repository, location: location, clock: clock}
}

func (service *Service) InboxViews(ctx context.Context, userID uuid.UUID) (uuid.UUID, []View, error) {
	return service.repository.InboxViews(ctx, userID)
}

func (service *Service) ListTasks(ctx context.Context, params ListTasksParams) (TaskPage, error) {
	if params.Limit < 1 || params.Limit > 100 {
		return TaskPage{}, ErrInvalidLimit
	}

	view, err := service.repository.Get(ctx, params.UserID, params.ViewID)
	if err != nil {
		return TaskPage{}, err
	}

	now := service.clock().In(service.location)
	year, month, day := now.Date()
	dayStart := time.Date(year, month, day, 0, 0, 0, 0, service.location)
	dayEnd := dayStart.AddDate(0, 0, 1)
	today := now.Format("2006-01-02")
	date := time.Date(
		year, month, day,
		0, 0, 0, 0,
		time.UTC,
	)

	var cursor *PageCursor
	if params.Cursor != "" {
		cursor, err = decodeCursor(params.Cursor)
		if err != nil {
			return TaskPage{}, err
		}
		if cursor.ViewID != view.ID || cursor.Version != view.Version || cursor.Today != today {
			return TaskPage{}, ErrStaleCursor
		}
	}

	rows, err := service.repository.ListTasks(ctx, TaskQuery{
		UserID:   params.UserID,
		ViewID:   params.ViewID,
		Today:    date,
		Cursor:   cursor,
		Limit:    params.Limit + 1,
		DayStart: dayStart,
		DayEnd:   dayEnd,
	})
	if err != nil {
		return TaskPage{}, err
	}

	page := TaskPage{
		Items:       make([]tasks.Task, 0, min(len(rows), params.Limit)),
		ViewVersion: view.Version,
		Today:       today,
	}
	if len(rows) > params.Limit {
		rows = rows[:params.Limit]
		last := rows[len(rows)-1]
		value, err := encodeCursor(PageCursor{
			ViewID:  view.ID,
			Version: view.Version,
			Today:   today,
			SortKey: last.SortKey,
			ID:      last.Task.ID,
		})
		if err != nil {
			return TaskPage{}, err
		}
		page.NextCursor = &value
	}
	for _, row := range rows {
		page.Items = append(page.Items, row.Task)
	}

	counts, err := service.repository.Count(ctx, view, date, dayStart, dayEnd)
	if err != nil {
		return TaskPage{}, err
	}
	page.Total = counts.Total
	page.CompletedTotal = counts.CompletedTotal
	page.VisibleTotal = counts.VisibleTotal
	page.DayEnd = dayEnd.Format(time.RFC3339)

	current, err := service.repository.Get(ctx, params.UserID, params.ViewID)
	if err != nil {
		return TaskPage{}, err
	}
	if current.Version != view.Version {
		return TaskPage{}, ErrStaleCursor
	}

	return page, nil
}

func (service *Service) UpdateViewSettings(ctx context.Context, params UpdateSettingsParams) (View, error) {
	if !params.SortField.IsValid() {
		return View{}, ErrInvalidSortField
	}
	if !params.SortDirection.IsValid() {
		return View{}, ErrInvalidSortDirection
	}
	if params.ExpectedVersion < 1 {
		return View{}, ErrInvalidExpectedVersion
	}

	view, err := service.repository.Get(ctx, params.UserID, params.ViewID)
	if err != nil {
		return View{}, err
	}
	if view.Version != params.ExpectedVersion {
		return View{}, &VersionConflictError{view}
	}

	savedView, err := service.repository.SaveSettings(ctx, params)
	if err != nil {
		if errors.Is(err, ErrVersionConflict) {
			view, err := service.repository.Get(ctx, params.UserID, params.ViewID)
			if err != nil {
				return View{}, err
			}
			return View{}, &VersionConflictError{view}
		}
		return View{}, err
	}
	return savedView, nil
}

func encodeCursor(cursor PageCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal task view cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(value string) (*PageCursor, error) {
	if len(value) > 4096 {
		return nil, ErrInvalidCursor
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var cursor PageCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.ViewID == uuid.Nil() || cursor.ID == uuid.Nil() || cursor.Version < 1 {
		return nil, ErrInvalidCursor
	}
	if _, err := time.Parse("2006-01-02", cursor.Today); err != nil {
		return nil, ErrInvalidCursor
	}
	return &cursor, nil
}
