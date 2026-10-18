package tasks

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	dbsqlc "github.com/DK-0x3/TickTackBackend/internal/platform/postgres/sqlc"
)

func TestRepositoryAndServiceWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create isolated schema: %v", err)
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop isolated schema: %v", err)
		}
		admin.Close()
	})

	for _, path := range []string{
		"../../../migrations/00001_create_tasks.sql",
		"../../../migrations/00002_add_task_schedule.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		up := strings.Split(string(migration), "-- +goose Down")[0]
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("apply task migration %s: %v", path, err)
		}
	}

	repository := NewRepository(dbsqlc.New(pool))
	service := NewService(repository)
	userID := uuid.NewV7()
	otherUserID := uuid.NewV7()
	baseTime := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	created := make([]Task, 0, 3)
	for index, title := range []string{"old", "middle", "new"} {
		task, err := repository.Create(ctx, CreateParams{UserID: userID, Title: title, Priority: PriorityNone})
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
		}
		if _, err := pool.Exec(ctx, "UPDATE tasks SET created_at = $1 WHERE id = $2", baseTime.Add(time.Duration(index)*time.Minute), task.ID); err != nil {
			t.Fatal(err)
		}
		created = append(created, task)
	}
	otherTask, err := repository.Create(ctx, CreateParams{UserID: otherUserID, Title: "private", Priority: PriorityNone})
	if err != nil {
		t.Fatal(err)
	}
	deletedTask, err := repository.Create(ctx, CreateParams{UserID: userID, Title: "deleted", Priority: PriorityNone})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE tasks SET deleted_at = now() WHERE id = $1", deletedTask.ID); err != nil {
		t.Fatal(err)
	}

	firstPage, err := service.List(ctx, ListParams{UserID: userID, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if firstPage.Total != 3 || len(firstPage.Items) != 2 || firstPage.NextCursor == nil {
		t.Fatalf("first page = %#v, want 2 tasks, cursor and total 3", firstPage)
	}
	if firstPage.Items[0].ID != created[2].ID || firstPage.Items[1].ID != created[1].ID {
		t.Fatalf("first page IDs = %s, %s; want new then middle", firstPage.Items[0].ID, firstPage.Items[1].ID)
	}
	secondPage, err := service.List(ctx, ListParams{UserID: userID, Cursor: firstPage.NextCursor, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if secondPage.Total != 3 || len(secondPage.Items) != 1 || secondPage.Items[0].ID != created[0].ID || secondPage.NextCursor != nil {
		t.Fatalf("second page = %#v, want only old task, total 3 and no cursor", secondPage)
	}

	if _, err := repository.Get(ctx, userID, otherTask.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get another user's task error = %v, want not found", err)
	}
	completed, err := repository.SetCompleted(ctx, userID, created[0].ID, true)
	if err != nil || completed.CompletedAt == nil || completed.Version != created[0].Version+1 {
		t.Fatalf("complete task = %#v, %v", completed, err)
	}
}
