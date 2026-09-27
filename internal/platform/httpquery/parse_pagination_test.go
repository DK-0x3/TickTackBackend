package httpquery

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestParsePaginationDefaultsAndValidValues(t *testing.T) {
	for _, tc := range []struct {
		name       string
		query      string
		wantLimit  int
		wantCursor string
	}{
		{name: "defaults", wantLimit: 20},
		{name: "explicit values", query: "limit=100&cursor=opaque", wantLimit: 100, wantCursor: "opaque"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ParsePagination(values, 20, 100)
			if err != nil || got.Limit != tc.wantLimit || got.Cursor != tc.wantCursor {
				t.Fatalf("ParsePagination() = %#v, %v; want limit %d, cursor %q", got, err, tc.wantLimit, tc.wantCursor)
			}
		})
	}
}

func TestParsePaginationRejectsInvalidValues(t *testing.T) {
	for _, query := range []string{
		"limit=", "limit=0", "limit=-1", "limit=101", "limit=abc", "limit=1&limit=2",
		"cursor=", "cursor=a&cursor=b", "cursor=" + strings.Repeat("x", 513),
	} {
		t.Run(query, func(t *testing.T) {
			values, err := url.ParseQuery(query)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParsePagination(values, 20, 100); !errors.Is(err, ErrInvalidPagination) {
				t.Fatalf("ParsePagination(%q) error = %v, want invalid pagination", query, err)
			}
		})
	}
}
