package parser

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"
)

var filenamePattern = regexp.MustCompile(`^(\d{2}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2})_([A-Za-z0-9_.-]+)\.txt$`)

const filenameLayout = "02-01-06_15-04-05"

type FileNameInfo struct {
	FileTimestamp time.Time
	LogType       string
}

func ParseFileName(name string, loc *time.Location) (FileNameInfo, error) {
	base := filepath.Base(name)
	matches := filenamePattern.FindStringSubmatch(base)
	if len(matches) != 3 {
		return FileNameInfo{}, fmt.Errorf("invalid filename format: %s", base)
	}

	timestamp, err := time.ParseInLocation(filenameLayout, matches[1], loc)
	if err != nil {
		return FileNameInfo{}, fmt.Errorf("parse filename timestamp: %w", err)
	}

	return FileNameInfo{FileTimestamp: timestamp, LogType: matches[2]}, nil
}
