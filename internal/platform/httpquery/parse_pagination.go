package httpquery

import (
	"errors"
	"net/url"
	"strconv"
)

var ErrInvalidPagination = errors.New("invalid pagination params")

type PaginationParams struct {
	Limit  int
	Cursor string
}

func ParsePagination(
	values url.Values,
	defaultLimit int,
	maxLimit int,
) (PaginationParams, error) {
	params := PaginationParams{
		Limit: defaultLimit,
	}

	rawLimits, exists := values["limit"]
	if exists {
		if len(rawLimits) != 1 || rawLimits[0] == "" {
			return PaginationParams{}, ErrInvalidPagination
		}

		limit, err := strconv.Atoi(rawLimits[0])
		if err != nil || limit < 1 || limit > maxLimit {
			return PaginationParams{}, ErrInvalidPagination
		}

		params.Limit = limit
	}

	rawCursors, exists := values["cursor"]
	if exists {
		if len(rawCursors) != 1 || rawCursors[0] == "" || len(rawCursors[0]) > 512 {
			return PaginationParams{}, ErrInvalidPagination
		}
		params.Cursor = rawCursors[0]
	}

	return params, nil
}
