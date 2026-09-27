package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type databaseStub struct {
	err error
}

func (stub databaseStub) Ping(context.Context) error {
	return stub.err
}

func TestCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterRoutes(router, databaseStub{})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	const expectedBody = `{"status":"ok"}`
	if recorder.Body.String() != expectedBody {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), expectedBody)
	}
}

func TestReady(t *testing.T) {
	tests := []struct {
		name         string
		database     Database
		expectedCode int
		expectedBody string
	}{
		{
			name:         "database is available",
			database:     databaseStub{},
			expectedCode: http.StatusOK,
			expectedBody: `{"status":"ready"}`,
		},
		{
			name:         "database is unavailable",
			database:     databaseStub{err: errors.New("connection failed")},
			expectedCode: http.StatusServiceUnavailable,
			expectedBody: `{"status":"unavailable"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterRoutes(router, test.database)

			request := httptest.NewRequest(http.MethodGet, "/ready", nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != test.expectedCode {
				t.Fatalf("status code = %d, want %d", recorder.Code, test.expectedCode)
			}

			if recorder.Body.String() != test.expectedBody {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), test.expectedBody)
			}
		})
	}
}
