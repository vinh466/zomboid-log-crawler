package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"zomboid-log-crawler/internal/parser"
)

type FileMeta struct {
	Path          string
	LogType       string
	FileTimestamp time.Time
}

func ScanLatestByType(logDir string, loc *time.Location) (map[string]FileMeta, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("read log dir: %w", err)
	}

	latest := make(map[string]FileMeta)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		parsed, err := parser.ParseFileName(entry.Name(), loc)
		if err != nil {
			continue
		}

		fullPath := filepath.Join(logDir, entry.Name())
		meta := FileMeta{
			Path:          fullPath,
			LogType:       parsed.LogType,
			FileTimestamp: parsed.FileTimestamp,
		}

		existing, ok := latest[meta.LogType]
		if !ok || meta.FileTimestamp.After(existing.FileTimestamp) {
			latest[meta.LogType] = meta
		}
	}

	return latest, nil
}
