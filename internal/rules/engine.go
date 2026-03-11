package rules

import (
	"context"
	"log"

	"zomboid-log-crawler/internal/model"
	"zomboid-log-crawler/internal/notifier"
)

type Rule interface {
	Name() string
	Apply(logType string, entry model.LogEntry) (*notifier.Message, bool)
}

type Engine struct {
	notifier *notifier.DiscordNotifier
	rules    []Rule
}

func NewEngine(discordNotifier *notifier.DiscordNotifier, rules []Rule) *Engine {
	return &Engine{
		notifier: discordNotifier,
		rules:    rules,
	}
}

func (e *Engine) Process(ctx context.Context, logType string, entry model.LogEntry) {
	matched := false
	for _, rule := range e.rules {
		msg, ok := rule.Apply(logType, entry)
		if !ok || msg == nil {
			continue
		}
		matched = true
		if err := e.notifier.Send(ctx, *msg); err != nil {
			log.Printf("rule %s notify error: %v", rule.Name(), err)
		}
	}

	if matched {
		return
	}
	if err := e.notifier.NotifyIfMatch(ctx, logType, entry); err != nil {
		log.Printf("keyword notify error: %v", err)
	}
}
