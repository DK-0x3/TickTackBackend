-- +goose Up
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    priority TEXT NOT NULL DEFAULT 'none',
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 1,

    CONSTRAINT tasks_title_not_blank CHECK (char_length(btrim(title)) > 0),
    CONSTRAINT tasks_description_max_length CHECK (
        description IS NULL OR char_length(description) <= 10000
    ),
    CONSTRAINT tasks_priority_valid CHECK (
        priority IN ('none', 'low', 'medium', 'high')
    ),
    CONSTRAINT tasks_version_positive CHECK (version > 0)
);

CREATE INDEX tasks_user_created_at_active_idx
    ON tasks (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE tasks;
