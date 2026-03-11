package model

import "time"

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	RawLine   string    `json:"raw_line"`
}

type LogTypeInfo struct {
	LogType        string    `json:"log_type"`
	FilePath       string    `json:"file_path"`
	FileTimestamp  time.Time `json:"file_timestamp"`
	LastReadOffset int64     `json:"last_read_offset"`
	EntryCount     int       `json:"entry_count"`
}
