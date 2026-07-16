// Package datetime provides DateTime, a JSON-friendly time type for API responses.
package datetime

import (
	"fmt"
	"strings"
	"time"
)

// Layout is the unified API datetime format (second precision).
const Layout = "2006-01-02 15:04:05"

// DateTime marshals to JSON as "2006-01-02 15:04:05" in local timezone.
// It does not embed time.Time to avoid inheriting RFC3339 MarshalJSON.
type DateTime struct {
	t time.Time
}

// From wraps a time.Time as DateTime.
func From(t time.Time) DateTime {
	return DateTime{t: t}
}

// Time returns the underlying time.Time.
func (t DateTime) Time() time.Time {
	return t.t
}

// IsZero reports whether the datetime is the zero value.
func (t DateTime) IsZero() bool {
	return t.t.IsZero()
}

// After reports whether t is after u (useful for sorting VO lists).
func (t DateTime) After(u DateTime) bool {
	return t.t.After(u.t)
}

func (t DateTime) MarshalJSON() ([]byte, error) {
	if t.t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + t.t.In(time.Local).Format(Layout) + `"`), nil
}

func (t *DateTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		t.t = time.Time{}
		return nil
	}
	parsed, err := time.ParseInLocation(Layout, s, time.Local)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return fmt.Errorf("datetime: parse %q: %w", s, err)
		}
		parsed = parsed.In(time.Local)
	}
	t.t = parsed
	return nil
}

func (t DateTime) String() string {
	if t.t.IsZero() {
		return ""
	}
	return t.t.In(time.Local).Format(Layout)
}
