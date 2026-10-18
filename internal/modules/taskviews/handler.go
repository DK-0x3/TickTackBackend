package taskviews

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/DK-0x3/TickTackBackend/internal/modules/tasks"
	"github.com/DK-0x3/TickTackBackend/internal/platform/httpquery"
	"github.com/gin-gonic/gin"
)

type ViewService interface {
	InboxViews(ctx context.Context, userID uuid.UUID) (uuid.UUID, []View, error)
	ListTasks(ctx context.Context, params ListTasksParams) (TaskPage, error)
	UpdateViewSettings(ctx context.Context, params UpdateSettingsParams) (View, error)
}

type Handler struct {
	service ViewService
	userID  uuid.UUID
}

func RegisterRoutes(router gin.IRoutes, service ViewService, userID uuid.UUID) {
	handler := Handler{service: service, userID: userID}
	router.GET("/inbox/views", handler.inboxViews)
	router.GET("/task-views/:viewID/tasks", handler.listTasks)
	router.PATCH("/task-views/:viewID/settings", handler.updateSettings)
}

func (handler Handler) inboxViews(ctx *gin.Context) {
	listID, views, err := handler.service.InboxViews(ctx.Request.Context(), handler.userID)
	if err != nil {
		respondViewError(ctx, err)
		return
	}
	result := make([]viewResponse, 0, len(views))
	for _, view := range views {
		result = append(result, newViewResponse(view))
	}
	ctx.JSON(http.StatusOK, gin.H{"list_id": listID, "views": result})
}

func (handler Handler) listTasks(ctx *gin.Context) {
	viewID, ok := parseViewID(ctx)
	if !ok {
		return
	}
	pagination, err := httpquery.ParsePagination(ctx.Request.URL.Query(), 20, 100)
	if err != nil {
		respondViewError(ctx, ErrInvalidLimit)
		return
	}
	page, err := handler.service.ListTasks(ctx.Request.Context(), ListTasksParams{
		UserID: handler.userID,
		ViewID: viewID,
		Limit:  pagination.Limit,
		Cursor: pagination.Cursor,
	})
	if err != nil {
		respondViewError(ctx, err)
		return
	}
	items := make([]tasks.TaskResponse, 0, len(page.Items))
	for _, task := range page.Items {
		items = append(items, tasks.NewTaskResponse(task))
	}
	ctx.JSON(http.StatusOK, gin.H{
		"items":           items,
		"next_cursor":     page.NextCursor,
		"total":           page.Total,
		"completed_total": page.CompletedTotal,
		"visible_total":   page.VisibleTotal,
		"view_version":    page.ViewVersion,
		"today":           page.Today,
		"next_refresh_at": page.DayEnd,
	})
}

func (handler Handler) updateSettings(ctx *gin.Context) {
	viewID, ok := parseViewID(ctx)
	if !ok {
		return
	}
	var request struct {
		SortField       SortField     `json:"sort_field"`
		SortDirection   SortDirection `json:"sort_direction"`
		ShowCompleted   *bool         `json:"show_completed"`
		ExpectedVersion *int64        `json:"expected_version"`
	}
	err := ctx.ShouldBindJSON(&request)
	if err != nil {
		respondViewError(ctx, ErrInvalidRequest)
		return
	} else if request.ExpectedVersion == nil {
		respondViewError(ctx, ErrInvalidExpectedVersion)
		return
	} else if request.ShowCompleted == nil {
		respondViewError(ctx, ErrInvalidShowCompleted)
		return
	}

	view, err := handler.service.UpdateViewSettings(ctx.Request.Context(), UpdateSettingsParams{
		UserID:          handler.userID,
		ViewID:          viewID,
		SortField:       request.SortField,
		SortDirection:   request.SortDirection,
		ShowCompleted:   *request.ShowCompleted,
		ExpectedVersion: *request.ExpectedVersion,
	})
	if err != nil {
		respondViewError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newViewResponse(view))
}

func parseViewID(ctx *gin.Context) (uuid.UUID, bool) {
	viewID, err := uuid.Parse(ctx.Param("viewID"))
	if err != nil || viewID == uuid.Nil() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_view_id", "message": "View ID is invalid"}})
		return uuid.Nil(), false
	}
	return viewID, true
}

func respondViewError(ctx *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := "Internal server error"

	if conflict, ok := errors.AsType[*VersionConflictError](err); ok {
		current := conflict.Current
		ctx.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "view_conflict", "message": err.Error()}, "current_view": newViewResponse(current)})
		return
	}

	switch {
	case errors.Is(err, ErrNotFound):
		status, code, message = http.StatusNotFound, "view_not_found", ErrNotFound.Error()
	case errors.Is(err, ErrInvalidSortField), errors.Is(err, ErrInvalidSortDirection),
		errors.Is(err, ErrInvalidExpectedVersion), errors.Is(err, ErrInvalidCursor),
		errors.Is(err, ErrInvalidLimit), errors.Is(err, ErrInvalidShowCompleted), errors.Is(err, ErrInvalidRequest):
		status, code, message = http.StatusBadRequest, "validation_failed", err.Error()
	case errors.Is(err, ErrStaleCursor):
		status, code, message = http.StatusConflict, "view_conflict", err.Error()
	}
	ctx.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

type viewResponse struct {
	ID            uuid.UUID     `json:"id"`
	ListID        uuid.UUID     `json:"list_id"`
	Section       string        `json:"section"`
	Position      int32         `json:"position"`
	SortField     SortField     `json:"sort_field"`
	SortDirection SortDirection `json:"sort_direction"`
	ShowCompleted bool          `json:"show_completed"`
	Version       int64         `json:"version"`
}

func newViewResponse(view View) viewResponse {
	return viewResponse{
		ID: view.ID, ListID: view.ListID, Section: view.Section, Position: view.Position,
		SortField: view.SortField, SortDirection: view.SortDirection, ShowCompleted: view.ShowCompleted,
		Version: view.Version,
	}
}
