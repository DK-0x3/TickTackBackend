package tasks

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/DK-0x3/TickTackBackend/internal/platform/httpquery"
	"github.com/gin-gonic/gin"
)

const defaultLimit = 20
const maxPageSize = 100

type TaskService interface {
	Create(context.Context, CreateParams) (Task, error)
	List(context.Context, ListParams) (ListPage, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (Task, error)
	SetCompleted(context.Context, uuid.UUID, uuid.UUID, bool) (Task, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
	Restore(context.Context, uuid.UUID, uuid.UUID) (Task, error)
}

type Handler struct {
	service TaskService
	userID  uuid.UUID
}

func RegisterRoutes(router gin.IRoutes, service TaskService, userID uuid.UUID) {
	handler := Handler{service: service, userID: userID}

	router.POST("/tasks", handler.create)
	router.GET("/tasks", handler.list)
	router.GET("/tasks/:id", handler.get)
	router.PATCH("/tasks/:id/completion", handler.setCompleted)
	router.DELETE("/tasks/:id", handler.delete)
	router.POST("/tasks/:id/restore", handler.restore)
}

func (handler Handler) create(ctx *gin.Context) {
	var request createTaskRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}

	priority := Priority(request.Priority)
	if request.Priority == "" {
		priority = PriorityNone
	}

	task, err := handler.service.Create(ctx.Request.Context(), CreateParams{
		UserID:      handler.userID,
		Title:       request.Title,
		Description: request.Description,
		Priority:    priority,
	})
	if err != nil {
		respondTaskError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newTaskResponse(task))
}

func (handler Handler) list(ctx *gin.Context) {
	pagination, paginationErr := httpquery.ParsePagination(
		ctx.Request.URL.Query(),
		defaultLimit,
		maxPageSize,
	)
	if paginationErr != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_pagination", "Pagination parameters are invalid")
		return
	}

	var cursor *ListCursor
	if pagination.Cursor != "" {
		decodedCursor, err := decodeTaskCursor(pagination.Cursor)
		if err != nil {
			respondError(ctx, http.StatusBadRequest, "invalid_pagination", "Pagination parameters are invalid")
			return
		}
		cursor = &decodedCursor
	}

	page, err := handler.service.List(ctx.Request.Context(),
		ListParams{
			UserID: handler.userID,
			Cursor: cursor,
			Limit:  pagination.Limit,
		})
	if err != nil {
		respondTaskError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newTaskListResponse(page))
}

func (handler Handler) get(ctx *gin.Context) {
	taskID, ok := parseTaskID(ctx)
	if !ok {
		return
	}

	task, err := handler.service.Get(ctx.Request.Context(), handler.userID, taskID)
	if err != nil {
		respondTaskError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newTaskResponse(task))
}

func (handler Handler) setCompleted(ctx *gin.Context) {
	taskID, ok := parseTaskID(ctx)
	if !ok {
		return
	}

	var request setTaskCompletedRequest
	if err := ctx.ShouldBindJSON(&request); err != nil || request.Completed == nil {
		respondError(ctx, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}

	task, err := handler.service.SetCompleted(
		ctx.Request.Context(),
		handler.userID,
		taskID,
		*request.Completed,
	)
	if err != nil {
		respondTaskError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newTaskResponse(task))
}

func (handler Handler) delete(ctx *gin.Context) {
	taskID, ok := parseTaskID(ctx)
	if !ok {
		return
	}

	if err := handler.service.Delete(ctx.Request.Context(), handler.userID, taskID); err != nil {
		respondTaskError(ctx, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (handler Handler) restore(ctx *gin.Context) {
	taskID, ok := parseTaskID(ctx)
	if !ok {
		return
	}

	task, err := handler.service.Restore(ctx.Request.Context(), handler.userID, taskID)
	if err != nil {
		respondTaskError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newTaskResponse(task))
}

func parseTaskID(ctx *gin.Context) (uuid.UUID, bool) {
	taskID, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		respondError(ctx, http.StatusBadRequest, "invalid_task_id", "Task ID is invalid")
		return uuid.Nil(), false
	}

	return taskID, true
}

func respondTaskError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrTitleRequired),
		errors.Is(err, ErrTitleTooLong),
		errors.Is(err, ErrDescriptionTooLong),
		errors.Is(err, ErrInvalidPriority):
		respondError(ctx, http.StatusBadRequest, "validation_failed", err.Error())
	case errors.Is(err, ErrNotFound):
		respondError(ctx, http.StatusNotFound, "task_not_found", ErrNotFound.Error())
	default:
		respondError(ctx, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func respondError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, errorResponse{Error: errorBody{Code: code, Message: message}})
}
