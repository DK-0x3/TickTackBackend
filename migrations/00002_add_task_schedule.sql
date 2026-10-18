-- +goose Up
ALTER TABLE tasks
    ADD COLUMN due_date DATE,
    ADD COLUMN due_time TIME(0),
    ADD CONSTRAINT tasks_due_time_requires_date CHECK (due_time IS NULL OR due_date IS NOT NULL),
    ADD CONSTRAINT tasks_due_time_minute_precision CHECK (
        due_time IS NULL OR EXTRACT(SECOND FROM due_time) = 0
    );

-- +goose Down
ALTER TABLE tasks
    DROP CONSTRAINT tasks_due_time_minute_precision,
    DROP CONSTRAINT tasks_due_time_requires_date,
    DROP COLUMN due_time,
    DROP COLUMN due_date;
