package tasks

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"uuid"
)

var errInvalidCursor = errors.New("invalid task cursor")

func encodeTaskCursor(cursor ListCursor) string {
	value := "v1|" + cursor.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + cursor.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeTaskCursor(value string) (ListCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return ListCursor{}, errInvalidCursor
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 3 || parts[0] != "v1" {
		return ListCursor{}, errInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil || createdAt.IsZero() {
		return ListCursor{}, errInvalidCursor
	}

	id, err := uuid.Parse(parts[2])
	if err != nil || id == uuid.Nil() {
		return ListCursor{}, errInvalidCursor
	}

	return ListCursor{CreatedAt: createdAt, ID: id}, nil
}
