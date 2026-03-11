package parser

import (
	"fmt"
	"regexp"
	"time"
)

var linePattern = regexp.MustCompile(`^\[(\d{2}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\]\s?(.*)$`)

const lineLayout = "02-01-06 15:04:05.000"

type LineInfo struct {
	Timestamp time.Time
	Message   string
}

func ParseLogLine(line string, loc *time.Location) (LineInfo, error) {
	matches := linePattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return LineInfo{}, fmt.Errorf("invalid log line format")
	}

	ts, err := time.ParseInLocation(lineLayout, matches[1], loc)
	if err != nil {
		return LineInfo{}, fmt.Errorf("parse line timestamp: %w", err)
	}

	return LineInfo{Timestamp: ts, Message: matches[2]}, nil
}
