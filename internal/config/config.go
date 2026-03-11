package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultScanInterval = 3 * time.Second
)

type Config struct {
	LogDir       string  `yaml:"log_dir"`
	ScanInterval string  `yaml:"scan_interval"`
	Timezone     string  `yaml:"timezone"`
	API          API     `yaml:"api"`
	Discord      Discord `yaml:"discord"`
}

type API struct {
	Port int `yaml:"port"`
}

type Discord struct {
	WebhookURL string   `yaml:"webhook_url"`
	Keywords   []string `yaml:"keywords"`
	UserEvents DiscordUserEvents `yaml:"user_events"`
}

type DiscordUserEvents struct {
	Enabled    bool   `yaml:"enabled"`
	LogType    string `yaml:"log_type"`
	JoinRegex  string `yaml:"join_regex"`
	LeaveRegex string `yaml:"leave_regex"`
	DieRegex   string `yaml:"die_regex"`
	JoinColor  string `yaml:"join_color"`
	LeaveColor string `yaml:"leave_color"`
	DieColor   string `yaml:"die_color"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		LogDir:       "/logs",
		ScanInterval: defaultScanInterval.String(),
		Timezone:     "Asia/Ho_Chi_Minh",
		API:          API{Port: 8080},
		Discord: Discord{
			UserEvents: DiscordUserEvents{
				Enabled:    true,
				LogType:    "user",
				JoinRegex:  `(?i)^\d+\s+"([^"]+)"\s+fully connected\b`,
				LeaveRegex: `(?i)^\d+\s+"([^"]+)"\s+disconnected player\b`,
				DieRegex:   `(?i)^user\s+([A-Za-z0-9_\- ]+)\s+died\b`,
				JoinColor:  "#22c55e",
				LeaveColor: "#ef4444",
				DieColor:   "#f59e0b",
			},
		},
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config yaml: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	overrideFromEnv(cfg)
	trimKeywords(cfg)
	trimDiscordUserEvents(cfg)

	if cfg.LogDir == "" {
		return nil, fmt.Errorf("log_dir is required")
	}
	if cfg.API.Port <= 0 {
		return nil, fmt.Errorf("api.port must be > 0")
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Ho_Chi_Minh"
	}

	return cfg, nil
}

func (c *Config) ScanEvery() time.Duration {
	if c.ScanInterval == "" {
		return defaultScanInterval
	}
	d, err := time.ParseDuration(c.ScanInterval)
	if err != nil || d <= 0 {
		return defaultScanInterval
	}
	return d
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("LOG_DIR"); v != "" {
		cfg.LogDir = v
	}
	if v := os.Getenv("SCAN_INTERVAL"); v != "" {
		cfg.ScanInterval = v
	}
	if v := os.Getenv("TIMEZONE"); v != "" {
		cfg.Timezone = v
	}
	if v := os.Getenv("API_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.API.Port = p
		}
	}
	if v := os.Getenv("DISCORD_WEBHOOK_URL"); v != "" {
		cfg.Discord.WebhookURL = v
	}
	if v := os.Getenv("DISCORD_KEYWORDS"); v != "" {
		cfg.Discord.Keywords = strings.Split(v, ",")
	}
}

func trimKeywords(cfg *Config) {
	result := make([]string, 0, len(cfg.Discord.Keywords))
	for _, keyword := range cfg.Discord.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		result = append(result, keyword)
	}
	cfg.Discord.Keywords = result
}

func trimDiscordUserEvents(cfg *Config) {
	cfg.Discord.UserEvents.LogType = strings.TrimSpace(cfg.Discord.UserEvents.LogType)
	cfg.Discord.UserEvents.JoinRegex = strings.TrimSpace(cfg.Discord.UserEvents.JoinRegex)
	cfg.Discord.UserEvents.LeaveRegex = strings.TrimSpace(cfg.Discord.UserEvents.LeaveRegex)
	cfg.Discord.UserEvents.DieRegex = strings.TrimSpace(cfg.Discord.UserEvents.DieRegex)
	cfg.Discord.UserEvents.JoinColor = strings.TrimSpace(cfg.Discord.UserEvents.JoinColor)
	cfg.Discord.UserEvents.LeaveColor = strings.TrimSpace(cfg.Discord.UserEvents.LeaveColor)
	cfg.Discord.UserEvents.DieColor = strings.TrimSpace(cfg.Discord.UserEvents.DieColor)

	if cfg.Discord.UserEvents.LogType == "" {
		cfg.Discord.UserEvents.LogType = "user"
	}
}
