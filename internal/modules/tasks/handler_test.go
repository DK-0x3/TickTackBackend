package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
)

type serviceStub struct {
	createParams       CreateParams
	createResult       Task
	createError        error
	listParams         ListParams
	listResult         ListPage
	listError          error
	getCalls           int
	setCompletedTaskID uuid.UUID
	setCompletedValue  bool
}

func (stub *serviceStub) Create(_ context.Context, params CreateParams) (Task, error) {
	stub.createParams = params
	return stub.createResult, stub.createError
}

func (stub *serviceStub) List(_ context.Context, params ListParams) (ListPage, error) {
	stub.listParams = params
	return stub.listResult, stub.listError
}

func (stub *serviceStub) Get(context.Context, uuid.UUID, uuid.UUID) (Task, error) {
	stub.getCalls++
	return Task{}, nil
}

func (stub *serviceStub) SetCompleted(
	_ context.Context,
	_ uuid.UUID,
	taskID uuid.UUID,
	completed bool,
) (Task, error) {
	stub.setCompletedTaskID = taskID
	stub.setCompletedValue = completed
	return Task{ID: taskID}, nil
}

func (stub *serviceStub) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (stub *serviceStub) Restore(context.Context, uuid.UUID, uuid.UUID) (Task, error) {
	return Task{}, nil
}

func TestHandlerCreateUsesDevelopmentUserAndDefaultPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.NewV7()
	taskID := uuid.NewV7()
	service := &serviceStub{createResult: Task{ID: taskID, UserID: userID, Priority: PriorityNone}}
	router := gin.New()
	RegisterRoutes(router, service, userID)

	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"Task"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if service.createParams.UserID != userID {
		t.Errorf("user ID = %s, want %s", service.createParams.UserID, userID)
	}
	if service.createParams.Priority != PriorityNone {
		t.Errorf("priority = %q, want %q", service.createParams.Priority, PriorityNone)
	}
}

func TestHandlerCreateMapsValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serviceStub{createError: ErrTitleRequired}
	router := gin.New()
	RegisterRoutes(router, service, uuid.NewV7())

	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":""}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if !strings.Contains(recorder.Body.String(), `"code":"validation_failed"`) {
		t.Fatalf("body = %s, want validation_failed error", recorder.Body.String())
	}
}

func TestHandlerCreateRejectsMalformedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serviceStub{}
	router := gin.New()
	RegisterRoutes(router, service, uuid.NewV7())

	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlerGetRejectsInvalidTaskID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serviceStub{}
	router := gin.New()
	RegisterRoutes(router, service, uuid.NewV7())

	request := httptest.NewRequest(http.MethodGet, "/tasks/invalid", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if service.getCalls != 0 {
		t.Fatalf("service Get() calls = %d, want 0", service.getCalls)
	}
}

func TestHandlerSetCompletedAcceptsFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	taskID := uuid.NewV7()
	service := &serviceStub{}
	router := gin.New()
	RegisterRoutes(router, service, uuid.NewV7())

	request := httptest.NewRequest(
		http.MethodPatch,
		"/tasks/"+taskID.String()+"/completion",
		strings.NewReader(`{"completed":false}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if service.setCompletedTaskID != taskID {
		t.Errorf("task ID = %s, want %s", service.setCompletedTaskID, taskID)
	}
	if service.setCompletedValue {
		t.Error("completed = true, want false")
	}
}

func TestRespondTaskErrorMapsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &serviceStub{createError: errors.Join(ErrNotFound, errors.New("database detail"))}
	router := gin.New()
	RegisterRoutes(router, service, uuid.NewV7())

	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"Task"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestHandlerListReturnsPageAndTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.NewV7()
	createdAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	task := Task{ID: uuid.NewV7(), UserID: userID, Title: "Task", Priority: PriorityNone, CreatedAt: createdAt}
	nextCursor := &ListCursor{ID: task.ID, CreatedAt: createdAt}
	previousCursor := ListCursor{ID: uuid.NewV7(), CreatedAt: createdAt.Add(time.Hour)}
	service := &serviceStub{listResult: ListPage{Items: []Task{task}, NextCursor: nextCursor, Total: 5, CompletedTotal: 2}}
	router := gin.New()
	RegisterRoutes(router, service, userID)

	request := httptest.NewRequest(http.MethodGet, "/tasks?limit=1&cursor="+encodeTaskCursor(previousCursor), nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	if service.listParams.UserID != userID || service.listParams.Limit != 1 || service.listParams.Cursor == nil || service.listParams.Cursor.ID != previousCursor.ID {
		t.Fatalf("List() params = %#v", service.listParams)
	}
	var response taskListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != task.ID || response.Total != 5 || response.CompletedTotal != 2 {
		t.Fatalf("response = %#v, want one task, total 5 and completed total 2", response)
	}
	if response.NextCursor == nil || *response.NextCursor != encodeTaskCursor(*nextCursor) {
		t.Fatalf("next cursor = %v, want encoded cursor", response.NextCursor)
	}
}

func TestHandlerListRejectsInvalidPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, query := range []string{"limit=0", "limit=101", "limit=x", "limit=1&limit=2", "cursor=bad", "cursor="} {
		t.Run(query, func(t *testing.T) {
			service := &serviceStub{}
			router := gin.New()
			RegisterRoutes(router, service, uuid.NewV7())
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tasks?"+query, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s", recorder.Code, recorder.Body.String())
			}
			if service.listParams.UserID != uuid.Nil() {
				t.Fatal("List() must not be called for invalid pagination")
			}
		})
	}
}
