package main

import (
	"context"
	"github.com/nikmy/meowbot/internal/interviews"
	"github.com/nikmy/meowbot/internal/users"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

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

	interviewsRepo, err := interviews.New(ctx, log, db.Storage, db.Sources.Interviews)
	if err != nil {
		log.Panic(errors.WrapFail(err, "init interviews repo"))
	}

	usersRepo, err := users.New(ctx, log, db.Storage, db.Sources.Users)
	if err != nil {
		log.Panic(errors.WrapFail(err, "init users repo"))
	}

	bot, err := telegram.New(log, cfg.Telegram, interviewsRepo, usersRepo)
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
