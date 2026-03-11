package watcher

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"zomboid-log-crawler/internal/discovery"
	"zomboid-log-crawler/internal/model"
	"zomboid-log-crawler/internal/parser"
	"zomboid-log-crawler/internal/store"
)

type EntryProcessor interface {
	Process(ctx context.Context, logType string, entry model.LogEntry)
}

type Service struct {
	logDir    string
	loc       *time.Location
	scanEvery time.Duration
	store     *store.Store
	processor EntryProcessor

	watcher *fsnotify.Watcher

	mu      sync.RWMutex
	latest  map[string]discovery.FileMeta
	offsets map[string]int64
}

func NewService(logDir string, loc *time.Location, scanEvery time.Duration, logStore *store.Store, processor EntryProcessor) (*Service, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Service{
		logDir:    logDir,
		loc:       loc,
		scanEvery: scanEvery,
		store:     logStore,
		processor: processor,
		watcher:   w,
		latest:    make(map[string]discovery.FileMeta),
		offsets:   make(map[string]int64),
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	if err := s.watcher.Add(s.logDir); err != nil {
		return err
	}

	if err := s.refreshLatest(ctx, true); err != nil {
		return err
	}

	go s.runEventLoop(ctx)
	go s.runTicker(ctx)
	return nil
}

func (s *Service) runEventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			_ = s.watcher.Close()
			return
		case err := <-s.watcher.Errors:
			if err != nil {
				log.Printf("watcher error: %v", err)
			}
		case event := <-s.watcher.Events:
			s.handleEvent(ctx, event)
		}
	}
}

func (s *Service) runTicker(ctx context.Context) {
	ticker := time.NewTicker(s.scanEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.refreshLatest(ctx, false); err != nil {
				log.Printf("scan error: %v", err)
			}
		}
	}
}

func (s *Service) handleEvent(ctx context.Context, event fsnotify.Event) {
	if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
		if err := s.refreshLatest(ctx, false); err != nil {
			log.Printf("refresh on create/rename: %v", err)
		}
		return
	}
	if event.Op&fsnotify.Write == 0 {
		return
	}

	logType, ok := s.logTypeByPath(event.Name)
	if !ok {
		return
	}
	if err := s.readNewLines(ctx, logType, event.Name, true); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("read new lines error: %v", err)
		}
	}
}

func (s *Service) refreshLatest(ctx context.Context, initial bool) error {
	latest, err := discovery.ScanLatestByType(s.logDir, s.loc)
	if err != nil {
		return err
	}

	s.removeMissingLogTypes(latest)

	for logType, meta := range latest {
		s.store.EnsureType(logType)
		current, exists := s.currentMeta(logType)
		if !exists || meta.FileTimestamp.After(current.FileTimestamp) {
			s.setMeta(logType, meta)
			if !initial {
				s.setOffset(meta.Path, 0)
			}
			if err := s.readNewLines(ctx, logType, meta.Path, !initial); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("read tracked file error: %v", err)
			}
		}
	}

	return nil
}

func (s *Service) removeMissingLogTypes(latest map[string]discovery.FileMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for logType, meta := range s.latest {
		if _, ok := latest[logType]; ok {
			continue
		}
		delete(s.latest, logType)
		delete(s.offsets, meta.Path)
	}
}

func (s *Service) readNewLines(ctx context.Context, logType, path string, notify bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	offset := s.getOffset(path)
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	newEntries := make([]model.LogEntry, 0)
	var bytesRead int64
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			bytesRead += int64(len(line))
			trimmed := strings.TrimRight(line, "\r\n")
			parsed, parseErr := parser.ParseLogLine(trimmed, s.loc)
			if parseErr == nil {
				newEntries = append(newEntries, model.LogEntry{
					Timestamp: parsed.Timestamp,
					Message:   parsed.Message,
					RawLine:   trimmed,
				})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	s.setOffset(path, offset+bytesRead)
	s.store.Append(logType, newEntries)
	if notify && s.processor != nil {
		for _, entry := range newEntries {
			s.processor.Process(ctx, logType, entry)
		}
	}

	return nil
}

func (s *Service) LogTypes() []model.LogTypeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.LogTypeInfo, 0, len(s.latest))
	for logType, meta := range s.latest {
		result = append(result, model.LogTypeInfo{
			LogType:        logType,
			FilePath:       meta.Path,
			FileTimestamp:  meta.FileTimestamp,
			LastReadOffset: s.offsets[meta.Path],
			EntryCount:     s.store.Count(logType),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LogType < result[j].LogType
	})
	return result
}

func (s *Service) logTypeByPath(path string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for logType, meta := range s.latest {
		if strings.EqualFold(meta.Path, path) {
			return logType, true
		}
	}
	return "", false
}

func (s *Service) currentMeta(logType string) (discovery.FileMeta, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	meta, ok := s.latest[logType]
	return meta, ok
}

func (s *Service) setMeta(logType string, meta discovery.FileMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest[logType] = meta
	if _, ok := s.offsets[meta.Path]; !ok {
		s.offsets[meta.Path] = 0
	}
}

func (s *Service) getOffset(path string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.offsets[path]
}

func (s *Service) setOffset(path string, offset int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offsets[path] = offset
}
