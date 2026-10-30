-- +goose Up
ALTER TABLE task_views
    ADD COLUMN show_priority BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE task_views
    DROP COLUMN show_priority;
