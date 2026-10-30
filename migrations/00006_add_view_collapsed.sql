-- +goose Up
ALTER TABLE task_views
    ADD COLUMN is_collapsed BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE task_views
    DROP COLUMN is_collapsed;
