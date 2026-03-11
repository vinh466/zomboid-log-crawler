package parser

import (
	"testing"
	"time"
)

func TestParseLogLine(t *testing.T) {
	loc := time.UTC
	entry, err := ParseLogLine("[11-03-26 08:32:05.088] Something happened", loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Message != "Something happened" {
		t.Fatalf("unexpected message: %s", entry.Message)
	}
	if got := entry.Timestamp.Format("02-01-06 15:04:05.000"); got != "11-03-26 08:32:05.088" {
		t.Fatalf("unexpected timestamp: %s", got)
	}
}
