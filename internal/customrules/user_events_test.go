package customrules

import (
	"strings"
	"testing"
	"time"

	"zomboid-log-crawler/internal/config"
	"zomboid-log-crawler/internal/model"
)

func TestUserEventRule_LeaveIncludesDuration(t *testing.T) {
	rule, err := NewUserEventRule(config.DiscordUserEvents{
		Enabled:    true,
		LogType:    "user",
		JoinRegex:  `(?i)^([A-Za-z0-9_\- ]+) joined$`,
		LeaveRegex: `(?i)^([A-Za-z0-9_\- ]+) left$`,
		DieRegex:   `(?i)^([A-Za-z0-9_\- ]+) died$`,
		JoinColor:  "#22c55e",
		LeaveColor: "#ef4444",
		DieColor:   "#f59e0b",
	})
	if err != nil {
		t.Fatalf("new rule: %v", err)
	}

	base := time.Date(2026, 3, 12, 8, 0, 0, 0, time.UTC)
	_, matched := rule.Apply("user", model.LogEntry{Timestamp: base, Message: "Alice joined"})
	if !matched {
		t.Fatal("expected join to match")
	}

	msg, matched := rule.Apply("user", model.LogEntry{Timestamp: base.Add(90 * time.Second), Message: "Alice left"})
	if !matched || msg == nil || len(msg.Embeds) == 0 {
		t.Fatal("expected leave message")
	}
	if !strings.Contains(msg.Embeds[0].Description, "Online for") {
		t.Fatalf("expected duration in leave description, got: %s", msg.Embeds[0].Description)
	}
}

func TestUserEventRule_DieMatch(t *testing.T) {
	rule, err := NewUserEventRule(config.DiscordUserEvents{
		Enabled:    true,
		LogType:    "user",
		JoinRegex:  `(?i)^([A-Za-z0-9_\- ]+) joined$`,
		LeaveRegex: `(?i)^([A-Za-z0-9_\- ]+) left$`,
		DieRegex:   `(?i)^([A-Za-z0-9_\- ]+) died$`,
		JoinColor:  "#22c55e",
		LeaveColor: "#ef4444",
		DieColor:   "#f59e0b",
	})
	if err != nil {
		t.Fatalf("new rule: %v", err)
	}

	msg, matched := rule.Apply("user", model.LogEntry{Timestamp: time.Now(), Message: "Bob died"})
	if !matched || msg == nil || len(msg.Embeds) == 0 {
		t.Fatal("expected die match")
	}
	if msg.Embeds[0].Title != "Player Died" {
		t.Fatalf("unexpected title: %s", msg.Embeds[0].Title)
	}
}
