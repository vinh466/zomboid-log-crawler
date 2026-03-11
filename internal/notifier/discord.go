package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"zomboid-log-crawler/internal/model"
)

type DiscordNotifier struct {
	webhookURL string
	keywords   []string
	client     *http.Client

	mu       sync.Mutex
	dedup    map[string]time.Time
	dedupTTL time.Duration
}

type Embed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
}

type Message struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

func NewDiscordNotifier(webhookURL string, keywords []string) *DiscordNotifier {
	normalized := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword == "" {
			continue
		}
		normalized = append(normalized, keyword)
	}

	return &DiscordNotifier{
		webhookURL: webhookURL,
		keywords:   normalized,
		client:     &http.Client{Timeout: 8 * time.Second},
		dedup:      make(map[string]time.Time),
		dedupTTL:   2 * time.Minute,
	}
}

func (n *DiscordNotifier) NotifyIfMatch(ctx context.Context, logType string, entry model.LogEntry) error {
	if n.webhookURL == "" || len(n.keywords) == 0 {
		return nil
	}

	msg := strings.ToLower(entry.Message)
	matched := false
	for _, keyword := range n.keywords {
		if strings.Contains(msg, keyword) {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}

	return n.Send(ctx, Message{
		Content: fmt.Sprintf("[%s] [%s] %s", entry.Timestamp.Format(time.RFC3339), logType, entry.Message),
	})
}

func (n *DiscordNotifier) Send(ctx context.Context, msg Message) error {
	if n.webhookURL == "" {
		return nil
	}
	if msg.Content == "" && len(msg.Embeds) == 0 {
		return nil
	}

	key := dedupKey(msg)
	if !n.markAndCheck(key) {
		return nil
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	var lastErr error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webhookURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := n.client.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("discord webhook status: %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(i+1) * 500 * time.Millisecond):
		}
	}

	return lastErr
}

func (n *DiscordNotifier) markAndCheck(key string) bool {
	now := time.Now()

	n.mu.Lock()
	defer n.mu.Unlock()
	for k, t := range n.dedup {
		if now.Sub(t) > n.dedupTTL {
			delete(n.dedup, k)
		}
	}
	if _, exists := n.dedup[key]; exists {
		return false
	}
	n.dedup[key] = now
	return true
}

func dedupKey(msg Message) string {
	b := strings.Builder{}
	b.WriteString(msg.Content)
	for _, embed := range msg.Embeds {
		b.WriteString("|")
		b.WriteString(embed.Title)
		b.WriteString("|")
		b.WriteString(embed.Description)
		b.WriteString("|")
		b.WriteString(fmt.Sprintf("%d", embed.Color))
	}
	return b.String()
}
