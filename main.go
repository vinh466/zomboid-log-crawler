package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"zomboid-log-crawler/internal/api"
	"zomboid-log-crawler/internal/config"
	"zomboid-log-crawler/internal/customrules"
	"zomboid-log-crawler/internal/notifier"
	"zomboid-log-crawler/internal/rules"
	"zomboid-log-crawler/internal/store"
	"zomboid-log-crawler/internal/watcher"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Fatalf("load timezone: %v", err)
	}

	logStore := store.New()
	discordNotifier := notifier.NewDiscordNotifier(cfg.Discord.WebhookURL, cfg.Discord.Keywords)
	ruleSet, err := customrules.Build(cfg.Discord)
	if err != nil {
		log.Fatalf("build custom rules: %v", err)
	}
	ruleEngine := rules.NewEngine(discordNotifier, ruleSet)

	watchSvc, err := watcher.NewService(cfg.LogDir, loc, cfg.ScanEvery(), logStore, ruleEngine)
	if err != nil {
		log.Fatalf("create watcher: %v", err)
	}

	handler := api.NewHandler(logStore, watchSvc, loc)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.API.Port),
		Handler:      handler.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := watchSvc.Start(ctx); err != nil {
		log.Fatalf("start watcher: %v", err)
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown error: %v", err)
		}
	}()

	log.Printf("server listening on %s, log_dir=%s", srv.Addr, cfg.LogDir)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http server error: %v", err)
	}
}
