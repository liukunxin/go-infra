package datetime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateTimeJSON(t *testing.T) {
	loc := time.Local
	parsed, err := time.ParseInLocation(Layout, "2024-01-15 10:23:45", loc)
	if err != nil {
		t.Fatal(err)
	}
	dt := From(parsed)

	b, err := json.Marshal(dt)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2024-01-15 10:23:45"` {
		t.Fatalf("marshal = %s", b)
	}

	var out DateTime
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Time().Equal(parsed) {
		t.Fatalf("unmarshal = %v, want %v", out.Time(), parsed)
	}
}

func TestDateTimeZeroJSON(t *testing.T) {
	var dt DateTime
	b, err := json.Marshal(dt)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "null" {
		t.Fatalf("zero marshal = %s, want null", b)
	}
}

func TestDateTimeUnmarshalRFC3339(t *testing.T) {
	var dt DateTime
	if err := json.Unmarshal([]byte(`"2024-01-15T10:23:45+08:00"`), &dt); err != nil {
		t.Fatal(err)
	}
	if dt.IsZero() {
		t.Fatal("expected non-zero after RFC3339 unmarshal")
	}
}
