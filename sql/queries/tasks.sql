-- name: CreateTask :one
WITH inbox AS (
    INSERT INTO lists (user_id, kind, title)
    VALUES (sqlc.arg(user_id), 'inbox', 'Inbox')
    ON CONFLICT (user_id) WHERE kind = 'inbox'
    DO UPDATE SET title = lists.title
    RETURNING id
)
INSERT INTO tasks (
     user_id,
     list_id,
     title,
     description,
     priority,
     due_date,
     due_time
)
SELECT
        sqlc.arg(user_id),
        inbox.id,
        sqlc.arg(title),
        sqlc.arg(description),
        sqlc.arg(priority),
        sqlc.narg(due_date),
        sqlc.narg(due_time)
FROM inbox
RETURNING *;

-- name: ListTasks :many
SELECT task.* FROM tasks AS task
JOIN lists AS list ON list.id = task.list_id AND list.user_id = task.user_id
WHERE task.user_id = sqlc.arg(user_id)
AND list.kind = 'inbox'
AND list.deleted_at IS NULL
AND task.deleted_at IS NULL
AND (
    (
        sqlc.narg(cursor_created_at)::timestamptz IS NULL
        AND sqlc.narg(cursor_id)::uuid IS NULL
    )
    OR
    (task.created_at, task.id) < (
        sqlc.narg(cursor_created_at)::timestamptz,
        sqlc.narg(cursor_id)::uuid
    )
)
ORDER BY task.created_at DESC, task.id DESC
LIMIT sqlc.arg(cursor_limit);

-- name: CountTasks :one
SELECT
    count(*) AS total,
    count(*) FILTER (WHERE completed_at IS NOT NULL) AS completed_total
FROM tasks
JOIN lists AS list ON list.id = tasks.list_id AND list.user_id = tasks.user_id
WHERE tasks.user_id = sqlc.arg(user_id)
AND list.kind = 'inbox'
AND list.deleted_at IS NULL
AND tasks.deleted_at IS NULL;

-- name: GetTask :one
SELECT * FROM tasks
WHERE user_id = sqlc.arg(user_id)
AND id = sqlc.arg(id)
AND deleted_at IS NULL;

-- name: SetTaskCompleted :one
UPDATE tasks
SET
    completed_at = CASE WHEN sqlc.arg(completed)::boolean THEN NOW() END,
    updated_at = NOW(),
    version = version + 1
WHERE user_id = sqlc.arg(user_id)
AND id = sqlc.arg(id)
AND deleted_at IS NULL
RETURNING *;

-- name: SetTaskSchedule :one
UPDATE tasks
SET due_date = sqlc.narg(due_date),
    due_time = sqlc.narg(due_time),
    updated_at = NOW(),
    version = version + 1
WHERE user_id = sqlc.arg(user_id)
AND id = sqlc.arg(id)
AND deleted_at IS NULL
RETURNING *;

-- name: DeleteTask :execrows
UPDATE tasks
SET deleted_at = NOW(),
    updated_at = NOW(),
    version = version + 1
WHERE user_id = sqlc.arg(user_id)
AND id = sqlc.arg(id)
AND deleted_at IS NULL;

-- name: RestoreTask :one
UPDATE tasks
SET deleted_at = NULL,
    updated_at = NOW(),
    version = version + 1
WHERE user_id = sqlc.arg(user_id)
AND id = sqlc.arg(id)
AND deleted_at IS NOT NULL
RETURNING *;


-- name: CountTasksForView :one
WITH classified AS (
    SELECT
        task.*,
        target.section AS requested_section,
        target.show_completed AS requested_show_completed,
        target.sort_field,
        target.sort_direction,
        CASE
            WHEN task.due_date IS NULL THEN
                CASE
                    WHEN task.completed_at IS NULL
                        OR (
                             task.completed_at >= sqlc.arg(day_start)::timestamptz
                                 AND task.completed_at < sqlc.arg(day_end)::timestamptz
                             )
                        THEN 'today'
                    ELSE 'completed'
                    END
            WHEN task.due_date < sqlc.arg(today)::date THEN
                CASE
                    WHEN task.completed_at IS NULL THEN 'overdue'
                    ELSE 'completed'
                    END
            WHEN task.due_date = sqlc.arg(today)::date
                THEN 'today'
            WHEN task.due_date = sqlc.arg(today)::date + 1
                THEN 'tomorrow'
            WHEN task.due_date < sqlc.arg(today)::date + 7
                THEN 'next_week'
            ELSE 'next'
            END::text AS source_section
    FROM task_views AS target
             JOIN lists AS list
                  ON list.id = target.list_id
                      AND list.user_id = target.user_id
             JOIN tasks AS task
                  ON task.list_id = target.list_id
                      AND task.user_id = target.user_id
    WHERE target.id = sqlc.arg(view_id)
      AND target.user_id = sqlc.arg(user_id)
      AND list.deleted_at IS NULL
      AND task.deleted_at IS NULL
),
     visible AS (
         SELECT
             classified.*,
             CASE
                 WHEN requested_section = 'completed' THEN
                     completed_at IS NOT NULL
                         AND (
                         source_section = 'completed'
                             OR source.show_completed IS FALSE
                         )
                 WHEN requested_section = 'all' THEN
                     completed_at IS NULL OR requested_show_completed
                 ELSE
                     source_section = requested_section
                         AND (
                         completed_at IS NULL
                             OR requested_show_completed
                         )
                 END AS is_visible
         FROM classified
                  LEFT JOIN task_views AS source
                            ON source.user_id = classified.user_id
                                AND source.list_id = classified.list_id
                                AND source.section = classified.source_section
     )

SELECT
    count(*) AS total,
    count(*) FILTER (
        WHERE completed_at IS NOT NULL
        ) AS completed_total,
    count(*) FILTER (
        WHERE is_visible
        ) AS visible_total
FROM visible
WHERE CASE
          WHEN requested_section = 'completed' THEN is_visible
          ELSE
              requested_section = 'all'
                  OR source_section = requested_section
          END;

-- name: ListTasksForView :many
WITH classified AS (
    SELECT
        task.*,
        target.section AS requested_section,
        target.show_completed AS requested_show_completed,
        target.sort_field,
        target.sort_direction,
        CASE
            WHEN task.due_date IS NULL THEN
                CASE
                    WHEN task.completed_at IS NULL
                        OR (
                             task.completed_at >= sqlc.arg(day_start)::timestamptz
                                 AND task.completed_at < sqlc.arg(day_end)::timestamptz
                             )
                        THEN 'today'
                    ELSE 'completed'
                    END
            WHEN task.due_date < sqlc.arg(today)::date THEN
                CASE
                    WHEN task.completed_at IS NULL THEN 'overdue'
                    ELSE 'completed'
                    END
            WHEN task.due_date = sqlc.arg(today)::date
                THEN 'today'
            WHEN task.due_date = sqlc.arg(today)::date + 1
                THEN 'tomorrow'
            WHEN task.due_date < sqlc.arg(today)::date + 7
                THEN 'next_week'
            ELSE 'next'
            END::text AS source_section
    FROM task_views AS target
             JOIN lists AS list
                  ON list.id = target.list_id
                      AND list.user_id = target.user_id
             JOIN tasks AS task
                  ON task.list_id = target.list_id
                      AND task.user_id = target.user_id
    WHERE target.id = sqlc.arg(view_id)
      AND target.user_id = sqlc.arg(user_id)
      AND list.deleted_at IS NULL
      AND task.deleted_at IS NULL
),
     visible AS (
         SELECT
             classified.*,
             CASE
                 WHEN requested_section = 'completed' THEN
                     completed_at IS NOT NULL
                         AND (
                         source_section = 'completed'
                             OR source.show_completed IS FALSE
                         )
                 WHEN requested_section = 'all' THEN
                     completed_at IS NULL OR requested_show_completed
                 ELSE
                     source_section = requested_section
                         AND (
                         completed_at IS NULL
                             OR requested_show_completed
                         )
                 END AS is_visible
         FROM classified
                  LEFT JOIN task_views AS source
                            ON source.user_id = classified.user_id
                                AND source.list_id = classified.list_id
                                AND source.section = classified.source_section
     ),
scoped AS (
    SELECT
        task.*,
        COALESCE(CASE task.sort_field
                     WHEN 'title' THEN lower(task.title)
                     WHEN 'due_at' THEN COALESCE(
                             to_char(task.due_date, 'YYYY-MM-DD') || 'T' ||
                             to_char(
                                     COALESCE(task.due_time, time '00:00'),
                                     'HH24:MI:SS'
                             ),
                             ''
                                        )
                     WHEN 'created_at' THEN
                         to_char(
                                 task.created_at AT TIME ZONE 'UTC',
                                 'YYYY-MM-DD"T"HH24:MI:SS.US'
                         )
                     WHEN 'updated_at' THEN
                         to_char(
                                 task.updated_at AT TIME ZONE 'UTC',
                                 'YYYY-MM-DD"T"HH24:MI:SS.US'
                         )
                     WHEN 'priority' THEN CASE task.priority
                                              WHEN 'none' THEN '0'
                                              WHEN 'low' THEN '1'
                                              WHEN 'medium' THEN '2'
                                              WHEN 'high' THEN '3'
                         END
                     END, '')::text AS sort_key
    FROM visible AS task
    WHERE task.is_visible
)
SELECT * FROM scoped
WHERE (
    sqlc.narg(cursor_sort_key)::text IS NULL
    AND sqlc.narg(cursor_id)::uuid IS NULL
) OR (
    sort_direction = 'asc'
    AND (sort_key COLLATE "C", id) > (sqlc.narg(cursor_sort_key)::text COLLATE "C", sqlc.narg(cursor_id)::uuid)
) OR (
    sort_direction = 'desc'
    AND (sort_key COLLATE "C", id) < (sqlc.narg(cursor_sort_key)::text COLLATE "C", sqlc.narg(cursor_id)::uuid)
)
ORDER BY
    CASE WHEN sort_direction = 'asc' THEN sort_key END COLLATE "C" ASC,
    CASE WHEN sort_direction = 'desc' THEN sort_key END COLLATE "C" DESC,
    CASE WHEN sort_direction = 'asc' THEN id END ASC,
    CASE WHEN sort_direction = 'desc' THEN id END DESC
LIMIT sqlc.arg(cursor_limit);
