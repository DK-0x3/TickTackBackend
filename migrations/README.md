# Migrations

Database schema changes are stored here as sequential Goose SQL migrations.
Filenames use zero-padded numeric versions so Goose and sqlc process them in
the same lexicographic order.

Applied migrations must not be edited. Add a new migration for every subsequent
schema change.
