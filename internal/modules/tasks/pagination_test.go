package tasks

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"
)

func TestServiceListPageBoundaries(t *testing.T) {
	userID := uuid.NewV7()
	createdAt := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	first := Task{ID: uuid.NewV7(), CreatedAt: createdAt}
	second := Task{ID: uuid.NewV7(), CreatedAt: createdAt.Add(-time.Minute)}
	lookahead := Task{ID: uuid.NewV7(), CreatedAt: createdAt.Add(-2 * time.Minute)}

	cases := []struct {
		name       string
		found      []Task
		wantItems  int
		wantCursor *Task
	}{
		{name: "empty", found: []Task{}, wantItems: 0},
		{name: "short page", found: []Task{first}, wantItems: 1},
		{name: "exact page", found: []Task{first, second}, wantItems: 2},
		{name: "more tasks", found: []Task{first, second, lookahead}, wantItems: 2, wantCursor: &second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			previousCursor := &ListCursor{ID: uuid.NewV7(), CreatedAt: createdAt.Add(time.Hour)}
			repository := &repositoryStub{listResult: tc.found, countResult: TaskCounts{Total: 42, CompletedTotal: 17}}
			page, err := NewService(repository).List(context.Background(), ListParams{
				UserID: userID,
				Cursor: previousCursor,
				Limit:  2,
			})
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if repository.listParams.UserID != userID || repository.listParams.Cursor != previousCursor || repository.listParams.Limit != 3 {
				t.Fatalf("repository List() params = %#v, want same user/cursor and limit 3", repository.listParams)
			}
			if repository.countUserID != userID || page.Total != 42 || page.CompletedTotal != 17 {
				t.Fatalf("count user = %s, total = %d, completed total = %d; want %s, 42 and 17", repository.countUserID, page.Total, page.CompletedTotal, userID)
			}
			if len(page.Items) != tc.wantItems {
				t.Fatalf("items = %d, want %d", len(page.Items), tc.wantItems)
			}
			if tc.wantCursor == nil {
				if page.NextCursor != nil {
					t.Fatalf("next cursor = %#v, want nil", page.NextCursor)
				}
				return
			}
			if page.NextCursor == nil || page.NextCursor.ID != tc.wantCursor.ID || !page.NextCursor.CreatedAt.Equal(tc.wantCursor.CreatedAt) {
				t.Fatalf("next cursor = %#v, want last visible task %#v", page.NextCursor, tc.wantCursor)
			}
			if page.Items[len(page.Items)-1].ID != tc.wantCursor.ID {
				t.Fatalf("last visible task = %s, want %s", page.Items[len(page.Items)-1].ID, tc.wantCursor.ID)
			}
			for _, item := range page.Items {
				if item.ID == lookahead.ID {
					t.Fatal("lookahead task must not be returned in the current page")
				}
			}
		})
	}
}

func TestServiceListPropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("database unavailable")
	userID := uuid.NewV7()

	t.Run("list", func(t *testing.T) {
		repository := &repositoryStub{listError: wantErr}
		_, err := NewService(repository).List(context.Background(), ListParams{UserID: userID, Limit: 2})
		if !errors.Is(err, wantErr) {
			t.Fatalf("List() error = %v, want %v", err, wantErr)
		}
		if repository.countUserID != uuid.Nil() {
			t.Fatal("Count() must not run after List() fails")
		}
	})

	t.Run("count", func(t *testing.T) {
		repository := &repositoryStub{countError: wantErr}
		_, err := NewService(repository).List(context.Background(), ListParams{UserID: userID, Limit: 2})
		if !errors.Is(err, wantErr) {
			t.Fatalf("List() error = %v, want %v", err, wantErr)
		}
	})
}

func TestTaskCursorRoundTripAndInvalidValues(t *testing.T) {
	cursor := ListCursor{
		ID:        uuid.NewV7(),
		CreatedAt: time.Date(2026, time.September, 25, 12, 30, 0, 123456000, time.FixedZone("UTC+5", 5*60*60)),
	}
	decoded, err := decodeTaskCursor(encodeTaskCursor(cursor))
	if err != nil || decoded.ID != cursor.ID || !decoded.CreatedAt.Equal(cursor.CreatedAt) {
		t.Fatalf("cursor round trip = %#v, %v; want %#v", decoded, err, cursor)
	}

	for _, value := range []string{"invalid", "", "djJ8MjAyNi0wOS0yNVQxMjowMDowMFp8YmFk", "djF8YmFkfGJhZA"} {
		if _, err := decodeTaskCursor(value); !errors.Is(err, errInvalidCursor) {
			t.Errorf("decodeTaskCursor(%q) error = %v, want invalid cursor", value, err)
		}
	}
}
