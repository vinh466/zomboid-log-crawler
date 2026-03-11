package store

import (
	"sort"
	"strings"
	"sync"
	"time"

	"zomboid-log-crawler/internal/model"
)

type QueryOptions struct {
	Q      string
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

type Store struct {
	mu      sync.RWMutex
	entries map[string][]model.LogEntry
}

func New() *Store {
	return &Store{entries: make(map[string][]model.LogEntry)}
}

func (s *Store) EnsureType(logType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[logType]; !ok {
		s.entries[logType] = make([]model.LogEntry, 0)
	}
}

func (s *Store) Append(logType string, newEntries []model.LogEntry) {
	if len(newEntries) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[logType] = append(s.entries[logType], newEntries...)
}

func (s *Store) Count(logType string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries[logType])
}

func (s *Store) Query(logType string, opts QueryOptions) ([]model.LogEntry, int, bool) {
	s.mu.RLock()
	entries, ok := s.entries[logType]
	if !ok {
		s.mu.RUnlock()
		return nil, 0, false
	}
	copied := make([]model.LogEntry, len(entries))
	copy(copied, entries)
	s.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(opts.Q))
	filtered := make([]model.LogEntry, 0, len(copied))
	for _, entry := range copied {
		if q != "" && !strings.Contains(strings.ToLower(entry.Message), q) {
			continue
		}
		if opts.From != nil && entry.Timestamp.Before(*opts.From) {
			continue
		}
		if opts.To != nil && entry.Timestamp.After(*opts.To) {
			continue
		}
		filtered = append(filtered, entry)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.Before(filtered[j].Timestamp)
	})

	total := len(filtered)
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.Offset >= total {
		return []model.LogEntry{}, total, true
	}
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	end := opts.Offset + opts.Limit
	if end > total {
		end = total
	}

	return filtered[opts.Offset:end], total, true
}
