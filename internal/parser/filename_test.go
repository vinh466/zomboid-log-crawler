package parser

import (
	"testing"
	"time"
)

func TestParseFileName(t *testing.T) {
	loc := time.UTC
	info, err := ParseFileName("21-03-26_16-25-51_combat.txt", loc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.LogType != "combat" {
		t.Fatalf("unexpected logType: %s", info.LogType)
	}
	if got := info.FileTimestamp.Format("02-01-06_15-04-05"); got != "21-03-26_16-25-51" {
		t.Fatalf("unexpected timestamp: %s", got)
	}
}
