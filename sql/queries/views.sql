-- name: EnsureInbox :one
INSERT INTO lists (user_id, kind, title)
VALUES (sqlc.arg(user_id), 'inbox', 'Inbox')
ON CONFLICT (user_id) WHERE kind = 'inbox'
DO UPDATE SET title = lists.title
RETURNING *;

-- name: GetInbox :one
SELECT * FROM lists
WHERE user_id = sqlc.arg(user_id)
  AND kind = 'inbox';

-- name: EnsureInboxViews :exec
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
WHERE list.id = sqlc.arg(list_id)
  AND list.user_id = sqlc.arg(user_id)
  AND list.kind = 'inbox'
ON CONFLICT (list_id, section) WHERE section <> 'all' DO NOTHING;

-- name: ListTaskViews :many
SELECT view.*
FROM task_views AS view
JOIN lists AS list ON list.id = view.list_id AND list.user_id = view.user_id
WHERE view.user_id = sqlc.arg(user_id)
  AND view.list_id = sqlc.arg(list_id)
  AND list.deleted_at IS NULL
ORDER BY view.position, view.id;

-- name: GetTaskView :one
SELECT view.*
FROM task_views AS view
JOIN lists AS list ON list.id = view.list_id AND list.user_id = view.user_id
WHERE view.id = sqlc.arg(view_id)
  AND view.user_id = sqlc.arg(user_id)
  AND list.deleted_at IS NULL;

-- name: SaveTaskViewSettings :one
WITH previous AS (
    SELECT *
    FROM task_views
    WHERE task_views.id = sqlc.arg(view_id)
      AND task_views.user_id = sqlc.arg(user_id)
      AND task_views.version = sqlc.arg(expected_version)
        FOR UPDATE
),
     updated AS (
         UPDATE task_views AS target
             SET sort_field = sqlc.arg(sort_field),
                 sort_direction = sqlc.arg(sort_direction),
                 show_completed = sqlc.arg(show_completed),
                 version = target.version + 1,
                 updated_at = now()
             FROM previous
             WHERE target.id = previous.id
             RETURNING target.*
     ),
     completed_updated AS (
         UPDATE task_views AS completed
             SET version = completed.version + 1,
                 updated_at = now()
             FROM updated
                 JOIN previous ON previous.id = updated.id
             WHERE completed.user_id = updated.user_id
                 AND completed.list_id = updated.list_id
                 AND completed.section = 'completed'
                 AND updated.section IN (
                                         'today', 'tomorrow', 'next_week', 'next'
                     )
                 AND previous.show_completed IS DISTINCT FROM updated.show_completed
             RETURNING completed.id
     )
SELECT updated.*
FROM updated;
