-- name: CreateTask :one
INSERT INTO tasks (
     user_id,
     title,
     description,
     priority
)
VALUES (
        sqlc.arg(user_id),
        sqlc.arg(title),
        sqlc.arg(description),
        sqlc.arg(priority)
       )
RETURNING *;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE user_id = sqlc.arg(user_id)
AND deleted_at IS NULL
AND (
    (
        sqlc.narg(cursor_created_at)::timestamptz IS NULL
        AND sqlc.narg(cursor_id)::uuid IS NULL
    )
    OR
    (created_at, id) < (
        sqlc.narg(cursor_created_at)::timestamptz,
        sqlc.narg(cursor_id)::uuid
    )
)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(cursor_limit);

-- name: CountTasks :one
SELECT
    count(*) AS total,
    count(*) FILTER (WHERE completed_at IS NOT NULL) AS completed_total
FROM tasks
WHERE user_id = sqlc.arg(user_id)
AND deleted_at IS NULL;

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
