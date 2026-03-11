package customrules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"zomboid-log-crawler/internal/config"
	"zomboid-log-crawler/internal/model"
	"zomboid-log-crawler/internal/notifier"
	"zomboid-log-crawler/internal/rules"
)

type UserEventRule struct {
	logType   string
	joinRegex *regexp.Regexp
	leaveRegex *regexp.Regexp
	dieRegex  *regexp.Regexp

	joinColor  int
	leaveColor int
	dieColor   int

	mu       sync.Mutex
	sessions map[string]time.Time
}

func Build(discordCfg config.Discord) ([]rules.Rule, error) {
	if !discordCfg.UserEvents.Enabled {
		return nil, nil
	}
	rule, err := NewUserEventRule(discordCfg.UserEvents)
	if err != nil {
		return nil, err
	}
	return []rules.Rule{rule}, nil
}

func NewUserEventRule(cfg config.DiscordUserEvents) (*UserEventRule, error) {
	joinRegex, err := regexp.Compile(cfg.JoinRegex)
	if err != nil {
		return nil, fmt.Errorf("compile join_regex: %w", err)
	}
	leaveRegex, err := regexp.Compile(cfg.LeaveRegex)
	if err != nil {
		return nil, fmt.Errorf("compile leave_regex: %w", err)
	}
	dieRegex, err := regexp.Compile(cfg.DieRegex)
	if err != nil {
		return nil, fmt.Errorf("compile die_regex: %w", err)
	}

	joinColor, err := parseHexColor(cfg.JoinColor)
	if err != nil {
		return nil, fmt.Errorf("parse join_color: %w", err)
	}
	leaveColor, err := parseHexColor(cfg.LeaveColor)
	if err != nil {
		return nil, fmt.Errorf("parse leave_color: %w", err)
	}
	dieColor, err := parseHexColor(cfg.DieColor)
	if err != nil {
		return nil, fmt.Errorf("parse die_color: %w", err)
	}

	return &UserEventRule{
		logType:    cfg.LogType,
		joinRegex:  joinRegex,
		leaveRegex: leaveRegex,
		dieRegex:   dieRegex,
		joinColor:  joinColor,
		leaveColor: leaveColor,
		dieColor:   dieColor,
		sessions:   make(map[string]time.Time),
	}, nil
}

func (r *UserEventRule) Name() string {
	return "UserEventRule"
}

func (r *UserEventRule) Apply(logType string, entry model.LogEntry) (*notifier.Message, bool) {
	if !strings.EqualFold(logType, r.logType) {
		return nil, false
	}

	if username, ok := matchUsername(r.joinRegex, entry.Message); ok {
		r.mu.Lock()
		r.sessions[username] = entry.Timestamp
		r.mu.Unlock()

		return &notifier.Message{Embeds: []notifier.Embed{{
			Title:       "User Joined",
			Description: fmt.Sprintf("`%s` joined at `%s`", username, entry.Timestamp.Format("15:04:05")),
			Color:       r.joinColor,
		}}}, true
	}

	if username, ok := matchUsername(r.leaveRegex, entry.Message); ok {
		r.mu.Lock()
		joinedAt, hasSession := r.sessions[username]
		if hasSession {
			delete(r.sessions, username)
		}
		r.mu.Unlock()

		description := fmt.Sprintf("`%s` left at `%s`", username, entry.Timestamp.Format("15:04:05"))
		if hasSession {
			description = fmt.Sprintf("%s\nOnline for `%s`", description, humanDuration(entry.Timestamp.Sub(joinedAt)))
		}

		return &notifier.Message{Embeds: []notifier.Embed{{
			Title:       "User Left",
			Description: description,
			Color:       r.leaveColor,
		}}}, true
	}

	if username, ok := matchUsername(r.dieRegex, entry.Message); ok {
		return &notifier.Message{Embeds: []notifier.Embed{{
			Title:       "Player Died",
			Description: fmt.Sprintf("`%s` died\n%s", username, entry.Message),
			Color:       r.dieColor,
		}}}, true
	}

	return nil, false
}

func matchUsername(re *regexp.Regexp, message string) (string, bool) {
	matches := re.FindStringSubmatch(message)
	if len(matches) < 2 {
		return "", false
	}
	username := strings.TrimSpace(matches[1])
	if username == "" {
		return "", false
	}
	return username, true
}

func parseHexColor(hex string) (int, error) {
	hex = strings.TrimSpace(strings.TrimPrefix(hex, "#"))
	if len(hex) != 6 {
		return 0, fmt.Errorf("invalid hex color: %s", hex)
	}
	v, err := strconv.ParseInt(hex, 16, 32)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

func humanDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
