package taskviews

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/DK-0x3/TickTackBackend/internal/modules/tasks"
	dbsqlc "github.com/DK-0x3/TickTackBackend/internal/platform/postgres/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	queries dbsqlc.Querier
}

func NewRepository(queries dbsqlc.Querier) *Repository {
	return &Repository{queries: queries}
}

func (repository *Repository) InboxViews(ctx context.Context, userID uuid.UUID) (uuid.UUID, []View, error) {
	inbox, err := repository.queries.GetInbox(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		inbox, err = repository.queries.EnsureInbox(ctx, userID)
	}
	if err != nil {
		return uuid.Nil(), nil, fmt.Errorf("get Inbox: %w", err)
	}

	rows, err := repository.queries.ListTaskViews(ctx, dbsqlc.ListTaskViewsParams{
		UserID: userID,
		ListID: inbox.ID,
	})
	if err != nil {
		return uuid.Nil(), nil, fmt.Errorf("list Inbox views: %w", err)
	}
	if len(rows) < 6 {
		if err := repository.queries.EnsureInboxViews(ctx, dbsqlc.EnsureInboxViewsParams{
			UserID: userID,
			ListID: inbox.ID,
		}); err != nil {
			return uuid.Nil(), nil, fmt.Errorf("ensure Inbox views: %w", err)
		}
		rows, err = repository.queries.ListTaskViews(ctx, dbsqlc.ListTaskViewsParams{
			UserID: userID,
			ListID: inbox.ID,
		})
		if err != nil {
			return uuid.Nil(), nil, fmt.Errorf("list Inbox views: %w", err)
		}
	}

	views := make([]View, 0, len(rows))
	for _, row := range rows {
		views = append(views, viewFromDatabase(row))
	}
	return inbox.ID, views, nil
}

func (repository *Repository) Get(ctx context.Context, userID, viewID uuid.UUID) (View, error) {
	row, err := repository.queries.GetTaskView(ctx, dbsqlc.GetTaskViewParams{
		UserID: userID,
		ViewID: viewID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrNotFound
	}
	if err != nil {
		return View{}, fmt.Errorf("get task view: %w", err)
	}
	return viewFromDatabase(row), nil
}

func (repository *Repository) ListTasks(ctx context.Context, params TaskQuery) ([]TaskRow, error) {
	var cursorSortKey *string
	var cursorID pgtype.UUID
	if params.Cursor != nil {
		cursorSortKey = &params.Cursor.SortKey
		cursorID = pgtype.UUID{Bytes: params.Cursor.ID, Valid: true}
	}

	rows, err := repository.queries.ListTasksForView(ctx, dbsqlc.ListTasksForViewParams{
		UserID:        params.UserID,
		ViewID:        params.ViewID,
		Today:         pgtype.Date{Time: params.Today, Valid: true},
		DayStart:      pgtype.Timestamptz{Time: params.DayStart, Valid: true},
		DayEnd:        pgtype.Timestamptz{Time: params.DayEnd, Valid: true},
		CursorSortKey: cursorSortKey,
		CursorID:      cursorID,
		CursorLimit:   int32(params.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list tasks for view: %w", err)
	}

	result := make([]TaskRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, TaskRow{Task: tasks.TaskFromViewRow(row), SortKey: row.SortKey})
	}
	return result, nil
}

func (repository *Repository) Count(ctx context.Context, view View, today time.Time, dayStart time.Time, dayEnd time.Time) (Counts, error) {
	row, err := repository.queries.CountTasksForView(ctx, dbsqlc.CountTasksForViewParams{
		UserID:   view.UserID,
		ViewID:   view.ID,
		Today:    pgtype.Date{Time: today, Valid: true},
		DayStart: pgtype.Timestamptz{Time: dayStart, Valid: true},
		DayEnd:   pgtype.Timestamptz{Time: dayEnd, Valid: true},
	})
	if err != nil {
		return Counts{}, fmt.Errorf("count tasks for view: %w", err)
	}
	return Counts{Total: row.Total, CompletedTotal: row.CompletedTotal, VisibleTotal: row.VisibleTotal}, nil
}

func (repository *Repository) SaveSettings(ctx context.Context, params UpdateSettingsParams) (View, error) {
	row, err := repository.queries.SaveTaskViewSettings(ctx, dbsqlc.SaveTaskViewSettingsParams{
		UserID:          params.UserID,
		ViewID:          params.ViewID,
		SortField:       string(params.SortField),
		SortDirection:   string(params.SortDirection),
		ShowCompleted:   params.ShowCompleted,
		ExpectedVersion: params.ExpectedVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return View{}, ErrVersionConflict
	}
	if err != nil {
		return View{}, fmt.Errorf("save task view settings: %w", err)
	}
	return viewFromDatabase(dbsqlc.TaskView(row)), nil
}

func viewFromDatabase(row dbsqlc.TaskView) View {
	return View{
		ID:            row.ID,
		UserID:        row.UserID,
		ListID:        row.ListID,
		Section:       row.Section,
		Position:      row.Position,
		SortField:     SortField(row.SortField),
		SortDirection: SortDirection(row.SortDirection),
		ShowCompleted: row.ShowCompleted,
		Version:       row.Version,
	}
}
