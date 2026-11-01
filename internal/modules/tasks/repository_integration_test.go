package tasks

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
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
		"../../../migrations/00003_create_lists_and_views.sql",
		"../../../migrations/00004_add_show_completed.sql",
		"../../../migrations/00005_add_show_priority.sql",
		"../../../migrations/00006_add_view_collapsed.sql",
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
	userID := uuid.NewV7()
	otherUserID := uuid.NewV7()
	created := make([]Task, 0, 3)
	for _, title := range []string{"old", "middle", "new"} {
		task, err := repository.Create(ctx, CreateParams{UserID: userID, Title: title, Priority: PriorityNone})
		if err != nil {
			t.Fatalf("create %s: %v", title, err)
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

	if _, err := repository.Get(ctx, userID, otherTask.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get another user's task error = %v, want not found", err)
	}
	completed, err := repository.SetCompleted(ctx, userID, created[0].ID, true)
	if err != nil || completed.CompletedAt == nil || completed.Version != created[0].Version+1 {
		t.Fatalf("complete task = %#v, %v", completed, err)
	}
}
