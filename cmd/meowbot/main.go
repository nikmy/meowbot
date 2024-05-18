package main

import (
	"context"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikmy/meowbot/internal/repo"
	"github.com/nikmy/meowbot/internal/telegram"
	"github.com/nikmy/meowbot/pkg/errors"
	"github.com/nikmy/meowbot/pkg/logger"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		stdlog.Panic(errors.WrapFail(err, "load config"))
	}

	log, err := logger.New(cfg.Environment)
	if err != nil {
		log.Panic(errors.WrapFail(err, "init logger"))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)
	defer cancel()

	db := cfg.Database

	repoClient, err := repo.NewMongoClient(ctx, db.Mongo, db.Sources.Interviews, db.Sources.Users)
	if err != nil {
		log.Panic(errors.WrapFail(err, "init repo client"))
	}

	bot, err := telegram.New(log, cfg.Telegram, repoClient)
	if err != nil {
		log.Panic(errors.WrapFail(err, "initialize bot service"))
	}

	stopped := make(chan struct{})
	context.AfterFunc(ctx, func() {
		stdlog.Println("Graceful shutdown...")
		bot.Stop()
		stopped <- struct{}{}
	})

	err = bot.Run(ctx)
	if err != nil {
		log.Panic(err)
	}
	stdlog.Println("Bot has been started")

	<-stopped
	stdlog.Println("Shutdown complete")
}
