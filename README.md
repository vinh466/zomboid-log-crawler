# log-crawler

Go service to read game log files, expose no-auth REST API, watch new appended lines, and send Discord webhook notifications when message matches configured keywords.

## Supported filename and line format

- Filename: `DD-MM-YY_HH-mm-ss_logType.txt`
- Line: `[DD-MM-YY HH:MM:SS.mmm] message`

Only files that match filename pattern are considered. If multiple files exist for same `logType`, app only tracks the newest timestamp file.

## Configuration

Copy `config.yaml.example` to `config.yaml` and update values.

```yaml
log_dir: /logs
scan_interval: 3s
timezone: Asia/Ho_Chi_Minh
api:
  port: 8080
discord:
  webhook_url: ""
  keywords:
    - error
    - failed
  user_events:
    enabled: true
    log_type: user
    join_regex: '(?i)\b(?:player\s+)?"?([A-Za-z0-9_\- ]+)"?\b.*\b(joined|connect(?:ed)?)\b'
    leave_regex: '(?i)\b(?:player\s+)?"?([A-Za-z0-9_\- ]+)"?\b.*\b(left|disconnect(?:ed)?|quit)\b'
    die_regex: '(?i)\b(?:player\s+)?"?([A-Za-z0-9_\- ]+)"?\b.*\b(died|dead|killed)\b'
    join_color: "#22c55e"
    leave_color: "#ef4444"
    die_color: "#f59e0b"
```

Environment overrides:

- `CONFIG_PATH`
- `LOG_DIR`
- `SCAN_INTERVAL`
- `TIMEZONE`
- `API_PORT`
- `DISCORD_WEBHOOK_URL`
- `DISCORD_KEYWORDS` (comma-separated)

## Custom Notification Rules

- Built-in rule: `discord.user_events` for `join/leave/die` on a configured `log_type` (default `user`).
- `join` is sent as green embed, `leave` as red embed, `die` as amber embed.
- For `leave`, app computes online duration if matching `join` was seen before.

Extension point for your own logic:

- Add a new rule in `internal/customrules` implementing `rules.Rule`.
- Register it in `internal/customrules/user_events.go` function `Build`.
- Rule engine lives at `internal/rules/engine.go` and is called by watcher for each new log entry.

## API

- `GET /health`
- `GET /api/log-types`
- `GET /api/logs/{logType}?q=&from=&to=&limit=&offset=`

Time query formats accepted by `from` and `to`:

- RFC3339 / RFC3339Nano
- `DD-MM-YY HH:MM:SS.mmm`

If only `from` is provided, `to` is treated as now.

## Run local

```powershell
Copy-Item config.yaml.example config.yaml
$env:CONFIG_PATH = "config.yaml"
go mod tidy
go run .\main.go
```

## Run with Docker

```powershell
$env:HOST_LOG_DIR = "D:/path/to/game/logs"
$env:DISCORD_WEBHOOK_URL = "https://discord.com/api/webhooks/..."
docker compose up --build
```

Container reads logs from mounted `/logs`.
