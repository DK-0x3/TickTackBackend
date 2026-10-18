package tasks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"uuid"
)

type repositoryStub struct {
	createCalls        int
	createParams       CreateParams
	createResult       Task
	createError        error
	listUserID         uuid.UUID
	listParams         ListParams
	listResult         []Task
	listError          error
	countUserID        uuid.UUID
	countResult        TaskCounts
	countError         error
	getUserID          uuid.UUID
	getTaskID          uuid.UUID
	getResult          Task
	getError           error
	setCompletedUserID uuid.UUID
	setCompletedTaskID uuid.UUID
	setCompletedValue  bool
	setCompletedResult Task
	setCompletedError  error
}

func (stub *repositoryStub) Create(_ context.Context, params CreateParams) (Task, error) {
	stub.createCalls++
	stub.createParams = params
	return stub.createResult, stub.createError
}

func (stub *repositoryStub) List(_ context.Context, params ListParams) ([]Task, error) {
	stub.listUserID = params.UserID
	stub.listParams = params
	return stub.listResult, stub.listError
}

func (stub *repositoryStub) Count(_ context.Context, userID uuid.UUID) (TaskCounts, error) {
	stub.countUserID = userID
	return stub.countResult, stub.countError
}

func (stub *repositoryStub) Get(
	_ context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
) (Task, error) {
	stub.getUserID = userID
	stub.getTaskID = taskID
	return stub.getResult, stub.getError
}

func (stub *repositoryStub) SetCompleted(
	_ context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	completed bool,
) (Task, error) {
	stub.setCompletedUserID = userID
	stub.setCompletedTaskID = taskID
	stub.setCompletedValue = completed
	return stub.setCompletedResult, stub.setCompletedError
}

func (stub *repositoryStub) SetSchedule(context.Context, uuid.UUID, uuid.UUID, TaskSchedule) (Task, error) {
	return Task{}, nil
}

func (stub *repositoryStub) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (stub *repositoryStub) Restore(context.Context, uuid.UUID, uuid.UUID) (Task, error) {
	return Task{}, nil
}

func TestServiceCreateNormalizesInput(t *testing.T) {
	userID := uuid.NewV7()
	description := "  Important task  "
	expectedTask := Task{ID: uuid.NewV7(), UserID: userID, Title: "Write tests"}
	repository := &repositoryStub{createResult: expectedTask}
	service := NewService(repository)

	task, err := service.Create(context.Background(), CreateParams{
		UserID:      userID,
		Title:       "  Write tests  ",
		Description: &description,
		Priority:    PriorityHigh,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if task != expectedTask {
		t.Fatalf("Create() task = %#v, want %#v", task, expectedTask)
	}
	if repository.createCalls != 1 {
		t.Fatalf("repository Create() calls = %d, want 1", repository.createCalls)
	}
	if repository.createParams.Title != "Write tests" {
		t.Errorf("title = %q, want %q", repository.createParams.Title, "Write tests")
	}
	if repository.createParams.Description == nil || *repository.createParams.Description != "Important task" {
		t.Errorf("description = %v, want %q", repository.createParams.Description, "Important task")
	}
	if repository.createParams.Priority != PriorityHigh {
		t.Errorf("priority = %q, want %q", repository.createParams.Priority, PriorityHigh)
	}
}

func TestServiceCreateConvertsBlankDescriptionToNil(t *testing.T) {
	description := "  \t\n  "
	repository := &repositoryStub{}
	service := NewService(repository)

	_, err := service.Create(context.Background(), CreateParams{
		Title:       "Task",
		Description: &description,
		Priority:    PriorityNone,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repository.createParams.Description != nil {
		t.Fatalf("description = %q, want nil", *repository.createParams.Description)
	}
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		params      CreateParams
		expectedErr error
	}{
		{
			name:        "blank title",
			params:      CreateParams{Title: "  \t\n  ", Priority: PriorityNone},
			expectedErr: ErrTitleRequired,
		},
		{
			name:        "title longer than unicode limit",
			params:      CreateParams{Title: strings.Repeat("任", MaxTitleLength+1), Priority: PriorityNone},
			expectedErr: ErrTitleTooLong,
		},
		{
			name: "description longer than unicode limit",
			params: CreateParams{
				Title:       "Task",
				Description: stringPointer(strings.Repeat("任", MaxDescriptionLength+1)),
				Priority:    PriorityNone,
			},
			expectedErr: ErrDescriptionTooLong,
		},
		{
			name:        "invalid priority",
			params:      CreateParams{Title: "Task", Priority: Priority("urgent")},
			expectedErr: ErrInvalidPriority,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &repositoryStub{}
			service := NewService(repository)

			_, err := service.Create(context.Background(), test.params)

			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("Create() error = %v, want %v", err, test.expectedErr)
			}
			if repository.createCalls != 0 {
				t.Fatalf("repository Create() calls = %d, want 0", repository.createCalls)
			}
		})
	}
}

func TestServiceDelegatesQueries(t *testing.T) {
	userID := uuid.NewV7()
	taskID := uuid.NewV7()
	expectedError := errors.New("repository failed")
	repository := &repositoryStub{
		listError:         expectedError,
		getError:          expectedError,
		setCompletedError: expectedError,
	}
	service := NewService(repository)

	if _, err := service.List(context.Background(), ListParams{UserID: userID, Limit: 20}); !errors.Is(err, expectedError) {
		t.Fatalf("List() error = %v, want %v", err, expectedError)
	}
	if repository.listUserID != userID {
		t.Errorf("List() user ID = %s, want %s", repository.listUserID, userID)
	}

	if _, err := service.Get(context.Background(), userID, taskID); !errors.Is(err, expectedError) {
		t.Fatalf("Get() error = %v, want %v", err, expectedError)
	}
	if repository.getUserID != userID || repository.getTaskID != taskID {
		t.Errorf("Get() IDs = (%s, %s), want (%s, %s)", repository.getUserID, repository.getTaskID, userID, taskID)
	}

	if _, err := service.SetCompleted(context.Background(), userID, taskID, true); !errors.Is(err, expectedError) {
		t.Fatalf("SetCompleted() error = %v, want %v", err, expectedError)
	}
	if repository.setCompletedUserID != userID || repository.setCompletedTaskID != taskID {
		t.Errorf(
			"SetCompleted() IDs = (%s, %s), want (%s, %s)",
			repository.setCompletedUserID,
			repository.setCompletedTaskID,
			userID,
			taskID,
		)
	}
	if !repository.setCompletedValue {
		t.Error("SetCompleted() completed = false, want true")
	}
}

func stringPointer(value string) *string {
	return &value
}
