package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"zomboid-log-crawler/internal/store"
	"zomboid-log-crawler/internal/watcher"
)

type Handler struct {
	store   *store.Store
	watcher *watcher.Service
	loc     *time.Location
}

type errorResponse struct {
	Error string `json:"error"`
}

type logsResponse struct {
	LogType string      `json:"log_type"`
	Total   int         `json:"total"`
	Count   int         `json:"count"`
	Offset  int         `json:"offset"`
	Limit   int         `json:"limit"`
	Items   interface{} `json:"items"`
}

func NewHandler(logStore *store.Store, watchSvc *watcher.Service, loc *time.Location) *Handler {
	return &Handler{store: logStore, watcher: watchSvc, loc: loc}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", h.health)
	r.Get("/api/log-types", h.logTypes)
	r.Get("/api/logs/{logType}", h.logsByType)
	return r
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) logTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": h.watcher.LogTypes()})
}

func (h *Handler) logsByType(w http.ResponseWriter, r *http.Request) {
	logType := chi.URLParam(r, "logType")
	if strings.TrimSpace(logType) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "logType is required"})
		return
	}

	query := r.URL.Query()
	limit := parseInt(query.Get("limit"), 100)
	if limit > 1000 {
		limit = 1000
	}
	offset := parseInt(query.Get("offset"), 0)

	from, err := parseTimeParam(query.Get("from"), h.loc)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid from time"})
		return
	}
	to, err := parseTimeParam(query.Get("to"), h.loc)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid to time"})
		return
	}
	if from != nil && to == nil {
		now := time.Now().In(h.loc)
		to = &now
	}

	items, total, exists := h.store.Query(logType, store.QueryOptions{
		Q:      query.Get("q"),
		From:   from,
		To:     to,
		Limit:  limit,
		Offset: offset,
	})
	if !exists {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "logType not found"})
		return
	}

	writeJSON(w, http.StatusOK, logsResponse{
		LogType: logType,
		Total:   total,
		Count:   len(items),
		Offset:  offset,
		Limit:   limit,
		Items:   items,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func parseInt(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func parseTimeParam(v string, loc *time.Location) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "02-01-06 15:04:05.000"} {
		if ts, err := time.ParseInLocation(layout, v, loc); err == nil {
			out := ts.In(loc)
			return &out, nil
		}
	}
	return nil, strconv.ErrSyntax
}
