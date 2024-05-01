package main

import (
	"context"
	stdlog "log"
	"os"
	"os/signal"

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

	bot, err := telegram.New(log, cfg.Telegram, cfg.Mongo)
	if err != nil {
		log.Panic(errors.WrapFail(err, "initialize bot service"))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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
