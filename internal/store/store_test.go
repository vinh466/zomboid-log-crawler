package store

import (
	"testing"
	"time"

	"zomboid-log-crawler/internal/model"
)

func TestQuery(t *testing.T) {
	s := New()
	s.EnsureType("combat")
	base := time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC)
	s.Append("combat", []model.LogEntry{
		{Timestamp: base.Add(1 * time.Minute), Message: "player joined"},
		{Timestamp: base.Add(2 * time.Minute), Message: "error happened"},
		{Timestamp: base.Add(3 * time.Minute), Message: "player left"},
	})

	from := base.Add(90 * time.Second)
	items, total, ok := s.Query("combat", QueryOptions{Q: "error", From: &from, Limit: 10})
	if !ok {
		t.Fatal("expected logType exists")
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected query result total=%d len=%d", total, len(items))
	}
}
