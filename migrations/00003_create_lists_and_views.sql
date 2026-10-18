-- +goose Up
CREATE TABLE lists (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    kind TEXT NOT NULL,
    title VARCHAR(500) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 1,

    CONSTRAINT lists_kind_valid CHECK (kind IN ('inbox', 'custom')),
    CONSTRAINT lists_title_not_blank CHECK (char_length(btrim(title)) > 0),
    CONSTRAINT lists_version_positive CHECK (version > 0),
    CONSTRAINT lists_user_id_id_unique UNIQUE (user_id, id)
);

CREATE UNIQUE INDEX lists_one_inbox_per_user_idx
    ON lists (user_id)
    WHERE kind = 'inbox';

INSERT INTO lists (user_id, kind, title)
SELECT DISTINCT user_id, 'inbox', 'Inbox'
FROM tasks;

ALTER TABLE tasks ADD COLUMN list_id UUID;

UPDATE tasks AS task
SET list_id = list.id
FROM lists AS list
WHERE task.user_id = list.user_id
  AND list.kind = 'inbox';

ALTER TABLE tasks
    ALTER COLUMN list_id SET NOT NULL,
    ADD CONSTRAINT tasks_list_owner_fk FOREIGN KEY (user_id, list_id)
        REFERENCES lists (user_id, id);

CREATE INDEX tasks_user_list_created_at_active_idx
    ON tasks (user_id, list_id, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE task_views (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL,
    list_id UUID NOT NULL,
    section TEXT NOT NULL,
    position INTEGER NOT NULL,
    sort_field TEXT NOT NULL DEFAULT 'created_at',
    sort_direction TEXT NOT NULL DEFAULT 'desc',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT task_views_list_owner_fk FOREIGN KEY (user_id, list_id)
        REFERENCES lists (user_id, id),
    CONSTRAINT task_views_section_valid CHECK (
        section IN ('overdue', 'today', 'tomorrow', 'next_week', 'next', 'completed', 'all')
    ),
    CONSTRAINT task_views_sort_field_valid CHECK (
        sort_field IN ('title', 'due_at', 'created_at', 'updated_at', 'priority')
    ),
    CONSTRAINT task_views_sort_direction_valid CHECK (sort_direction IN ('asc', 'desc')),
    CONSTRAINT task_views_version_positive CHECK (version > 0)
);

CREATE INDEX task_views_list_position_idx
    ON task_views (user_id, list_id, position, id);

CREATE UNIQUE INDEX task_views_one_inbox_section_idx
    ON task_views (list_id, section)
    WHERE section <> 'all';

INSERT INTO task_views (user_id, list_id, section, position)
SELECT list.user_id, list.id, section.name, section.position
FROM lists AS list
CROSS JOIN (VALUES
    ('overdue', 0),
    ('today', 1),
    ('tomorrow', 2),
    ('next_week', 3),
    ('next', 4),
    ('completed', 5)
) AS section(name, position)
WHERE list.kind = 'inbox';

-- +goose Down
ALTER TABLE tasks DROP CONSTRAINT tasks_list_owner_fk;
DROP TABLE task_views;
DROP INDEX tasks_user_list_created_at_active_idx;
ALTER TABLE tasks DROP COLUMN list_id;
DROP TABLE lists;
